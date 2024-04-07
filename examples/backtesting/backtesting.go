package main

import (
	"context"

	"github.com/rodrigo-brito/ninjabot"
	"github.com/rodrigo-brito/ninjabot/examples/strategies"
	"github.com/rodrigo-brito/ninjabot/exchange"
	"github.com/rodrigo-brito/ninjabot/plot"
	"github.com/rodrigo-brito/ninjabot/plot/indicator"
	"github.com/rodrigo-brito/ninjabot/storage"
	"github.com/rodrigo-brito/ninjabot/tools/log"
)

// 在 NinjaBot 中使用回测功能。
// 回测是在历史数据（来自 CSV 文件）上对策略进行模拟的过程
// This example shows how to use backtesting with NinjaBot
// Backtesting is a simulation of the strategy in historical data (from CSV)
func main() {
	ctx := context.Background()

	// bot settings (eg: pairs, telegram, etc)
	settings := ninjabot.Settings{
		Pairs: []string{
			"BTCUSDT",
			"ETHUSDT",
		},
	}

	// 初始化你的策略
	// initialize your strategy
	strategy := new(strategies.CrossEMA)

	// 从 CSV 文件加载历史数据
	// load historical data from CSV files
	csvFeed, err := exchange.NewCSVFeed(
		strategy.Timeframe(),
		exchange.PairFeed{
			Pair:      "BTCUSDT",
			File:      "testdata/btc-1h.csv",
			Timeframe: "1h",
		},
		exchange.PairFeed{
			Pair:      "ETHUSDT",
			File:      "testdata/eth-1h.csv",
			Timeframe: "1h",
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	// initialize a database in memory
	// 在内存中初始化数据库
	storage, err := storage.FromMemory()
	if err != nil {
		log.Fatal(err)
	}

	// create a paper wallet for simulation, initializing with 10.000 USDT
	// 为模拟创建一个纸币钱包，初始资金为 10,000 USDT
	wallet := exchange.NewPaperWallet(
		ctx,
		"USDT",
		exchange.WithPaperAsset("USDT", 10000),
		exchange.WithDataFeed(csvFeed),
	)

	// create a chart  with indicators from the strategy and a custom additional RSI indicator
	// 创建一个图表，其中包含策略的指标和一个自定义的 RSI 指标
	chart, err := plot.NewChart(
		plot.WithStrategyIndicators(strategy),
		plot.WithCustomIndicators(
			indicator.RSI(14, "purple"),
		),
		plot.WithPaperWallet(wallet),
	)
	if err != nil {
		log.Fatal(err)
	}

	// initializer Ninjabot with the objects created before
	// 使用之前创建的对象初始化 Ninjabot
	bot, err := ninjabot.NewBot(
		ctx,
		settings,
		wallet,
		strategy,
		// 回测模式必需的选项
		ninjabot.WithBacktest(wallet), // Required for Backtest mode
		ninjabot.WithStorage(storage),

		// connect bot feed (candle and orders) to the chart
		// 将 bot feed（蜡烛和订单）连接到图表
		ninjabot.WithCandleSubscription(chart),
		ninjabot.WithOrderSubscription(chart),
		ninjabot.WithLogLevel(log.WarnLevel),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Initializer simulation
	// 初始化模拟
	err = bot.Run(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Print bot results
	// 打印 bot 结果
	bot.Summary()

	// Display candlesticks chart in local browser
	// 在本地浏览器中显示蜡烛图表
	err = chart.Start()
	if err != nil {
		log.Fatal(err)
	}
}
