package config

import (
	"time"
)

type Config struct {
	// Strategy Parameters
	Shares    float64       `json:"shares"`     // Position size (e.g. 20)
	SumTarget float64       `json:"sum_target"` // Hedge threshold (e.g. 0.95)
	MovePct   float64       `json:"move_pct"`   // Dump threshold (e.g. 0.15 for 15%)
	WindowMin time.Duration `json:"window_min"` // Time window for Leg 1 (e.g. 2 minutes)
	FeeRate   float64       `json:"fee_rate"`   // Fee rate for simulation (e.g. 0.001)

	// System
	MarketID     string        `json:"market_id"` // The Market ID to trade
	PollInterval time.Duration `json:"poll_interval"`
}

func DefaultConfig() *Config {
	return &Config{
		Shares:       20.0,
		SumTarget:    0.95,
		MovePct:      0.15,
		WindowMin:    2 * time.Minute,
		FeeRate:      0.0, // Polymarket rebate?
		PollInterval: 1 * time.Second,
	}
}
