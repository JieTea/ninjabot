package strategy

import (
	"time"

	"github.com/rodrigo-brito/ninjabot/model"
)

// MetricStyle 指标样式的类型
type MetricStyle string

const (
	StyleBar       = "bar"
	StyleScatter   = "scatter"
	StyleLine      = "line"
	StyleHistogram = "histogram"
	StyleWaterfall = "waterfall"
)

// IndicatorMetric 一个指标的度量，包含名称、颜色、样式和值等信息。
type IndicatorMetric struct {
	Name   string
	Color  string
	Style  MetricStyle // default: line
	Values model.Series[float64]
}

// ChartIndicator 一个指标，包含时间、指标度量、是否叠加、组名和预热期等信息
type ChartIndicator struct {
	Time      []time.Time
	Metrics   []IndicatorMetric
	Overlay   bool
	GroupName string
	Warmup    int
}
