package order

import (
	"github.com/rodrigo-brito/ninjabot/model"
)

// DataFeed 用于传递订单数据和错误信息的通道
type DataFeed struct {
	Data chan model.Order // 订单数据通道
	Err  chan error       // 错误信息通道
}

// FeedConsumer 订单数据消费者的函数类型
type FeedConsumer func(order model.Order)

// Subscription 订阅信息结构体
type Subscription struct {
	onlyNewOrder bool         // 是否只订阅新订单
	consumer     FeedConsumer // 订单数据消费者
}

// Feed 订单数据通道和订阅信息的映射关系
type Feed struct {
	OrderFeeds            map[string]*DataFeed      // 交易对和订单数据通道的映射关系
	SubscriptionsBySymbol map[string][]Subscription // 交易对和订阅信息的映射关系
}

// NewOrderFeed 创建一个新的订单数据通道和订阅信息的实例
func NewOrderFeed() *Feed {
	return &Feed{
		OrderFeeds:            make(map[string]*DataFeed),
		SubscriptionsBySymbol: make(map[string][]Subscription),
	}
}

// Subscribe 向指定交易对的订阅信息中添加一个订阅者
func (d *Feed) Subscribe(pair string, consumer FeedConsumer, onlyNewOrder bool) {
	if _, ok := d.OrderFeeds[pair]; !ok {
		d.OrderFeeds[pair] = &DataFeed{
			Data: make(chan model.Order),
			Err:  make(chan error),
		}
	}

	d.SubscriptionsBySymbol[pair] = append(d.SubscriptionsBySymbol[pair], Subscription{
		onlyNewOrder: onlyNewOrder,
		consumer:     consumer,
	})
}

// Publish 将订单数据发布到指定交易对的订单数据通道中，以便通知所有订阅者
func (d *Feed) Publish(order model.Order, _ bool) {
	if _, ok := d.OrderFeeds[order.Pair]; ok {
		d.OrderFeeds[order.Pair].Data <- order
	}
}

// Start 启动订单数据的消费者，即开启一个 goroutine 来监听订单数据通道，并将数据传递给对应的订阅者
func (d *Feed) Start() {
	for pair := range d.OrderFeeds {
		go func(pair string, feed *DataFeed) {
			for order := range feed.Data {
				for _, subscription := range d.SubscriptionsBySymbol[pair] {
					subscription.consumer(order)
				}
			}
		}(pair, d.OrderFeeds[pair])
	}
}
