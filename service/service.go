//go:generate go run github.com/vektra/mockery/v2 --all --with-expecter --output=../testdata/mocks

package service

import (
	"context"
	"time"

	"github.com/rodrigo-brito/ninjabot/model"
)

// Exchange 合并了 Broker 和 Feeder 两个接口
type Exchange interface {
	Broker
	Feeder
}

// Feeder 些获取市场数据的方法，如获取资产信息、获取最新报价、获取K线数据等
type Feeder interface {
	AssetsInfo(pair string) model.AssetInfo
	LastQuote(ctx context.Context, pair string) (float64, error)
	CandlesByPeriod(ctx context.Context, pair, period string, start, end time.Time) ([]model.Candle, error)
	CandlesByLimit(ctx context.Context, pair, period string, limit int) ([]model.Candle, error)
	CandlesSubscription(ctx context.Context, pair, timeframe string) (chan model.Candle, chan error)
}

// Broker 与交易所交互的方法，如查询账户信息、查询持仓信息、下单
type Broker interface {
	Account() (model.Account, error)
	Position(pair string) (asset, quote float64, err error)
	Order(pair string, id int64) (model.Order, error)
	CreateOrderOCO(side model.SideType, pair string, size, price, stop, stopLimit float64) ([]model.Order, error)
	CreateOrderLimit(side model.SideType, pair string, size float64, limit float64) (model.Order, error)
	CreateOrderMarket(side model.SideType, pair string, size float64) (model.Order, error)
	CreateOrderMarketQuote(side model.SideType, pair string, quote float64) (model.Order, error)
	CreateOrderStop(pair string, quantity float64, limit float64) (model.Order, error)
	Cancel(model.Order) error
}

// Notifier 通知、订单和错误通知
type Notifier interface {
	Notify(string)
	OnOrder(order model.Order)
	OnError(err error)
}

// Telegram 通知、订单和错误通知
type Telegram interface {
	Notifier
	Start()
}
