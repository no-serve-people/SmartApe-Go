package exchange

import (
	"errors"
	"math/rand"
	"time"
)

// MockExchange simulates a market for testing/backtesting
type MockExchange struct {
	CurrentTicker *Ticker
	Time          time.Time
}

func NewMockExchange() *MockExchange {
	return &MockExchange{
		Time: time.Now(),
		CurrentTicker: &Ticker{
			MarketID:  "mock-market",
			PriceUp:   0.50,
			PriceDown: 0.50,
		},
	}
}

func (m *MockExchange) GetTicker(marketID string) (*Ticker, error) {
	m.CurrentTicker.Timestamp = m.Time
	return m.CurrentTicker, nil
}

func (m *MockExchange) PlaceOrder(marketID string, side Side, size float64, price float64) (*Order, error) {
	// Simulate immediate fill at requested price
	if size <= 0 {
		return nil, errors.New("invalid size")
	}

	return &Order{
		ID:        "mock-order-id",
		MarketID:  marketID,
		Side:      side,
		Price:     price,
		Size:      size,
		Timestamp: m.Time,
	}, nil
}

func (m *MockExchange) CurrentTime() time.Time {
	return m.Time
}

// Helpers to manipulate simulation

func (m *MockExchange) SetPrice(up, down float64) {
	m.CurrentTicker.PriceUp = up
	m.CurrentTicker.PriceDown = down
}

func (m *MockExchange) AdvanceTime(d time.Duration) {
	m.Time = m.Time.Add(d)
}

func (m *MockExchange) SimulateDump(side Side, from, to float64, duration time.Duration) {
	// Simple linear interpolation or instant drop?
	// Let's just set it instantly for unit tests
	if side == SideUp {
		m.SetPrice(to, m.CurrentTicker.PriceDown)
	} else {
		m.SetPrice(m.CurrentTicker.PriceUp, to)
	}
}

// RandomWalk simulates price movement
func (m *MockExchange) RandomWalk() {
	change := (rand.Float64() - 0.5) * 0.02 // +/- 1%
	newUp := m.CurrentTicker.PriceUp + change
	if newUp < 0.01 {
		newUp = 0.01
	}
	if newUp > 0.99 {
		newUp = 0.99
	}

	m.SetPrice(newUp, 1.0-newUp-0.01) // Maintain ~1.0 sum with slight spread
}
