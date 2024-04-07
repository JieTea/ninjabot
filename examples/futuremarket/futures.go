package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/rodrigo-brito/ninjabot"
	"github.com/rodrigo-brito/ninjabot/examples/strategies"
	"github.com/rodrigo-brito/ninjabot/exchange"
)

// This example shows how to use futures market with NinjaBot.
// 示例演示如何在 NinjaBot 中使用期货市场。
func main() {
	var (
		ctx             = context.Background()
		apiKey          = os.Getenv("API_KEY")
		secretKey       = os.Getenv("API_SECRET")
		telegramToken   = os.Getenv("TELEGRAM_TOKEN")
		telegramUser, _ = strconv.Atoi(os.Getenv("TELEGRAM_USER"))
	)

	settings := ninjabot.Settings{
		Pairs: []string{
			"BTCUSDT",
			"ETHUSDT",
		},
		Telegram: ninjabot.TelegramSettings{
			Enabled: true,
			Token:   telegramToken,
			Users:   []int{telegramUser},
		},
	}

	// Initialize your exchange with futures
	// 使用期货市场初始化你的交易所
	binance, err := exchange.NewBinanceFuture(ctx,
		exchange.WithBinanceFutureCredentials(apiKey, secretKey),
		exchange.WithBinanceFutureLeverage("BTCUSDT", 1, exchange.MarginTypeIsolated),
		exchange.WithBinanceFutureLeverage("ETHUSDT", 1, exchange.MarginTypeIsolated),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize your strategy and bot
	// 初始化你的策略和 bot
	strategy := new(strategies.CrossEMA)                          // 初始化一个策略
	bot, err := ninjabot.NewBot(ctx, settings, binance, strategy) // 初始化交易的bot
	if err != nil {
		log.Fatalln(err)
	}

	err = bot.Run(ctx)
	if err != nil {
		log.Fatalln(err)
	}
}
