package exchange

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/common"

	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/service"
	"github.com/rodrigo-brito/ninjabot/tools/log"
)

/**
实现了一个模拟交易钱包的功能，主要用于在模拟环境中进行加密货币交易策略的测试和验证。其设计目的包括以下几个方面：
模拟环境: 提供一个模拟的交易环境，可以在其中测试和验证交易策略，而无需使用真实的加密货币资金。
交易逻辑: 实现了交易的各种逻辑，包括订单的创建、取消、成交等，以及资金的验证和更新。
账户管理: 提供了账户信息的管理，包括资产余额、持仓信息等，方便用户了解交易情况。
订单管理: 实现了订单的管理功能，包括创建不同类型的订单（限价、市价、止损止盈等）、查询订单信息、取消订单等。
资产价值计算: 可以计算不同交易对的资产价值，包括单个资产的价值变化和整个钱包的总价值变化。
风险控制: 提供了一些风险控制的功能，如最大回撤计算等，帮助用户评估交易策略的风险。
蜡烛图数据: 可以获取和处理蜡烛图数据，用于分析市场趋势和制定交易策略
*/

// 资产的结构体
type assetInfo struct {
	Free float64 // 可用余额
	Lock float64 // 锁定余额
}

// AssetValue 资产价值
type AssetValue struct {
	Time  time.Time // 时间
	Value float64   // 值
}

// PaperWallet 虚拟钱包的结构体
// 模拟交易环境中跟踪资产、订单、交易量等信息，
// 并提供订单创建、资金验证、持仓管理等功能
type PaperWallet struct {
	sync.Mutex
	ctx           context.Context         // 上下文对象
	baseCoin      string                  // 基础货币
	counter       int64                   // 计数器，用于生成唯一的订单ID
	takerFee      float64                 // 吃单手续费率
	makerFee      float64                 // 挂单手续费率
	initialValue  float64                 // 初始价值
	feeder        service.Feeder          // 数据源
	orders        []model.Order           // 订单列表
	assets        map[string]*assetInfo   // 资产信息，key为货币对，value为资产信息结构体指针
	avgShortPrice map[string]float64      // 平均空头价格，key为货币对，value为价格
	avgLongPrice  map[string]float64      // 平均多头价格，key为货币对，value为价格
	volume        map[string]float64      // 交易量，key为货币对，value为交易量
	lastCandle    map[string]model.Candle // 最后一根K线，key为货币对，value为K线数据
	fistCandle    map[string]model.Candle // 第一根K线，key为货币对，value为K线数据
	assetValues   map[string][]AssetValue // 资产价值历史记录，key为货币对，value为价值历史记录
	equityValues  []AssetValue            // 账户总价值历史记录
}

// AssetsInfo 返回给定交易对的资产信息
func (p *PaperWallet) AssetsInfo(pair string) model.AssetInfo {
	asset, quote := SplitAssetQuote(pair)
	return model.AssetInfo{
		BaseAsset:          asset,
		QuoteAsset:         quote,
		MaxPrice:           math.MaxFloat64,
		MaxQuantity:        math.MaxFloat64,
		StepSize:           0.00000001,
		TickSize:           0.00000001,
		QuotePrecision:     8,
		BaseAssetPrecision: 8,
	}
}

// PaperWalletOption 定义 PaperWallet 的选项函数类型
type PaperWalletOption func(*PaperWallet)

// WithPaperAsset 设置指定交易对的资产
func WithPaperAsset(pair string, amount float64) PaperWalletOption {
	return func(wallet *PaperWallet) {
		wallet.assets[pair] = &assetInfo{
			Free: amount,
			Lock: 0,
		}
	}
}

// WithPaperFee 设置 Maker 和 Taker 的手续费率
func WithPaperFee(maker, taker float64) PaperWalletOption {
	return func(wallet *PaperWallet) {
		wallet.makerFee = maker
		wallet.takerFee = taker
	}
}

// WithDataFeed 设置数据源
func WithDataFeed(feeder service.Feeder) PaperWalletOption {
	return func(wallet *PaperWallet) {
		wallet.feeder = feeder
	}
}

// NewPaperWallet 创建一个新的 PaperWallet 实例
func NewPaperWallet(ctx context.Context, baseCoin string, options ...PaperWalletOption) *PaperWallet {
	wallet := PaperWallet{
		ctx:           ctx,
		baseCoin:      baseCoin,
		orders:        make([]model.Order, 0),
		assets:        make(map[string]*assetInfo),
		fistCandle:    make(map[string]model.Candle),
		lastCandle:    make(map[string]model.Candle),
		avgShortPrice: make(map[string]float64),
		avgLongPrice:  make(map[string]float64),
		volume:        make(map[string]float64),
		assetValues:   make(map[string][]AssetValue),
		equityValues:  make([]AssetValue, 0),
	}

	for _, option := range options {
		option(&wallet)
	}

	wallet.initialValue = wallet.assets[wallet.baseCoin].Free
	log.Info("[SETUP] Using paper wallet")
	log.Infof("[SETUP] Initial Portfolio = %f %s", wallet.initialValue, wallet.baseCoin)

	return &wallet
}

// ID 返回一个唯一的订单ID
func (p *PaperWallet) ID() int64 {
	p.counter++
	return p.counter
}

// Pairs 返回钱包中的所有交易对
func (p *PaperWallet) Pairs() []string {
	pairs := make([]string, 0)
	for pair := range p.assets {
		pairs = append(pairs, pair)
	}
	return pairs
}

// LastQuote 返回指定交易对的最新报价
func (p *PaperWallet) LastQuote(ctx context.Context, pair string) (float64, error) {
	return p.feeder.LastQuote(ctx, pair)
}

// AssetValues 返回指定交易对的资产价值列表
func (p *PaperWallet) AssetValues(pair string) []AssetValue {
	return p.assetValues[pair]
}

// EquityValues 返回所有资产的总价值列表
func (p *PaperWallet) EquityValues() []AssetValue {
	return p.equityValues
}

// MaxDrawdown 计算最大回撤
func (p *PaperWallet) MaxDrawdown() (float64, time.Time, time.Time) {
	if len(p.equityValues) < 1 {
		return 0, time.Time{}, time.Time{}
	}

	localMin := math.MaxFloat64
	localMinBase := p.equityValues[0].Value
	localMinStart := p.equityValues[0].Time
	localMinEnd := p.equityValues[0].Time

	globalMin := localMin
	globalMinBase := localMinBase
	globalMinStart := localMinStart
	globalMinEnd := localMinEnd

	for i := 1; i < len(p.equityValues); i++ {
		diff := p.equityValues[i].Value - p.equityValues[i-1].Value

		if localMin > 0 {
			localMin = diff
			localMinBase = p.equityValues[i-1].Value
			localMinStart = p.equityValues[i-1].Time
			localMinEnd = p.equityValues[i].Time
		} else {
			localMin += diff
			localMinEnd = p.equityValues[i].Time
		}

		if localMin < globalMin {
			globalMin = localMin
			globalMinBase = localMinBase
			globalMinStart = localMinStart
			globalMinEnd = localMinEnd
		}
	}

	return globalMin / globalMinBase, globalMinStart, globalMinEnd
}

// Summary 输出钱包的总结信息
func (p *PaperWallet) Summary() {
	var (
		total        float64
		marketChange float64
		volume       float64
	)

	fmt.Println("----- FINAL WALLET -----")
	for pair := range p.lastCandle {
		asset, quote := SplitAssetQuote(pair)
		assetInfo, ok := p.assets[asset]
		if !ok {
			continue
		}

		quantity := assetInfo.Free + assetInfo.Lock
		value := quantity * p.lastCandle[pair].Close
		if quantity < 0 {
			totalShort := 2.0*p.avgShortPrice[pair]*quantity - p.lastCandle[pair].Close*quantity
			value = math.Abs(totalShort)
		}
		total += value
		marketChange += (p.lastCandle[pair].Close - p.fistCandle[pair].Close) / p.fistCandle[pair].Close
		fmt.Printf("%.4f %s = %.4f %s\n", quantity, asset, total, quote)
	}

	avgMarketChange := marketChange / float64(len(p.lastCandle))
	baseCoinValue := p.assets[p.baseCoin].Free + p.assets[p.baseCoin].Lock
	profit := total + baseCoinValue - p.initialValue
	fmt.Printf("%.4f %s\n", baseCoinValue, p.baseCoin)
	fmt.Println()
	maxDrawDown, _, _ := p.MaxDrawdown()
	fmt.Println("----- RETURNS -----")
	fmt.Printf("START PORTFOLIO     = %.2f %s\n", p.initialValue, p.baseCoin)
	fmt.Printf("FINAL PORTFOLIO     = %.2f %s\n", total+baseCoinValue, p.baseCoin)
	fmt.Printf("GROSS PROFIT        =  %f %s (%.2f%%)\n", profit, p.baseCoin, profit/p.initialValue*100)
	fmt.Printf("MARKET CHANGE (B&H) =  %.2f%%\n", avgMarketChange*100)
	fmt.Println()
	fmt.Println("------ RISK -------")
	fmt.Printf("MAX DRAWDOWN = %.2f %%\n", maxDrawDown*100)
	fmt.Println()
	fmt.Println("------ VOLUME -----")
	for pair, vol := range p.volume {
		volume += vol
		fmt.Printf("%s         = %.2f %s\n", pair, vol, p.baseCoin)
	}
	fmt.Printf("TOTAL           = %.2f %s\n", volume, p.baseCoin)
	fmt.Println("-------------------")
}

// validateFunds 验证资金是否足够进行交易
func (p *PaperWallet) validateFunds(side model.SideType, pair string, amount, value float64, fill bool) error {
	asset, quote := SplitAssetQuote(pair)
	if _, ok := p.assets[asset]; !ok {
		p.assets[asset] = &assetInfo{}
	}

	if _, ok := p.assets[quote]; !ok {
		p.assets[quote] = &assetInfo{}
	}

	funds := p.assets[quote].Free
	if side == model.SideTypeSell {
		if p.assets[asset].Free > 0 {
			funds += p.assets[asset].Free * value
		}

		if funds < amount*value {
			return &OrderError{
				Err:      ErrInsufficientFunds,
				Pair:     pair,
				Quantity: amount,
			}
		}

		lockedAsset := math.Min(math.Max(p.assets[asset].Free, 0), amount) // ignore negative asset amount to lock
		lockedQuote := (amount - lockedAsset) * value

		p.assets[asset].Free -= lockedAsset
		p.assets[quote].Free -= lockedQuote
		if fill {
			p.updateAveragePrice(side, pair, amount, value)
			if lockedQuote > 0 { // entering in short position
				p.assets[asset].Free -= amount
			} else { // liquidating long position
				p.assets[quote].Free += amount * value

			}
		} else {
			p.assets[asset].Lock += lockedAsset
			p.assets[quote].Lock += lockedQuote
		}

		log.Debugf("%s -> LOCK = %f / FREE %f", asset, p.assets[asset].Lock, p.assets[asset].Free)
	} else { // SideTypeBuy
		var liquidShortValue float64
		if p.assets[asset].Free < 0 {
			v := math.Abs(p.assets[asset].Free)
			liquidShortValue = 2*v*p.avgShortPrice[pair] - v*value // liquid price of short position
			funds += liquidShortValue
		}

		amountToBuy := amount
		if p.assets[asset].Free < 0 {
			amountToBuy = amount + p.assets[asset].Free
		}

		if funds < amountToBuy*value {
			return &OrderError{
				Err:      ErrInsufficientFunds,
				Pair:     pair,
				Quantity: amount,
			}
		}

		lockedAsset := math.Min(-math.Min(p.assets[asset].Free, 0), amount) // ignore positive amount to lock
		lockedQuote := (amount-lockedAsset)*value - liquidShortValue

		p.assets[asset].Free += lockedAsset
		p.assets[quote].Free -= lockedQuote

		if fill {
			p.updateAveragePrice(side, pair, amount, value)
			p.assets[asset].Free += amount - lockedAsset
		} else {
			p.assets[asset].Lock += lockedAsset
			p.assets[quote].Lock += lockedQuote
		}
		log.Debugf("%s -> LOCK = %f / FREE %f", asset, p.assets[asset].Lock, p.assets[asset].Free)
	}

	return nil
}

// updateAveragePrice 更新平均价格
func (p *PaperWallet) updateAveragePrice(side model.SideType, pair string, amount, value float64) {
	actualQty := 0.0
	asset, quote := SplitAssetQuote(pair)

	if p.assets[asset] != nil {
		actualQty = p.assets[asset].Free
	}

	// without previous position
	if actualQty == 0 {
		if side == model.SideTypeBuy {
			p.avgLongPrice[pair] = value
		} else {
			p.avgShortPrice[pair] = value
		}
		return
	}

	// actual long + order buy
	if actualQty > 0 && side == model.SideTypeBuy {
		positionValue := p.avgLongPrice[pair] * actualQty
		p.avgLongPrice[pair] = (positionValue + amount*value) / (actualQty + amount)
		return
	}

	// actual long + order sell
	if actualQty > 0 && side == model.SideTypeSell {
		profitValue := amount*value - math.Min(amount, actualQty)*p.avgLongPrice[pair]
		percentage := profitValue / (amount * p.avgLongPrice[pair])
		log.Infof("PROFIT = %.4f %s (%.2f %%)", profitValue, quote, percentage*100.0) // TODO: store profits

		if amount <= actualQty { // not enough quantity to close the position
			return
		}

		p.avgShortPrice[pair] = value

		return
	}

	// actual short + order sell
	if actualQty < 0 && side == model.SideTypeSell {
		positionValue := p.avgShortPrice[pair] * -actualQty
		p.avgShortPrice[pair] = (positionValue + amount*value) / (-actualQty + amount)

		return
	}

	// actual short + order buy
	if actualQty < 0 && side == model.SideTypeBuy {
		profitValue := math.Min(amount, -actualQty)*p.avgShortPrice[pair] - amount*value
		percentage := profitValue / (amount * p.avgShortPrice[pair])
		log.Infof("PROFIT = %.4f %s (%.2f %%)", profitValue, quote, percentage*100.0) // TODO: store profits

		if amount <= -actualQty { // not enough quantity to close the position
			return
		}

		p.avgLongPrice[pair] = value
	}
}

// OnCandle 处理蜡烛图更新
func (p *PaperWallet) OnCandle(candle model.Candle) {
	p.Lock()
	defer p.Unlock()

	p.lastCandle[candle.Pair] = candle
	if _, ok := p.fistCandle[candle.Pair]; !ok {
		p.fistCandle[candle.Pair] = candle
	}

	for i, order := range p.orders {
		if order.Pair != candle.Pair || order.Status != model.OrderStatusTypeNew {
			continue
		}

		if _, ok := p.volume[candle.Pair]; !ok {
			p.volume[candle.Pair] = 0
		}

		asset, quote := SplitAssetQuote(order.Pair)
		if order.Side == model.SideTypeBuy && order.Price >= candle.Close {
			if _, ok := p.assets[asset]; !ok {
				p.assets[asset] = &assetInfo{}
			}

			p.volume[candle.Pair] += order.Price * order.Quantity
			p.orders[i].UpdatedAt = candle.Time
			p.orders[i].Status = model.OrderStatusTypeFilled

			// update assets size
			p.updateAveragePrice(order.Side, order.Pair, order.Quantity, order.Price)
			p.assets[asset].Free = p.assets[asset].Free + order.Quantity
			p.assets[quote].Lock = p.assets[quote].Lock - order.Price*order.Quantity
		}

		if order.Side == model.SideTypeSell {
			var orderPrice float64
			if (order.Type == model.OrderTypeLimit ||
				order.Type == model.OrderTypeLimitMaker ||
				order.Type == model.OrderTypeTakeProfit ||
				order.Type == model.OrderTypeTakeProfitLimit) &&
				candle.High >= order.Price {
				orderPrice = order.Price
			} else if (order.Type == model.OrderTypeStopLossLimit ||
				order.Type == model.OrderTypeStopLoss) &&
				candle.Low <= *order.Stop {
				orderPrice = *order.Stop
			} else {
				continue
			}

			// Cancel other orders from same group
			if order.GroupID != nil {
				for j, groupOrder := range p.orders {
					if groupOrder.GroupID != nil && *groupOrder.GroupID == *order.GroupID &&
						groupOrder.ExchangeID != order.ExchangeID {
						p.orders[j].Status = model.OrderStatusTypeCanceled
						p.orders[j].UpdatedAt = candle.Time
						break
					}
				}
			}

			if _, ok := p.assets[quote]; !ok {
				p.assets[quote] = &assetInfo{}
			}

			orderVolume := order.Quantity * orderPrice

			p.volume[candle.Pair] += orderVolume
			p.orders[i].UpdatedAt = candle.Time
			p.orders[i].Status = model.OrderStatusTypeFilled

			// update assets size
			p.updateAveragePrice(order.Side, order.Pair, order.Quantity, orderPrice)
			p.assets[asset].Lock = p.assets[asset].Lock - order.Quantity
			p.assets[quote].Free = p.assets[quote].Free + order.Quantity*orderPrice
		}
	}

	if candle.Complete {
		var total float64
		for asset, info := range p.assets {
			amount := info.Free + info.Lock
			pair := strings.ToUpper(asset + p.baseCoin)
			if amount < 0 {
				v := math.Abs(amount)
				liquid := 2*v*p.avgShortPrice[pair] - v*p.lastCandle[pair].Close
				total += liquid
			} else {
				total += amount * p.lastCandle[pair].Close
			}

			p.assetValues[asset] = append(p.assetValues[asset], AssetValue{
				Time:  candle.Time,
				Value: amount * p.lastCandle[pair].Close,
			})
		}

		baseCoinInfo := p.assets[p.baseCoin]
		p.equityValues = append(p.equityValues, AssetValue{
			Time:  candle.Time,
			Value: total + baseCoinInfo.Lock + baseCoinInfo.Free,
		})
	}
}

// Account 返回钱包的账户信息
func (p *PaperWallet) Account() (model.Account, error) {
	balances := make([]model.Balance, 0)
	for pair, info := range p.assets {
		balances = append(balances, model.Balance{
			Asset: pair,
			Free:  info.Free,
			Lock:  info.Lock,
		})
	}

	return model.Account{
		Balances: balances,
	}, nil
}

// Position 返回指定交易对的持仓信息
func (p *PaperWallet) Position(pair string) (asset, quote float64, err error) {
	p.Lock()
	defer p.Unlock()

	assetTick, quoteTick := SplitAssetQuote(pair)
	acc, err := p.Account()
	if err != nil {
		return 0, 0, err
	}

	assetBalance, quoteBalance := acc.Balance(assetTick, quoteTick)

	return assetBalance.Free + assetBalance.Lock, quoteBalance.Free + quoteBalance.Lock, nil
}

// CreateOrderOCO 创建一个止损止盈订单
func (p *PaperWallet) CreateOrderOCO(side model.SideType, pair string,
	size, price, stop, stopLimit float64) ([]model.Order, error) {
	p.Lock()
	defer p.Unlock()

	if size == 0 {
		return nil, ErrInvalidQuantity
	}

	err := p.validateFunds(side, pair, size, price, false)
	if err != nil {
		return nil, err
	}

	groupID := p.ID()
	limitMaker := model.Order{
		ExchangeID: p.ID(),
		CreatedAt:  p.lastCandle[pair].Time,
		UpdatedAt:  p.lastCandle[pair].Time,
		Pair:       pair,
		Side:       side,
		Type:       model.OrderTypeLimitMaker,
		Status:     model.OrderStatusTypeNew,
		Price:      price,
		Quantity:   size,
		GroupID:    &groupID,
		RefPrice:   p.lastCandle[pair].Close,
	}

	stopOrder := model.Order{
		ExchangeID: p.ID(),
		CreatedAt:  p.lastCandle[pair].Time,
		UpdatedAt:  p.lastCandle[pair].Time,
		Pair:       pair,
		Side:       side,
		Type:       model.OrderTypeStopLoss,
		Status:     model.OrderStatusTypeNew,
		Price:      stopLimit,
		Stop:       &stop,
		Quantity:   size,
		GroupID:    &groupID,
		RefPrice:   p.lastCandle[pair].Close,
	}
	p.orders = append(p.orders, limitMaker, stopOrder)

	return []model.Order{limitMaker, stopOrder}, nil
}

// CreateOrderLimit 创建一个限价订单
func (p *PaperWallet) CreateOrderLimit(side model.SideType, pair string,
	size float64, limit float64) (model.Order, error) {

	p.Lock()
	defer p.Unlock()

	if size == 0 {
		return model.Order{}, ErrInvalidQuantity
	}

	err := p.validateFunds(side, pair, size, limit, false)
	if err != nil {
		return model.Order{}, err
	}
	order := model.Order{
		ExchangeID: p.ID(),
		CreatedAt:  p.lastCandle[pair].Time,
		UpdatedAt:  p.lastCandle[pair].Time,
		Pair:       pair,
		Side:       side,
		Type:       model.OrderTypeLimit,
		Status:     model.OrderStatusTypeNew,
		Price:      limit,
		Quantity:   size,
	}
	p.orders = append(p.orders, order)
	return order, nil
}

// CreateOrderMarket 创建一个市价订单
func (p *PaperWallet) CreateOrderMarket(side model.SideType, pair string, size float64) (model.Order, error) {
	p.Lock()
	defer p.Unlock()

	return p.createOrderMarket(side, pair, size)
}

// CreateOrderStop 创建一个止损订单
func (p *PaperWallet) CreateOrderStop(pair string, size float64, limit float64) (model.Order, error) {
	p.Lock()
	defer p.Unlock()

	if size == 0 {
		return model.Order{}, ErrInvalidQuantity
	}

	err := p.validateFunds(model.SideTypeSell, pair, size, limit, false)
	if err != nil {
		return model.Order{}, err
	}

	order := model.Order{
		ExchangeID: p.ID(),
		CreatedAt:  p.lastCandle[pair].Time,
		UpdatedAt:  p.lastCandle[pair].Time,
		Pair:       pair,
		Side:       model.SideTypeSell,
		Type:       model.OrderTypeStopLossLimit,
		Status:     model.OrderStatusTypeNew,
		Price:      limit,
		Stop:       &limit,
		Quantity:   size,
	}
	p.orders = append(p.orders, order)
	return order, nil
}

// CreateOrderMarket 创建一个市价订单
func (p *PaperWallet) createOrderMarket(side model.SideType, pair string, size float64) (model.Order, error) {
	if size == 0 {
		return model.Order{}, ErrInvalidQuantity
	}

	err := p.validateFunds(side, pair, size, p.lastCandle[pair].Close, true)
	if err != nil {
		return model.Order{}, err
	}

	if _, ok := p.volume[pair]; !ok {
		p.volume[pair] = 0
	}

	p.volume[pair] += p.lastCandle[pair].Close * size

	order := model.Order{
		ExchangeID: p.ID(),
		CreatedAt:  p.lastCandle[pair].Time,
		UpdatedAt:  p.lastCandle[pair].Time,
		Pair:       pair,
		Side:       side,
		Type:       model.OrderTypeMarket,
		Status:     model.OrderStatusTypeFilled,
		Price:      p.lastCandle[pair].Close,
		Quantity:   size,
	}

	p.orders = append(p.orders, order)

	return order, nil
}

// CreateOrderMarketQuote 根据报价创建一个市价订单
func (p *PaperWallet) CreateOrderMarketQuote(side model.SideType, pair string,
	quoteQuantity float64) (model.Order, error) {
	p.Lock()
	defer p.Unlock()

	info := p.AssetsInfo(pair)
	quantity := common.AmountToLotSize(info.StepSize, info.BaseAssetPrecision, quoteQuantity/p.lastCandle[pair].Close)
	return p.createOrderMarket(side, pair, quantity)
}

// Cancel 取消订单
func (p *PaperWallet) Cancel(order model.Order) error {
	p.Lock()
	defer p.Unlock()

	for i, o := range p.orders {
		if o.ExchangeID == order.ExchangeID {
			p.orders[i].Status = model.OrderStatusTypeCanceled
		}
	}
	return nil
}

// Order 返回指定ID的订单信息
func (p *PaperWallet) Order(_ string, id int64) (model.Order, error) {
	for _, order := range p.orders {
		if order.ExchangeID == id {
			return order, nil
		}
	}
	return model.Order{}, errors.New("order not found")
}

// CandlesByPeriod 根据时间段和交易对获取蜡烛图数据
func (p *PaperWallet) CandlesByPeriod(ctx context.Context, pair, period string,
	start, end time.Time) ([]model.Candle, error) {
	return p.feeder.CandlesByPeriod(ctx, pair, period, start, end)
}

// CandlesByLimit 根据数量限制获取蜡烛图数据
func (p *PaperWallet) CandlesByLimit(ctx context.Context, pair, period string, limit int) ([]model.Candle, error) {
	return p.feeder.CandlesByLimit(ctx, pair, period, limit)
}

// CandlesSubscription 订阅蜡烛图数据
func (p *PaperWallet) CandlesSubscription(ctx context.Context, pair, timeframe string) (chan model.Candle, chan error) {
	return p.feeder.CandlesSubscription(ctx, pair, timeframe)
}
