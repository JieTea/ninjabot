package storage

import (
	"time"

	"github.com/rodrigo-brito/ninjabot/model"
)

// OrderFilter 过滤订单的函数类型
type OrderFilter func(model.Order) bool

// Storage 存储接口，包括创建订单、更新订单和获取订单列表
type Storage interface {
	CreateOrder(order *model.Order) error
	UpdateOrder(order *model.Order) error
	Orders(filters ...OrderFilter) ([]*model.Order, error)
}

// WithStatusIn 根据订单状态过滤订单的函数，可传入多个状态
func WithStatusIn(status ...model.OrderStatusType) OrderFilter {
	return func(order model.Order) bool {
		for _, s := range status {
			if s == order.Status {
				return true
			}
		}
		return false
	}
}

// WithStatus 根据订单状态过滤订单的函数，只能传入一个状态
func WithStatus(status model.OrderStatusType) OrderFilter {
	return func(order model.Order) bool {
		return order.Status == status
	}
}

// WithPair 根据交易对过滤订单的函数
func WithPair(pair string) OrderFilter {
	return func(order model.Order) bool {
		return order.Pair == pair
	}
}

// WithUpdateAtBeforeOrEqual 根据更新时间早于或等于指定时间过滤订单的函数
func WithUpdateAtBeforeOrEqual(time time.Time) OrderFilter {
	return func(order model.Order) bool {
		return !order.UpdatedAt.After(time)
	}
}
