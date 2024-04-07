package strategies

import (
	"github.com/rodrigo-brito/ninjabot"
	"github.com/rodrigo-brito/ninjabot/indicator"
	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/service"
	"github.com/rodrigo-brito/ninjabot/strategy"
	"github.com/rodrigo-brito/ninjabot/tools"
	"github.com/rodrigo-brito/ninjabot/tools/log"
)

// 基于EMA和SMA指标来进行交易，并实现了动态止损功能。
type trailing struct {
	trailingStop map[string]*tools.TrailingStop // 一个映射，将交易对（pair）映射到动态止损工具（tools.TrailingStop）的实例。动态止损工具用于根据市场价格变化调整止损价格。
	scheduler    map[string]*tools.Scheduler    // 一个映射，将交易对（pair）映射到调度器（tools.Scheduler）的实例。调度器用于调度执行某些操作的时间点。
}

// NewTrailing 创建一个新的基于EMA和SMA指标的策略实例
func NewTrailing(pairs []string) strategy.HighFrequencyStrategy {
	strategy := &trailing{
		trailingStop: make(map[string]*tools.TrailingStop),
		scheduler:    make(map[string]*tools.Scheduler),
	}

	for _, pair := range pairs {
		strategy.trailingStop[pair] = tools.NewTrailingStop()
		strategy.scheduler[pair] = tools.NewScheduler(pair)
	}

	return strategy
}

// Timeframe 返回策略执行的时间周期
func (t trailing) Timeframe() string {
	return "4h"
}

// WarmupPeriod 返回策略执行前需要加载数据的时间周期
func (t trailing) WarmupPeriod() int {
	return 21
}

// Indicators 计算并返回用于图表显示的指标数据
func (t trailing) Indicators(df *model.Dataframe) []strategy.ChartIndicator {
	df.Metadata["ema_fast"] = indicator.EMA(df.Close, 8)
	df.Metadata["sma_slow"] = indicator.SMA(df.Close, 21)

	return nil
}

// OnCandle 根据EMA和SMA指标的交叉情况执行买入操作，并启动动态止损
func (t trailing) OnCandle(df *model.Dataframe, broker service.Broker) {
	// 获取当前交易对的资产和报价货币的持仓情况
	asset, quote, err := broker.Position(df.Pair)
	if err != nil {
		log.Error(err)
		return
	}

	// 检查是否有足够的报价货币来进行交易，是否还没有持仓，
	// 并且当前的快速指数移动平均线穿越了慢速简单移动平均线
	if quote > 10.0 && // enough cash?
		asset*df.Close.Last(0) < 10 && // without position yet
		df.Metadata["ema_fast"].Crossover(df.Metadata["sma_slow"]) {
		// 如果条件满足，以市场价格买入
		_, err = broker.CreateOrderMarketQuote(ninjabot.SideTypeBuy, df.Pair, quote)
		if err != nil {
			log.Error(err)
			return
		}

		// 启动跟踪止损功能
		t.trailingStop[df.Pair].Start(df.Close.Last(0), df.Low.Last(0))

		return
	}
}

// OnPartialCandle 在部分蜡烛数据上更新动态止损，并在触发止损条件时执行卖出操作
func (t trailing) OnPartialCandle(df *model.Dataframe, broker service.Broker) {
	// 检查是否存在跟踪止损功能，并更新最新价格
	if trailing := t.trailingStop[df.Pair]; trailing != nil && trailing.Update(df.Close.Last(0)) {
		// 获取当前持仓的资产数量
		asset, _, err := broker.Position(df.Pair)
		if err != nil {
			log.Error(err)
			return
		}

		// 如果持仓量大于零，则以市场价格卖出
		if asset > 0 {
			_, err = broker.CreateOrderMarket(ninjabot.SideTypeSell, df.Pair, asset)
			if err != nil {
				log.Error(err)
				return
			}
			// 停止跟踪止损功能
			trailing.Stop()
		}
	}
}
