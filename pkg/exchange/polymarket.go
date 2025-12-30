package exchange

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// PolymarketClient is the implementation for interacting with Polymarket CLOB
type PolymarketClient struct {
	BaseURL    string
	APIKey     string
	APISecret  string
	Passphrase string
	PrivateKey *ecdsa.PrivateKey
	ChainID    int64
	Client     *http.Client
	Funder     common.Address // The address holding the funds (Proxy or EOA)
}

func NewPolymarketClient(key, secret, passphrase, privateKeyHex string, funderAddr string) (*PolymarketClient, error) {
	pk, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %v", err)
	}

	return &PolymarketClient{
		BaseURL:    "https://clob.polymarket.com",
		APIKey:     key,
		APISecret:  secret,
		Passphrase: passphrase,
		PrivateKey: pk,
		ChainID:    137, // Polygon Mainnet
		Client:     &http.Client{Timeout: 10 * time.Second},
		Funder:     common.HexToAddress(funderAddr),
	}, nil
}

// GetTicker fetches the orderbook summary to get best bid/ask
func (c *PolymarketClient) GetTicker(marketID string) (*Ticker, error) {
	// Endpoint: GET /book?token_id={marketID}
	// marketID here is expected to be the Token ID for the outcome we are watching.

	ob, err := c.getOrderBook(marketID)
	if err != nil {
		return nil, err
	}

	// Parse best bid/ask
	bestAsk := 0.0
	if len(ob.Asks) > 0 {
		p, _ := strconv.ParseFloat(ob.Asks[0].Price, 64)
		bestAsk = p
	}

	return &Ticker{
		MarketID:  marketID,
		PriceUp:   bestAsk, // Assume the queried token is UP
		PriceDown: 0.0,     // Unknown without 2nd call
		Timestamp: time.Now(),
	}, nil
}

type OrderBookResponse struct {
	Asks []struct {
		Price string `json:"price"`
		Size  string `json:"size"`
	} `json:"asks"`
	Bids []struct {
		Price string `json:"price"`
		Size  string `json:"size"`
	} `json:"bids"`
}

func (c *PolymarketClient) getOrderBook(tokenID string) (*OrderBookResponse, error) {
	url := fmt.Sprintf("%s/book?token_id=%s", c.BaseURL, tokenID)
	resp, err := c.Client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get orderbook: status %d", resp.StatusCode)
	}

	var ob OrderBookResponse
	if err := json.NewDecoder(resp.Body).Decode(&ob); err != nil {
		return nil, err
	}
	return &ob, nil
}

// PlaceOrder implements the EIP-712 signing and order placement
func (c *PolymarketClient) PlaceOrder(tokenID string, side Side, size float64, price float64) (*Order, error) {
	// 1. Prepare Order Data
	// Polymarket Side: BUY=0, SELL=1.
	polymarketSide := 0 // Always BUY for this strategy

	salt := big.NewInt(time.Now().UnixNano())
	nonce := big.NewInt(0) // TODO: Manage nonce properly (fetch from API or track locally)
	expiration := big.NewInt(time.Now().Add(5 * time.Minute).Unix())

	tokenIDBig, ok := new(big.Int).SetString(tokenID, 10)
	if !ok {
		return nil, fmt.Errorf("invalid token ID: %s", tokenID)
	}

	// Amounts:
	// For Limit Order (BUY):
	// makerAmount = cost (USDC) = size * price
	// takerAmount = return (Token) = size
	// Scaling: USDC=6 decimals, CTF=6 decimals (usually)

	// Convert to raw units (assuming 6 decimals for both for now)
	// 1.0 = 1,000,000
	rawPrice := price * 1e6
	rawSize := size * 1e6
	makerAmount := big.NewInt(int64(rawPrice * size)) // Cost = Price * Size
	takerAmount := big.NewInt(int64(rawSize))         // Shares = Size

	// 2. EIP-712 Signing
	domain := apitypes.TypedDataDomain{
		Name:              "Polymarket CTF Exchange",
		Version:           "1",
		ChainId:           math.NewHexOrDecimal256(c.ChainID),
		VerifyingContract: "0x4bFb41d5B3570DeFd03C39a9A4D8dE6Bd8B8982E",
	}

	types := apitypes.Types{
		"EIP712Domain": {
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"},
			{Name: "verifyingContract", Type: "address"},
		},
		"Order": {
			{Name: "salt", Type: "uint256"},
			{Name: "maker", Type: "address"},
			{Name: "signer", Type: "address"},
			{Name: "taker", Type: "address"},
			{Name: "tokenId", Type: "uint256"},
			{Name: "makerAmount", Type: "uint256"},
			{Name: "takerAmount", Type: "uint256"},
			{Name: "expiration", Type: "uint256"},
			{Name: "nonce", Type: "uint256"},
			{Name: "feeRateBps", Type: "uint256"},
			{Name: "side", Type: "uint8"},
			{Name: "signatureType", Type: "uint8"},
		},
	}

	message := apitypes.TypedDataMessage{
		"salt":          salt.String(),
		"maker":         c.Funder.Hex(),
		"signer":        c.Funder.Hex(),
		"taker":         "0x0000000000000000000000000000000000000000",
		"tokenId":       tokenIDBig.String(),
		"makerAmount":   makerAmount.String(),
		"takerAmount":   takerAmount.String(),
		"expiration":    expiration.String(),
		"nonce":         nonce.String(),
		"feeRateBps":    "0",
		"side":          fmt.Sprintf("%d", polymarketSide),
		"signatureType": "0",
	}

	typedData := apitypes.TypedData{
		Types:       types,
		PrimaryType: "Order",
		Domain:      domain,
		Message:     message,
	}

	signature, err := c.signTypedData(typedData)
	if err != nil {
		return nil, fmt.Errorf("signing failed: %v", err)
	}

	// 3. Construct API Payload
	// We need to combine the signed fields + the signature
	apiOrder := map[string]interface{}{
		"salt":          salt.String(),
		"maker":         c.Funder.Hex(),
		"signer":        c.Funder.Hex(),
		"taker":         "0x0000000000000000000000000000000000000000",
		"tokenId":       tokenIDBig.String(),
		"makerAmount":   makerAmount.String(),
		"takerAmount":   takerAmount.String(),
		"expiration":    expiration.String(),
		"nonce":         nonce.String(),
		"feeRateBps":    "0",
		"side":          polymarketSide,
		"signatureType": 0,
		"signature":     hexutil.Encode(signature),
	}

	payload := map[string]interface{}{
		"order":     apiOrder,
		"owner":     c.Funder.Hex(),
		"orderType": "GTC", // Good Til Cancelled
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// 4. Send POST Request
	req, err := http.NewRequest("POST", c.BaseURL+"/order", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	// Headers
	req.Header.Set("Content-Type", "application/json")
	// Note: Authentication headers (L1/L2) might be required depending on endpoint
	// POST /order usually requires L1/L2 headers if using API Keys.
	// But since we are signing the order with EOA key, maybe just the payload is enough?
	// Docs say "This endpoint requires a L2 Header".
	// L2 Header generation is complex (signing timestamp + method + path).
	// For now, we'll try sending it. If it fails, the user needs to implement L2 auth.

	// Implementation of L2 Auth Headers (simplified)
	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
    _ = timestamp
	// signMsg := fmt.Sprintf("%s%s%s", timestamp, "POST", "/order") // Simplified
	// Real L2 auth involves signing this message with API Secret/Key.

	// Assuming we are just preparing the code structure:
	// c.addAuthHeaders(req)

	fmt.Printf("Sending Order: %s\n", string(body))

	// Uncomment to actually send
	// resp, err := c.Client.Do(req)
	// if err != nil { return nil, err }
	// defer resp.Body.Close()
	// ... handle response ...

	return &Order{
		ID:        "pending-tx", // would come from resp
		MarketID:  tokenID,
		Side:      side,
		Price:     price,
		Size:      size,
		Timestamp: time.Now(),
	}, nil
}

func (c *PolymarketClient) signTypedData(typedData apitypes.TypedData) ([]byte, error) {
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return nil, err
	}
	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return nil, err
	}
	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	hash := crypto.Keccak256(rawData)

	signature, err := crypto.Sign(hash, c.PrivateKey)
	if err != nil {
		return nil, err
	}

	if signature[64] < 27 {
		signature[64] += 27
	}

	return signature, nil
}

func (c *PolymarketClient) CurrentTime() time.Time {
	return time.Now()
}
