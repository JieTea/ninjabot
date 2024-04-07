package strategies

import (
	"github.com/rodrigo-brito/ninjabot"
	"github.com/rodrigo-brito/ninjabot/indicator"
	"github.com/rodrigo-brito/ninjabot/service"
	"github.com/rodrigo-brito/ninjabot/strategy"
	"github.com/rodrigo-brito/ninjabot/tools/log"
)

// Turtle 基于海龟交易策略的交易策略。海龟交易是一种趋势跟踪策略，根据市场价格的趋势进行交易。
// 该策略使用了以下指标：
//
//	最高价的40周期移动最大值(Max): 如果没有持仓且当前价格超过了最近40个K线的最高价，则以账户余额的一半买入该资产。
//	收盘价的20周期移动最小值(Min): 如果有持仓且当前价格低于最近20个K线的最低价，则卖出所有持仓
//
// https://www.investopedia.com/articles/trading/08/turtle-trading.asp
type Turtle struct{}

func (e Turtle) Timeframe() string {
	return "4h"
}

func (e Turtle) WarmupPeriod() int {
	return 40
}

func (e Turtle) Indicators(df *ninjabot.Dataframe) []strategy.ChartIndicator {
	// 计算最近40根K线的最高价
	df.Metadata["max40"] = indicator.Max(df.Close, 40)
	// 计算最近20根K线的最低价
	df.Metadata["low20"] = indicator.Min(df.Close, 20)

	return nil
}

func (e *Turtle) OnCandle(df *ninjabot.Dataframe, broker service.Broker) {
	closePrice := df.Close.Last(0)
	highest := df.Metadata["max40"].Last(0)
	lowest := df.Metadata["low20"].Last(0)

	assetPosition, quotePosition, err := broker.Position(df.Pair)
	if err != nil {
		log.Error(err)
		return
	}

	// 如果持仓已经开启，等待直到关闭
	// If position already open wait till it will be closed
	if assetPosition == 0 && closePrice >= highest {
		// 使用一半的可用资金购买资产
		_, err := broker.CreateOrderMarketQuote(ninjabot.SideTypeBuy, df.Pair, quotePosition/2)
		if err != nil {
			log.Error(err)
		}
		return
	}

	// 如果持仓已经开启且价格低于最近20根K线的最低价，卖出全部资产
	if assetPosition > 0 && closePrice <= lowest {
		_, err := broker.CreateOrderMarket(ninjabot.SideTypeSell, df.Pair, assetPosition)
		if err != nil {
			log.Error(err)
		}
	}
}
