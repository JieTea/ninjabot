package indicator

import (
	"fmt"
	"time"

	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/plot"

	"github.com/markcheno/go-talib"
)

func WillR(period int, color string) plot.Indicator {
	return &willR{
		Period: period,
		Color:  color,
	}
}

type willR struct {
	Period int                   // 周期长度
	Color  string                // 图表中的颜色
	Values model.Series[float64] // 威廉指标的数值序列
	Time   []time.Time           // 与指标值对应的时间序列
}

// Warmup 返回威廉指标的预热期
func (w willR) Warmup() int {
	return w.Period
}

// Name 返回威廉指标的名称
func (w willR) Name() string {
	return fmt.Sprintf("%%R(%d)", w.Period)
}

// Overlay 返回威廉指标是否叠加在价格图上
func (w willR) Overlay() bool {
	return false
}

// Load 载入数据并计算威廉指标
func (w *willR) Load(dataframe *model.Dataframe) {
	if len(dataframe.Time) < w.Period {
		return
	}

	w.Values = talib.WillR(dataframe.High, dataframe.Low, dataframe.Close, w.Period)[w.Period:]
	w.Time = dataframe.Time[w.Period:]
}

// Metrics 返回威廉指标的图表数据
func (w willR) Metrics() []plot.IndicatorMetric {
	return []plot.IndicatorMetric{
		{
			Style:  "line",
			Color:  w.Color,
			Values: w.Values,
			Time:   w.Time,
		},
	}
}
