package strategy

import (
	log "github.com/sirupsen/logrus"

	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/service"
)

// Controller 控制策略执行的结构体
type Controller struct {
	strategy  Strategy         // 策略实例
	dataframe *model.Dataframe // 数据帧用于存储蜡烛图数据
	broker    service.Broker   // 经纪人实例
	started   bool             // 标记策略是否已启动
}

// NewStrategyController 创建一个新的策略控制器实例
func NewStrategyController(pair string, strategy Strategy, broker service.Broker) *Controller {
	dataframe := &model.Dataframe{
		Pair:     pair,
		Metadata: make(map[string]model.Series[float64]),
	}

	return &Controller{
		dataframe: dataframe,
		strategy:  strategy,
		broker:    broker,
	}
}

// Start 启动策略
func (s *Controller) Start() {
	s.started = true
}

// OnPartialCandle 处理部分蜡烛图数据
func (s *Controller) OnPartialCandle(candle model.Candle) {
	if !candle.Complete && len(s.dataframe.Close) >= s.strategy.WarmupPeriod() {
		if str, ok := s.strategy.(HighFrequencyStrategy); ok {
			s.updateDataFrame(candle)
			str.Indicators(s.dataframe)
			str.OnPartialCandle(s.dataframe, s.broker)
		}
	}
}

// updateDataFrame 更新数据帧中的数据
func (s *Controller) updateDataFrame(candle model.Candle) {
	if len(s.dataframe.Time) > 0 && candle.Time.Equal(s.dataframe.Time[len(s.dataframe.Time)-1]) {
		last := len(s.dataframe.Time) - 1
		s.dataframe.Close[last] = candle.Close
		s.dataframe.Open[last] = candle.Open
		s.dataframe.High[last] = candle.High
		s.dataframe.Low[last] = candle.Low
		s.dataframe.Volume[last] = candle.Volume
		s.dataframe.Time[last] = candle.Time
		for k, v := range candle.Metadata {
			s.dataframe.Metadata[k][last] = v
		}
	} else {
		s.dataframe.Close = append(s.dataframe.Close, candle.Close)
		s.dataframe.Open = append(s.dataframe.Open, candle.Open)
		s.dataframe.High = append(s.dataframe.High, candle.High)
		s.dataframe.Low = append(s.dataframe.Low, candle.Low)
		s.dataframe.Volume = append(s.dataframe.Volume, candle.Volume)
		s.dataframe.Time = append(s.dataframe.Time, candle.Time)
		s.dataframe.LastUpdate = candle.Time
		for k, v := range candle.Metadata {
			s.dataframe.Metadata[k] = append(s.dataframe.Metadata[k], v)
		}
	}
}

// OnCandle 处理完整的蜡烛图数据
func (s *Controller) OnCandle(candle model.Candle) {
	if len(s.dataframe.Time) > 0 && candle.Time.Before(s.dataframe.Time[len(s.dataframe.Time)-1]) {
		log.Errorf("late candle received: %#v", candle)
		return
	}

	s.updateDataFrame(candle)

	if len(s.dataframe.Close) >= s.strategy.WarmupPeriod() {
		sample := s.dataframe.Sample(s.strategy.WarmupPeriod())
		s.strategy.Indicators(&sample)
		if s.started {
			s.strategy.OnCandle(&sample, s.broker)
		}
	}
}
