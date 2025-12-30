package market

import (
	"sync"
	"time"
)

// PricePoint stores a price at a specific time
type PricePoint struct {
	Price     float64
	Timestamp time.Time
}

// PriceBuffer maintains a history of prices for dump detection
type PriceBuffer struct {
	mu      sync.RWMutex
	history []PricePoint
	window  time.Duration
}

func NewPriceBuffer(window time.Duration) *PriceBuffer {
	return &PriceBuffer{
		history: make([]PricePoint, 0),
		window:  window,
	}
}

// Add appends a new price point and removes old ones
func (pb *PriceBuffer) Add(price float64, ts time.Time) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	pb.history = append(pb.history, PricePoint{Price: price, Timestamp: ts})

	// Prune old data
	cutoff := ts.Add(-pb.window)
	validIdx := 0
	for i, p := range pb.history {
		if p.Timestamp.After(cutoff) {
			validIdx = i
			break
		}
	}
	if validIdx > 0 {
		pb.history = pb.history[validIdx:]
	}
}

// GetPriceAgo returns the price exactly `duration` ago (or closest approximation)
// Returns -1 if insufficient history
func (pb *PriceBuffer) GetPriceAgo(duration time.Duration, now time.Time) float64 {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	if len(pb.history) == 0 {
		return -1
	}

	targetTime := now.Add(-duration)

	// Find the closest point to targetTime
	// Since history is sorted by time, we can iterate or binary search.
	// For small buffers (3s @ 10 ticks/s = 30 items), linear scan is fine.

	var bestPrice float64 = -1
	minDiff := time.Hour // Large initial diff

	for _, p := range pb.history {
		diff := p.Timestamp.Sub(targetTime)
		if diff < 0 {
			diff = -diff
		}

		if diff < minDiff {
			minDiff = diff
			bestPrice = p.Price
		}
	}

	// If the best match is too far off (e.g. gap in data), we might want to return -1
	// For now, we accept if within 1 second tolerance
	if minDiff > 1*time.Second {
		return -1
	}

	return bestPrice
}
