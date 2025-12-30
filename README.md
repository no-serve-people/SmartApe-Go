# SmartApe-Go 🦍

[中文](#中文) | [English](#english)

**SmartApe-Go** 是一个用 Go 语言编写的高性能 Polymarket 预测市场套利机器人。它实现了 @the_smart_ape 分享的 "Dump & Hedge"（暴跌对冲）策略，旨在捕捉市场剧烈波动时的无风险套利机会。

**SmartApe-Go** is a high-performance Polymarket prediction market arbitrage bot written in Go. It implements the "Dump & Hedge" strategy shared by @the_smart_ape, designed to capture risk-free arbitrage opportunities during high market volatility.

---

## 中文

### 核心特性 (Features)

*   **⚡ 毫秒级暴跌检测 (Leg 1)**: 利用环形缓冲区（Ring Buffer）实时监控价格，当检测到短时间内的剧烈下跌（例如：3 秒内跌幅 > 15%）时，机器人判定为非理性抛售并自动买入。
*   **🛡️ 自动无风险对冲 (Leg 2)**: 买入第一腿后，机器人持续监控反向结果（Opposite Outcome）的价格。当满足 `Leg1 成本 + Leg2 价格 < 1.0` (即存在确定性利润) 时，自动买入对冲，锁定收益。
*   **🔐 生产级交易执行**: 完整实现了 Polymarket CLOB API 的 **EIP-712** 签名和订单构建，支持 Go-Ethereum 原生签名，无需依赖复杂的外部库。
*   **🧪 模拟回测模式**: 内置 Mock 交易所，可在不消耗真实资金的情况下测试策略逻辑和参数敏感度。

### 快速开始 (Quick Start)

#### 1. 安装
确保已安装 Go 1.21+。
```bash
git clone https://github.com/yourusername/smartape-go.git
cd smartape-go
go mod download
```

#### 2. 运行模拟回测
默认配置下运行 `main.go` 将启动模拟演示，展示机器人在价格暴跌时的反应：
```bash
go run main.go
```

预期输出：
```text
启动 Polymarket Smart Ape 策略机器人 (模拟模式)...
...
DETECTED DUMP on UP! Drop: 40.00%
>>> EXECUTING LEG 1: Buy UP @ 0.300
...
HEDGE CONDITION MET! Sum: 0.950 <= Target: 0.950
>>> EXECUTING LEG 2 (HEDGE): Buy DOWN @ 0.650
CYCLE COMPLETE. ROI: 5.26%
```

#### 3. 实盘配置
要切换到实盘交易，请在 `main.go` 中初始化真实的 `PolymarketClient` 并替换 `MockExchange`。

你需要准备：
*   Polymarket (Polygon) 私钥
*   Polymarket API Keys (可在官网申请)
*   Funder Address (通常是你的钱包地址或 Proxy 地址)

```go
// 在 main.go 中修改
import "poly/pkg/exchange"

// ...

realClient, err := exchange.NewPolymarketClient(
    "YOUR_API_KEY",
    "YOUR_API_SECRET",
    "YOUR_PASSPHRASE",
    "YOUR_PRIVATE_KEY_HEX", // 0x...
    "YOUR_FUNDER_ADDRESS",  // 0x...
)

if err != nil {
    log.Fatal(err)
}

// 使用 realClient 启动机器人
bot := strategy.NewBot(cfg, realClient)
```

### 策略参数
可在 `pkg/config/config.go` 中调整：
*   `MovePct`: 暴跌判定阈值 (默认 0.15 即 15%)
*   `WindowMin`: 监控窗口时间 (默认 2 分钟)
*   `SumTarget`: 对冲总成本目标 (默认 0.95 USDC)
*   `Shares`: 单次交易手数

### 免责声明
本项目仅供教育和研究目的。加密货币交易和预测市场存在高风险，代码可能包含未发现的 bug。使用者需自行承担资金损失的风险。

---

## English

### Key Features

*   **⚡ Millisecond Dump Detection (Leg 1)**: Uses a high-performance Ring Buffer to monitor prices in real-time. Automatically executes a buy order when a sharp drop is detected within a short window (e.g., >15% drop in 3s), identifying irrational panic selling.
*   **🛡️ Auto Risk-Free Hedging (Leg 2)**: After executing Leg 1, the bot continuously monitors the price of the opposite outcome. When the condition `Leg1 Cost + Leg2 Price < 1.0` is met (guaranteeing profit), it automatically executes the hedge to lock in risk-free returns.
*   **🔐 Production-Ready Execution**: Fully implements **EIP-712** signing and order construction for the Polymarket CLOB API using native Go-Ethereum libraries.
*   **🧪 Backtesting Mode**: Built-in Mock Exchange allows you to test strategy logic and parameter sensitivity without risking real funds.

### Quick Start

#### 1. Installation
Ensure Go 1.21+ is installed.
```bash
git clone https://github.com/yourusername/smartape-go.git
cd smartape-go
go mod download
```

#### 2. Run Simulation
Running `main.go` with default settings will start a backtest simulation showing how the bot reacts to a price crash:
```bash
go run main.go
```

Expected Output:
```text
启动 Polymarket Smart Ape 策略机器人 (模拟模式)...
...
DETECTED DUMP on UP! Drop: 40.00%
>>> EXECUTING LEG 1: Buy UP @ 0.300
...
HEDGE CONDITION MET! Sum: 0.950 <= Target: 0.950
>>> EXECUTING LEG 2 (HEDGE): Buy DOWN @ 0.650
CYCLE COMPLETE. ROI: 5.26%
```

#### 3. Live Trading Configuration
To switch to live trading, initialize the real `PolymarketClient` in `main.go` and replace the `MockExchange`.

You will need:
*   Polymarket (Polygon) Private Key
*   Polymarket API Keys
*   Funder Address (Your wallet or Proxy address)

```go
// Modify in main.go
import "poly/pkg/exchange"

// ...

realClient, err := exchange.NewPolymarketClient(
    "YOUR_API_KEY",
    "YOUR_API_SECRET",
    "YOUR_PASSPHRASE",
    "YOUR_PRIVATE_KEY_HEX", // 0x...
    "YOUR_FUNDER_ADDRESS",  // 0x...
)

if err != nil {
    log.Fatal(err)
}

// Start bot with realClient
bot := strategy.NewBot(cfg, realClient)
```

### Strategy Parameters
Adjustable in `pkg/config/config.go`:
*   `MovePct`: Dump threshold (Default 0.15 for 15%)
*   `WindowMin`: Monitoring time window (Default 2 minutes)
*   `SumTarget`: Target total cost for hedging (Default 0.95 USDC)
*   `Shares`: Position size per trade

### Disclaimer
This project is for educational and research purposes only. Cryptocurrency trading and prediction markets involve high risks. The code may contain undiscovered bugs. Use at your own risk.
