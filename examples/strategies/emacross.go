package strategies

import (
	"github.com/rodrigo-brito/ninjabot"
	"github.com/rodrigo-brito/ninjabot/indicator"
	"github.com/rodrigo-brito/ninjabot/service"
	"github.com/rodrigo-brito/ninjabot/strategy"
	"github.com/rodrigo-brito/ninjabot/tools/log"
)

// CrossEMA 交叉EMA策略:基于交叉移动平均线（Exponential Moving Average，EMA）来进行交易决策
type CrossEMA struct{}

// Timeframe 返回策略执行的时间周期
func (e CrossEMA) Timeframe() string {
	return "4h"
}

// WarmupPeriod 返回策略执行前需要加载数据的时间周期
func (e CrossEMA) WarmupPeriod() int {
	return 22
}

// Indicators 计算并返回用于图表显示的指标数据
func (e CrossEMA) Indicators(df *ninjabot.Dataframe) []strategy.ChartIndicator {
	// 计算并存储 DataFrame 的 "Close" 列的指数移动平均值（EMA）和简单移动平均值（SMA），分别使用周期为 8 和 21。
	df.Metadata["ema8"] = indicator.EMA(df.Close, 8)
	df.Metadata["sma21"] = indicator.SMA(df.Close, 21)

	// 创建一个包含两个指标（EMA 8 和 SMA 21）的指标列表，用于绘制图表。
	// 这两个指标都属于一个名为 "MA's" 的组，并分别以红色和蓝色线条的样式显示。
	return []strategy.ChartIndicator{
		{
			Overlay:   true,    // 指标叠加在价格图表上
			GroupName: "MA's",  // 指标组的名称
			Time:      df.Time, // 时间戳
			Metrics: []strategy.IndicatorMetric{ // 指标数据
				{
					Values: df.Metadata["ema8"], // 指数移动平均值
					Name:   "EMA 8",             // 指标名称
					Color:  "red",               // 线条颜色
					Style:  strategy.StyleLine,  // 线条样式
				},
				{
					Values: df.Metadata["sma21"], // 简单移动平均值
					Name:   "SMA 21",             // 指标名称
					Color:  "blue",               // 线条颜色
					Style:  strategy.StyleLine,   // 线条样式
				},
			},
		},
	}
}

// OnCandle 根据EMA交叉情况执行交易逻辑
func (e *CrossEMA) OnCandle(df *ninjabot.Dataframe, broker service.Broker) {
	closePrice := df.Close.Last(0)

	assetPosition, quotePosition, err := broker.Position(df.Pair)
	if err != nil {
		log.Error(err)
		return
	}

	// 最小报价资产位置以进行交易
	if quotePosition >= 10 && // minimum quote position to trade
		// 交易信号（EMA8 > SMA21）
		df.Metadata["ema8"].Crossover(df.Metadata["sma21"]) { // trade signal (EMA8 > SMA21)

		// 计算要购买的资产数量
		amount := quotePosition / closePrice // calculate amount of asset to buy
		_, err := broker.CreateOrderMarket(ninjabot.SideTypeBuy, df.Pair, amount)
		if err != nil {
			log.Error(err)
		}

		return
	}

	// 交易信号（EMA8 < SMA21）
	if assetPosition > 0 &&
		df.Metadata["ema8"].Crossunder(df.Metadata["sma21"]) { // trade signal (EMA8 < SMA21)

		_, err = broker.CreateOrderMarket(ninjabot.SideTypeSell, df.Pair, assetPosition)
		if err != nil {
			log.Error(err)
		}
	}
}
