package tools

import (
	"github.com/rodrigo-brito/ninjabot"
	"github.com/rodrigo-brito/ninjabot/service"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

// OrderCondition 定义订单的条件，包括判断条件的函数、交易量和交易方向。
type OrderCondition struct {
	Condition func(df *ninjabot.Dataframe) bool // 判断条件的函数
	Size      float64                           // 交易量
	Side      ninjabot.SideType                 // 交易方向
}

// Scheduler 管理订单条件，并在满足条件时执行订单。
type Scheduler struct {
	pair            string           // 交易对
	orderConditions []OrderCondition // 订单条件列表
}

// NewScheduler 创建一个新的 Scheduler 实例。
func NewScheduler(pair string) *Scheduler {
	return &Scheduler{pair: pair}
}

// SellWhen 当满足条件时，执行卖出订单。
func (s *Scheduler) SellWhen(size float64, condition func(df *ninjabot.Dataframe) bool) {
	s.orderConditions = append(
		s.orderConditions,
		OrderCondition{Condition: condition, Size: size, Side: ninjabot.SideTypeSell},
	)
}

// BuyWhen 当满足条件时，执行买入订单。
func (s *Scheduler) BuyWhen(size float64, condition func(df *ninjabot.Dataframe) bool) {
	s.orderConditions = append(
		s.orderConditions,
		OrderCondition{Condition: condition, Size: size, Side: ninjabot.SideTypeBuy},
	)
}

// Update 根据当前数据更新订单条件，并执行满足条件的订单。
func (s *Scheduler) Update(df *ninjabot.Dataframe, broker service.Broker) {
	s.orderConditions = lo.Filter[OrderCondition](s.orderConditions, func(oc OrderCondition, _ int) bool {
		if oc.Condition(df) {
			_, err := broker.CreateOrderMarket(oc.Side, s.pair, oc.Size)
			if err != nil {
				log.Error(err)
				return true
			}
			return false
		}
		return true
	})
}
