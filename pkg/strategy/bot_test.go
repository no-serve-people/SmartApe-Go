package strategy

import (
	"testing"
	"time"

	"poly/pkg/config"
	"poly/pkg/exchange"
)

func TestBotLogic(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MovePct = 0.10   // 10% drop trigger
	cfg.SumTarget = 0.96 // Slightly higher to avoid float precision issues at boundary

	mockExc := exchange.NewMockExchange()
	bot := NewBot(cfg, mockExc)

	// Initial State
	mockExc.SetPrice(0.50, 0.50)
	bot.RunTick() // Fill buffer

	// Advance 3 seconds
	mockExc.AdvanceTime(3 * time.Second)
	bot.RunTick()

	// Trigger Dump on UP: 0.50 -> 0.40 (20% drop > 10%)
	mockExc.AdvanceTime(1 * time.Second)
	mockExc.SetPrice(0.40, 0.55)
	bot.RunTick()

	if bot.state != StateLeg1Bought {
		t.Errorf("Expected state Leg1Bought, got %v", bot.state)
	}
	if bot.leg1Side != exchange.SideUp {
		t.Errorf("Expected Leg1 side UP, got %v", bot.leg1Side)
	}

	// Check Hedge Condition
	// Entry 0.40. Opp Price 0.55. Sum 0.95. Target 0.95.
	// Should execute hedge immediately.
	// Note: In the previous RunTick, it might have executed immediately if the order of checkLeg1 and checkLeg2 allows it.
	// Let's see Bot.RunTick logic:
	// switch b.state { case StateWatching: checkLeg1 ... if executed -> b.state = Leg1Bought }
	// It breaks after checkLeg1. So need another tick to check Leg2.

	bot.RunTick()

	if bot.state != StateDone {
		t.Errorf("Expected state Done, got %v", bot.state)
	}
}
