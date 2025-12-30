package market

import (
	"testing"
	"time"
)

func TestPriceBuffer(t *testing.T) {
	pb := NewPriceBuffer(5 * time.Second)
	now := time.Now()

	// Add data points
	// T=0, P=100
	pb.Add(100, now.Add(-5*time.Second))
	// T=2, P=102
	pb.Add(102, now.Add(-3*time.Second))
	// T=4, P=104
	pb.Add(104, now.Add(-1*time.Second))

	// Test GetPriceAgo

	// 3 seconds ago from now (T=5) is T=2. Should be 102.
	price := pb.GetPriceAgo(3*time.Second, now)
	if price != 102 {
		t.Errorf("Expected price 102, got %f", price)
	}

	// 5 seconds ago from now is T=0. Should be 100.
	price = pb.GetPriceAgo(5*time.Second, now)
	if price != 100 {
		t.Errorf("Expected price 100, got %f", price)
	}

	// 1 second ago is T=4. Should be 104.
	price = pb.GetPriceAgo(1*time.Second, now)
	if price != 104 {
		t.Errorf("Expected price 104, got %f", price)
	}
}
