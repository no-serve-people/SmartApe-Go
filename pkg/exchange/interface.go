package exchange

import "time"

// Side represents the outcome side (UP/DOWN or YES/NO)
type Side string

const (
	SideUp   Side = "UP"
	SideDown Side = "DOWN"
)

// Order represents a trade order
type Order struct {
	ID        string
	MarketID  string
	Side      Side
	Price     float64
	Size      float64
	Timestamp time.Time
}

// Ticker represents the current best prices
type Ticker struct {
	MarketID  string
	PriceUp   float64 // Best Ask for UP
	PriceDown float64 // Best Ask for DOWN
	Timestamp time.Time
}

// Exchange defines the interface for interacting with the market
type Exchange interface {
	// GetTicker returns the latest prices
	GetTicker(marketID string) (*Ticker, error)

	// PlaceOrder places a limit order (or market buy via limit)
	PlaceOrder(marketID string, side Side, size float64, price float64) (*Order, error)

	// CurrentTime returns the exchange time (useful for backtesting)
	CurrentTime() time.Time
}
