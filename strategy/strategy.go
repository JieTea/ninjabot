package strategy

import (
	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/service"
)

// Strategy 策略的基本行为
type Strategy interface {
	// Timeframe is the time interval in which the strategy will be executed. eg: 1h, 1d, 1w
	// 策略执行的时间间隔
	Timeframe() string
	// WarmupPeriod is the necessary time to wait before executing the strategy, to load data for indicators.
	// This time is measured in the period specified in the `Timeframe` function.
	// 策略执行前需要等待的时间，以加载指标所需的数据。
	// 这个时间是以 Timeframe 函数中指定的周期为单位计算的
	WarmupPeriod() int
	// Indicators will be executed for each new candle, in order to fill indicators before `OnCandle` function is called.
	// 每个新蜡烛图中执行指标，以填充指标，以便在调用 OnCandle 函数之前执行。
	Indicators(df *model.Dataframe) []ChartIndicator
	// OnCandle will be executed for each new candle, after indicators are filled, here you can do your trading logic.
	// OnCandle is executed after the candle close.
	// 在每个新蜡烛图中执行，指标填充后执行，在这里可以编写交易逻辑。在蜡烛图关闭后执行。
	OnCandle(df *model.Dataframe, broker service.Broker)
}

// HighFrequencyStrategy 高频策略的行为
type HighFrequencyStrategy interface {
	Strategy

	// OnPartialCandle will be executed for each new partial candle, after indicators are filled.
	// 每个新的部分蜡烛图中执行，指标填充后执行
	OnPartialCandle(df *model.Dataframe, broker service.Broker)
}
