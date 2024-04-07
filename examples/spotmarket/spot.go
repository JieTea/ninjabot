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

// 本示例演示如何在 Binance 使用 NinjaBot 进行现货市场交易。
// This example shows how to use spot market with NinjaBot in Binance
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

	// Initialize your exchange
	// 初始化交易所
	binance, err := exchange.NewBinance(ctx, exchange.WithBinanceCredentials(apiKey, secretKey))
	if err != nil {
		log.Fatalln(err)
	}

	// Initialize your strategy and bot
	// 初始化策略和 bot
	strategy := new(strategies.CrossEMA)
	bot, err := ninjabot.NewBot(ctx, settings, binance, strategy)
	if err != nil {
		log.Fatalln(err)
	}

	// 运行交易 bot
	err = bot.Run(ctx)
	if err != nil {
		log.Fatalln(err)
	}
}
