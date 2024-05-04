package indicator

import (
	"fmt"
	"time"

	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/plot"

	"github.com/markcheno/go-talib"
)

// BollingerBands 返回一个布林带指标对象，用于计算布林带指标
func BollingerBands(period int, stdDeviation float64, upDnBandColor, midBandColor string) plot.Indicator {
	return &bollingerBands{
		Period:        period,
		StdDeviation:  stdDeviation,
		UpDnBandColor: upDnBandColor,
		MidBandColor:  midBandColor,
	}
}

// bollingerBands 表示布林带指标，包含了计算布林带所需的参数和计算结果
type bollingerBands struct {
	Period        int
	StdDeviation  float64
	UpDnBandColor string
	MidBandColor  string
	UpperBand     model.Series[float64]
	MiddleBand    model.Series[float64]
	LowerBand     model.Series[float64]
	Time          []time.Time
}

// Warmup 返回指标需要的预热周期数，即计算指标所需的初始数据量
func (bb bollingerBands) Warmup() int {
	return bb.Period
}

// Name 返回指标的名称，格式为"BB(周期, 标准差)"
func (bb bollingerBands) Name() string {
	return fmt.Sprintf("BB(%d, %.2f)", bb.Period, bb.StdDeviation)
}

// Overlay 返回一个布尔值，表示指标是否叠加在价格图上
func (bb bollingerBands) Overlay() bool {
	return true
}

// Load 根据传入的数据Dataframe计算布林带指标的上轨、中轨和下轨，并保存计算结果和时间序列
func (bb *bollingerBands) Load(dataframe *model.Dataframe) {
	// 检查数据长度是否小于指标期数，如果是，则直接返回
	if len(dataframe.Time) < bb.Period {
		return
	}

	// 使用 talib 计算布林带指标的上轨、中轨和下轨数据
	upper, mid, lower := talib.BBands(dataframe.Close, bb.Period, bb.StdDeviation, bb.StdDeviation, talib.EMA)
	// 更新指标的上轨、中轨和下轨数据，从 bb.Period 索引开始，因为前面的数据用于热身期
	bb.UpperBand, bb.MiddleBand, bb.LowerBand = upper[bb.Period:], mid[bb.Period:], lower[bb.Period:]

	// 更新指标的时间序列数据，从 bb.Period 索引开始
	bb.Time = dataframe.Time[bb.Period:]
}

// Metrics 返回一个指标度量的切片，包含了布林带指标的上轨、中轨和下轨的信息，用于绘制图表
func (bb bollingerBands) Metrics() []plot.IndicatorMetric {
	return []plot.IndicatorMetric{
		// 上轨数据
		{
			Style:  "line",
			Color:  bb.UpDnBandColor, // 上下轨相同颜色
			Values: bb.UpperBand,     // 上轨数据
			Time:   bb.Time,          // 时间序列
		},
		// 中轨数据
		{
			Style:  "line",
			Color:  bb.MidBandColor, // 中轨颜色
			Values: bb.MiddleBand,   // 中轨数据
			Time:   bb.Time,         // 时间序列
		},
		// 下轨数据
		{
			Style:  "line",
			Color:  bb.UpDnBandColor, // 上下轨相同颜色
			Values: bb.LowerBand,     // 下轨数据
			Time:   bb.Time,          // 时间序列
		},
	}
}
