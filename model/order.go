package model

import (
	"fmt"
	"time"
)

type SideType string        // SideType 交易方向类型
type OrderType string       // SideType 单类型
type OrderStatusType string // OrderStatusType 订单状态类型

var (
	// 定义交易方向常量
	SideTypeBuy  SideType = "BUY"
	SideTypeSell SideType = "SELL"

	// 定义订单类型常量
	OrderTypeLimit           OrderType = "LIMIT"
	OrderTypeMarket          OrderType = "MARKET"
	OrderTypeLimitMaker      OrderType = "LIMIT_MAKER"
	OrderTypeStopLoss        OrderType = "STOP_LOSS"
	OrderTypeStopLossLimit   OrderType = "STOP_LOSS_LIMIT"
	OrderTypeTakeProfit      OrderType = "TAKE_PROFIT"
	OrderTypeTakeProfitLimit OrderType = "TAKE_PROFIT_LIMIT"

	// 定义订单状态类型常量
	OrderStatusTypeNew             OrderStatusType = "NEW"
	OrderStatusTypePartiallyFilled OrderStatusType = "PARTIALLY_FILLED"
	OrderStatusTypeFilled          OrderStatusType = "FILLED"
	OrderStatusTypeCanceled        OrderStatusType = "CANCELED"
	OrderStatusTypePendingCancel   OrderStatusType = "PENDING_CANCEL"
	OrderStatusTypeRejected        OrderStatusType = "REJECTED"
	OrderStatusTypeExpired         OrderStatusType = "EXPIRED"
)

// Order 订单结构体
type Order struct {
	ID         int64           `db:"id" json:"id" gorm:"primaryKey,autoIncrement"`
	ExchangeID int64           `db:"exchange_id" json:"exchange_id"` // 交易所ID
	Pair       string          `db:"pair" json:"pair"`               // 交易对
	Side       SideType        `db:"side" json:"side"`               // 买卖方向
	Type       OrderType       `db:"type" json:"type"`               // 订单类型
	Status     OrderStatusType `db:"status" json:"status"`           // 订单状态
	Price      float64         `db:"price" json:"price"`             // 价格
	Quantity   float64         `db:"quantity" json:"quantity"`       // 数量

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`

	// OCO订单专用
	// OCO Orders only
	Stop    *float64 `db:"stop" json:"stop"`         // 止损价格
	GroupID *int64   `db:"group_id" json:"group_id"` // 组ID

	// 内部使用（Plot）
	// Internal use (Plot)
	RefPrice    float64 `json:"ref_price" gorm:"-"`    // 参考价格
	Profit      float64 `json:"profit" gorm:"-"`       // 利润
	ProfitValue float64 `json:"profit_value" gorm:"-"` // 利润价值
	Candle      Candle  `json:"-" gorm:"-"`            // K线数据: 分析订单执行时的市场情况
}

// String 返回订单的字符串表示
func (o Order) String() string {
	return fmt.Sprintf("[%s] %s %s | ID: %d, Type: %s, %f x $%f (~$%.f)",
		o.Status, o.Side, o.Pair, o.ID, o.Type, o.Quantity, o.Price, o.Quantity*o.Price)
}
