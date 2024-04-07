package strategies

import (
	"github.com/markcheno/go-talib"

	"github.com/rodrigo-brito/ninjabot/indicator"
	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/service"
	"github.com/rodrigo-brito/ninjabot/strategy"
	"github.com/rodrigo-brito/ninjabot/tools/log"
)

// OCOSell 随机指标的交易策略，该策略基于随机指标（Stochastic Oscillator）来进行交易决策基于随机指标，
// 并使用OCO（One Cancels the Other）订单来管理风险。
type OCOSell struct{}

// Timeframe 返回策略执行的时间周期
func (e OCOSell) Timeframe() string {
	return "1d"
}

// WarmupPeriod 策略执行前需要加载数据的时间周期。
func (e OCOSell) WarmupPeriod() int {
	return 9
}

// Indicators 计算并返回用于图表显示的指标数据
func (e OCOSell) Indicators(df *model.Dataframe) []strategy.ChartIndicator {
	df.Metadata["stoch"], df.Metadata["stoch_signal"] = indicator.Stoch(
		df.High,
		df.Low,
		df.Close,
		8,
		3,
		talib.SMA,
		3,
		talib.SMA,
	)

	return []strategy.ChartIndicator{
		{
			Overlay:   false,
			GroupName: "Stochastic",
			Time:      df.Time,
			Metrics: []strategy.IndicatorMetric{
				{
					Values: df.Metadata["stoch"],
					Name:   "K",
					Color:  "red",
					Style:  strategy.StyleLine,
				},
				{
					Values: df.Metadata["stoch_signal"],
					Name:   "D",
					Color:  "blue",
					Style:  strategy.StyleLine,
				},
			},
		},
	}
}

// OnCandle 根据随机指标的交叉情况执行买卖操作。
func (e *OCOSell) OnCandle(df *model.Dataframe, broker service.Broker) {
	closePrice := df.Close.Last(0)
	log.Info("New Candle = ", df.Pair, df.LastUpdate, closePrice)

	assetPosition, quotePosition, err := broker.Position(df.Pair)
	if err != nil {
		log.Error(err)
		return
	}

	buyAmount := 4000.0
	if quotePosition > buyAmount && df.Metadata["stoch"].Crossover(df.Metadata["stoch_signal"]) {
		size := buyAmount / closePrice
		_, err := broker.CreateOrderMarket(model.SideTypeBuy, df.Pair, size)
		if err != nil {
			log.WithFields(map[string]interface{}{
				"pair":  df.Pair,
				"side":  model.SideTypeBuy,
				"close": closePrice,
				"asset": assetPosition,
				"quote": quotePosition,
				"size":  size,
			}).Error(err)
		}

		_, err = broker.CreateOrderOCO(model.SideTypeSell, df.Pair, size, closePrice*1.1, closePrice*0.95, closePrice*0.95)
		if err != nil {
			log.WithFields(map[string]interface{}{
				"pair":  df.Pair,
				"side":  model.SideTypeBuy,
				"close": closePrice,
				"asset": assetPosition,
				"quote": quotePosition,
				"size":  size,
			}).Error(err)
		}
	}
}
