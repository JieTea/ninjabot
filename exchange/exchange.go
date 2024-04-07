package exchange

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/StudioSol/set"

	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/service"
	"github.com/rodrigo-brito/ninjabot/tools/log"
)

/*用于订阅和处理交易所交易数据的模块*/

var (
	ErrInvalidQuantity   = errors.New("invalid quantity")
	ErrInsufficientFunds = errors.New("insufficient funds or locked")
	ErrInvalidAsset      = errors.New("invalid asset")
)

// DataFeed 表示用于接收蜡烛图数据的通道
type DataFeed struct {
	Data chan model.Candle
	Err  chan error
}

// DataFeedSubscription 管理给定交易所的数据订阅，包括交易对、时间框架和相应的数据消费者。
type DataFeedSubscription struct {
	exchange                service.Exchange
	Feeds                   *set.LinkedHashSetString
	DataFeeds               map[string]*DataFeed
	SubscriptionsByDataFeed map[string][]Subscription
}

// Subscription 表示一个订阅，包括是否在蜡烛图关闭时触发以及相应的数据消费者
type Subscription struct {
	onCandleClose bool
	consumer      DataFeedConsumer
}

// OrderError 表示下单时发生的错误
type OrderError struct {
	Err      error
	Pair     string
	Quantity float64
}

func (o *OrderError) Error() string {
	return fmt.Sprintf("order error: %v", o.Err)
}

// DataFeedConsumer 是一个消费蜡烛图数据的函数
type DataFeedConsumer func(model.Candle)

// NewDataFeed 用于创建一个新的数据订阅
func NewDataFeed(exchange service.Exchange) *DataFeedSubscription {
	return &DataFeedSubscription{
		exchange:                exchange,
		Feeds:                   set.NewLinkedHashSetString(),
		DataFeeds:               make(map[string]*DataFeed),
		SubscriptionsByDataFeed: make(map[string][]Subscription),
	}
}

// feedKey 为给定交易对和时间框架生成一个键
func (d *DataFeedSubscription) feedKey(pair, timeframe string) string {
	return fmt.Sprintf("%s--%s", pair, timeframe)
}

// pairTimeframeFromKey 从给定键中提取交易对和时间框架
func (d *DataFeedSubscription) pairTimeframeFromKey(key string) (pair, timeframe string) {
	parts := strings.Split(key, "--")
	return parts[0], parts[1]
}

// Subscribe 向特定交易对和时间框架订阅数据，可以指定在蜡烛图关闭时触发
func (d *DataFeedSubscription) Subscribe(pair, timeframe string, consumer DataFeedConsumer, onCandleClose bool) {
	key := d.feedKey(pair, timeframe)
	d.Feeds.Add(key)
	d.SubscriptionsByDataFeed[key] = append(d.SubscriptionsByDataFeed[key], Subscription{
		onCandleClose: onCandleClose,
		consumer:      consumer,
	})
}

// Preload 预加载历史数据，将历史数据推送给已经订阅的消费者
func (d *DataFeedSubscription) Preload(pair, timeframe string, candles []model.Candle) {
	log.Infof("[SETUP] preloading %d candles for %s-%s", len(candles), pair, timeframe)
	key := d.feedKey(pair, timeframe)
	for _, candle := range candles {
		if !candle.Complete {
			continue
		}

		for _, subscription := range d.SubscriptionsByDataFeed[key] {
			subscription.consumer(candle)
		}
	}
}

// Connect 连接到交易所，开始接收数据。
func (d *DataFeedSubscription) Connect() {
	log.Infof("Connecting to the exchange.")
	for feed := range d.Feeds.Iter() {
		pair, timeframe := d.pairTimeframeFromKey(feed)
		ccandle, cerr := d.exchange.CandlesSubscription(context.Background(), pair, timeframe)
		d.DataFeeds[feed] = &DataFeed{
			Data: ccandle,
			Err:  cerr,
		}
	}
}

// Start 用于启动数据接收循环，将接收到的数据推送给相应的消费者。
func (d *DataFeedSubscription) Start(loadSync bool) {
	d.Connect()
	wg := new(sync.WaitGroup)
	for key, feed := range d.DataFeeds {
		wg.Add(1)
		go func(key string, feed *DataFeed) {
			for {
				select {
				case candle, ok := <-feed.Data:
					if !ok {
						wg.Done()
						return
					}
					for _, subscription := range d.SubscriptionsByDataFeed[key] {
						if subscription.onCandleClose && !candle.Complete {
							continue
						}
						subscription.consumer(candle)
					}
				case err := <-feed.Err:
					if err != nil {
						log.Error("dataFeedSubscription/start: ", err)
					}
				}
			}
		}(key, feed)
	}

	log.Infof("Data feed connected.")
	if loadSync {
		wg.Wait()
	}
}
