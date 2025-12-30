package main

import (
	"fmt"
	"time"

	"poly/pkg/config"
	"poly/pkg/exchange"
	"poly/pkg/strategy"
)

func main() {
	fmt.Println("启动 Polymarket Smart Ape 策略机器人 (模拟模式)...")

	// 1. 初始化配置
	cfg := config.DefaultConfig()
	cfg.WindowMin = 5 * time.Minute
	cfg.MovePct = 0.15 // 15% drop

	// 2. 初始化模拟交易所
	mockExc := exchange.NewMockExchange()
	
	// 3. 初始化机器人
	bot := strategy.NewBot(cfg, mockExc)

	// 4. 运行模拟循环
	// 场景：市场开始平稳，突然 UP 价格暴跌，触发 Leg 1，然后价格稳定，触发 Leg 2
	
	fmt.Println(">>> 模拟开始: 初始价格 UP: 0.50, DOWN: 0.50")
	mockExc.SetPrice(0.50, 0.50)

	// 前 10 秒平稳
	for i := 0; i < 10; i++ {
		mockExc.AdvanceTime(1 * time.Second)
		bot.RunTick()
	}

	// 触发暴跌：UP 从 0.50 -> 0.30 (跌幅 40% > 15%)
	fmt.Println("\n>>> 模拟暴跌事件! UP 0.50 -> 0.30")
	mockExc.AdvanceTime(1 * time.Second)
	mockExc.SetPrice(0.30, 0.55) // DOWN 稍微上涨但有滞后或价差
	bot.RunTick() // 这里应该触发 Leg 1 买入 UP

	// 此时 Leg 1 买入 UP @ 0.30
	// 此时 DOWN 价格 0.55
	// Sum = 0.30 + 0.55 = 0.85
	// Target = 0.95
	// 0.85 < 0.95，所以应该立即触发 Leg 2?
	// 让我们看看 Bot 的逻辑。是的，如果条件立即满足，会立即对冲。
	// 如果不满足，我们需要模拟价格恢复。

	// 假设暴跌时 DOWN 涨得很快，导致 Sum > 0.95
	// 例如 UP 0.30, DOWN 0.75 (Sum 1.05) -> 不买 Leg 2
	
	fmt.Println("\n>>> 模拟暴跌场景 2: DOWN 价格飙升导致无法立即对冲")
	// 重置
	bot.ResetCycle()
	mockExc.SetPrice(0.50, 0.50)
	// 填充历史 buffer
	for i := 0; i < 5; i++ {
		mockExc.AdvanceTime(1 * time.Second)
		bot.RunTick()
	}
	
	fmt.Println(">>> 暴跌发生...")
	mockExc.AdvanceTime(1 * time.Second)
	mockExc.SetPrice(0.30, 0.75) // Sum = 1.05
	bot.RunTick() // 触发 Leg 1

	// 此时持有 UP @ 0.30
	// 等待 DOWN 价格回落
	fmt.Println("\n>>> 等待对冲机会...")
	steps := 0
	for {
		steps++
		mockExc.AdvanceTime(1 * time.Second)
		
		// 模拟 DOWN 价格缓慢下降
		currentDown := mockExc.CurrentTicker.PriceDown
		if currentDown > 0.60 {
			mockExc.SetPrice(0.30, currentDown - 0.02)
		}
		
		fmt.Printf("Tick %d: UP=%.2f, DOWN=%.2f\n", steps, mockExc.CurrentTicker.PriceUp, mockExc.CurrentTicker.PriceDown)
		bot.RunTick()
		
		// 如果我们完成了，就退出
		// (在真实代码中可以通过检查 Bot 状态，这里简单跑几步)
		if steps > 10 {
			break
		}
	}
}
