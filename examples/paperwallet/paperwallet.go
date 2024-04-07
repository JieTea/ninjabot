package main

import (
	"context"
	"os"
	"strconv"

	"github.com/rodrigo-brito/ninjabot/plot"
	"github.com/rodrigo-brito/ninjabot/plot/indicator"

	"github.com/rodrigo-brito/ninjabot"
	"github.com/rodrigo-brito/ninjabot/examples/strategies"
	"github.com/rodrigo-brito/ninjabot/exchange"
	"github.com/rodrigo-brito/ninjabot/storage"
	"github.com/rodrigo-brito/ninjabot/tools/log"
)

// 使用 NinjaBot 进行模拟交易。它包括设置 Telegram 通知、创建虚拟交易所（PaperWallet）、初始化策略、创建图表以及运行交易的步骤。
// This example shows how to use NinjaBot with a simulation with a fake exchange
// A peperwallet is a wallet that is not connected to any exchange, it is a simulation with live data (realtime)
func main() {
	var (
		ctx             = context.Background()
		telegramToken   = os.Getenv("TELEGRAM_TOKEN")
		telegramUser, _ = strconv.Atoi(os.Getenv("TELEGRAM_USER"))
	)

	settings := ninjabot.Settings{
		Pairs: []string{
			"BTCUSDT",
			"ETHUSDT",
			"BNBUSDT",
			"LTCUSDT",
		},
		Telegram: ninjabot.TelegramSettings{
			Enabled: telegramToken != "" && telegramUser != 0,
			Token:   telegramToken,
			Users:   []int{telegramUser},
		},
	}

	// Use binance for realtime data feed
	// 使用 Binance 作为实时数据源
	binance, err := exchange.NewBinance(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// creating a storage to save trades
	// 创建一个用于保存交易的存储
	storage, err := storage.FromMemory()
	if err != nil {
		log.Fatal(err)
	}

	// creating a paper wallet to simulate an exchange waller for fake operataions
	// paper wallet is simulation of a real exchange wallet
	// 创建一个用于模拟交易的 PaperWallet
	paperWallet := exchange.NewPaperWallet(
		ctx,
		"USDT",
		exchange.WithPaperFee(0.001, 0.001),
		exchange.WithPaperAsset("USDT", 10000),
		exchange.WithDataFeed(binance),
	)

	// initializing my strategy
	// 初始化策略
	strategy := new(strategies.CrossEMA)

	// 创建一个图表
	chart, err := plot.NewChart(
		plot.WithCustomIndicators(
			indicator.EMA(8, "red"),
			indicator.SMA(21, "blue"),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	// initializer ninjabot
	// 初始化 NinjaBot
	bot, err := ninjabot.NewBot(
		ctx,
		settings,
		paperWallet,
		strategy,
		ninjabot.WithStorage(storage),
		ninjabot.WithPaperWallet(paperWallet),
		ninjabot.WithCandleSubscription(chart),
		ninjabot.WithOrderSubscription(chart),
	)
	if err != nil {
		log.Fatal(err)
	}

	// 启动图表
	go func() {
		err := chart.Start()
		if err != nil {
			log.Fatal(err)
		}
	}()

	// 运行 NinjaBot
	err = bot.Run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
