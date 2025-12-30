package strategy

import (
	"log"
	"time"

	"poly/pkg/config"
	"poly/pkg/exchange"
	"poly/pkg/market"
)

// State represents the bot's current state in the cycle
type State int

const (
	StateWatching State = iota
	StateLeg1Bought
	StateDone
)

type Bot struct {
	cfg      *config.Config
	exchange exchange.Exchange
	state    State

	// Market Data
	bufferUp   *market.PriceBuffer
	bufferDown *market.PriceBuffer

	// Cycle State
	leg1Side       exchange.Side
	leg1EntryPrice float64
	roundStartTime time.Time
}

func NewBot(cfg *config.Config, exc exchange.Exchange) *Bot {
	return &Bot{
		cfg:            cfg,
		exchange:       exc,
		state:          StateWatching,
		bufferUp:       market.NewPriceBuffer(5 * time.Second), // Keep 5s history
		bufferDown:     market.NewPriceBuffer(5 * time.Second),
		roundStartTime: exc.CurrentTime(), // Assume round starts when bot starts for simplicity, or fetch from API
	}
}

// ResetCycle resets the bot for a new round
func (b *Bot) ResetCycle() {
	log.Println("--- Resetting Cycle for New Round ---")
	b.state = StateWatching
	b.leg1Side = ""
	b.leg1EntryPrice = 0
	b.roundStartTime = b.exchange.CurrentTime()
	// Clear buffers? No, keep them for continuity or clear if different market
}

// RunTick executes one tick of logic
func (b *Bot) RunTick() {
	now := b.exchange.CurrentTime()
	ticker, err := b.exchange.GetTicker(b.cfg.MarketID)
	if err != nil {
		log.Printf("Error fetching ticker: %v", err)
		return
	}

	// Update Buffers
	b.bufferUp.Add(ticker.PriceUp, now)
	b.bufferDown.Add(ticker.PriceDown, now)

	// Logic Switch
	switch b.state {
	case StateWatching:
		b.checkLeg1(ticker, now)
	case StateLeg1Bought:
		b.checkLeg2(ticker)
	case StateDone:
		// Wait for next round (handled externally or by checking round ID change)
	}
}

func (b *Bot) checkLeg1(ticker *exchange.Ticker, now time.Time) {
	// Check window
	elapsed := now.Sub(b.roundStartTime)
	if elapsed > b.cfg.WindowMin {
		// Window closed, missed opportunity for this round?
		// Or keep watching but don't enter Leg 1?
		// The strategy says "only watches... during the first windowMin".
		// So if window passes, we effectively go to Done or just Idle until reset.
		// log.Printf("Window closed (elapsed %v > %v)", elapsed, b.cfg.WindowMin)
		return
	}

	// 1. Check UP Dump
	priceUp3sAgo := b.bufferUp.GetPriceAgo(3*time.Second, now)
	if priceUp3sAgo > 0 {
		drop := (priceUp3sAgo - ticker.PriceUp) / priceUp3sAgo
		if drop >= b.cfg.MovePct {
			log.Printf("DETECTED DUMP on UP! Drop: %.2f%% (%.3f -> %.3f)", drop*100, priceUp3sAgo, ticker.PriceUp)
			b.executeLeg1(exchange.SideUp, ticker.PriceUp)
			return
		}
	}

	// 2. Check DOWN Dump
	priceDown3sAgo := b.bufferDown.GetPriceAgo(3*time.Second, now)
	if priceDown3sAgo > 0 {
		drop := (priceDown3sAgo - ticker.PriceDown) / priceDown3sAgo
		if drop >= b.cfg.MovePct {
			log.Printf("DETECTED DUMP on DOWN! Drop: %.2f%% (%.3f -> %.3f)", drop*100, priceDown3sAgo, ticker.PriceDown)
			b.executeLeg1(exchange.SideDown, ticker.PriceDown)
			return
		}
	}
}

func (b *Bot) executeLeg1(side exchange.Side, price float64) {
	log.Printf(">>> EXECUTING LEG 1: Buy %s @ %.3f", side, price)

	order, err := b.exchange.PlaceOrder(b.cfg.MarketID, side, b.cfg.Shares, price)
	if err != nil {
		log.Printf("Failed to place Leg 1 order: %v", err)
		return
	}

	b.leg1Side = side
	b.leg1EntryPrice = order.Price // Use actual fill price
	b.state = StateLeg1Bought
	log.Printf("Leg 1 Filled. Waiting for Hedge (Target Sum <= %.2f)...", b.cfg.SumTarget)
}

func (b *Bot) checkLeg2(ticker *exchange.Ticker) {
	var oppositePrice float64
	var oppositeSide exchange.Side

	if b.leg1Side == exchange.SideUp {
		oppositePrice = ticker.PriceDown
		oppositeSide = exchange.SideDown
	} else {
		oppositePrice = ticker.PriceUp
		oppositeSide = exchange.SideUp
	}

	currentSum := b.leg1EntryPrice + oppositePrice

	// Strategy: leg1EntryPrice + oppositeAsk <= sumTarget
	if currentSum <= b.cfg.SumTarget {
		log.Printf("HEDGE CONDITION MET! Sum: %.3f (Entry: %.3f + Opp: %.3f) <= Target: %.3f",
			currentSum, b.leg1EntryPrice, oppositePrice, b.cfg.SumTarget)

		b.executeLeg2(oppositeSide, oppositePrice)
	}
}

func (b *Bot) executeLeg2(side exchange.Side, price float64) {
	log.Printf(">>> EXECUTING LEG 2 (HEDGE): Buy %s @ %.3f", side, price)

	order, err := b.exchange.PlaceOrder(b.cfg.MarketID, side, b.cfg.Shares, price)
	if err != nil {
		log.Printf("Failed to place Leg 2 order: %v", err)
		return
	}

	totalCost := b.leg1EntryPrice + order.Price
	profit := 1.0 - totalCost // Since we hold 1 share of YES and 1 share of NO, payout is $1.0
	roi := (profit / totalCost) * 100

	log.Printf("CYCLE COMPLETE. Total Cost: %.3f, Profit per share: %.3f, ROI: %.2f%%", totalCost, profit, roi)
	b.state = StateDone
}
