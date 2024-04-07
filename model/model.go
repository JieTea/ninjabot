package model

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

// TelegramSettings 电报设置
type TelegramSettings struct {
	Enabled bool   // 启用状态
	Token   string // 令牌
	Users   []int  // 用户
}

// Settings 设置
type Settings struct {
	Pairs    []string         // 交易对
	Telegram TelegramSettings // 电报设置
}

// Balance 余额
type Balance struct {
	Asset    string  // 资产
	Free     float64 // 可用
	Lock     float64 // 锁定
	Leverage float64 // 杠杆
}

// AssetInfo 资产信息
type AssetInfo struct {
	BaseAsset  string //基础资产
	QuoteAsset string // 报价资产(btc/USDT中的usdt)

	MinPrice    float64 // 最低价格
	MaxPrice    float64 // 最高价格
	MinQuantity float64 // 最小数量
	MaxQuantity float64 // 最大数量
	StepSize    float64 // 步长:交易数量的最小变动单位
	TickSize    float64 // 刻度:价格变动的最小单位

	QuotePrecision     int // 报价精度
	BaseAssetPrecision int // 基础资产精度
}

// Dataframe 数据帧
type Dataframe struct {
	// 交易对
	Pair string

	Close  Series[float64] // 收盘价序列
	Open   Series[float64] // 开盘价序列
	High   Series[float64] // 最高价序列
	Low    Series[float64] // 最低价序列
	Volume Series[float64] // 交易量序列

	Time       []time.Time // 时间戳序列
	LastUpdate time.Time   // 最后更新时间

	// 用户元数据，可以存储与数据点相关的任意附加信息
	// Custom user metadata
	Metadata map[string]Series[float64]
}

// Sample 从指定位置开始返回包含原始数据子集的新 Dataframe。
func (df Dataframe) Sample(positions int) Dataframe {
	size := len(df.Time)
	start := size - positions
	if start <= 0 {
		return df
	}

	// 创建一个新的 Dataframe，包含指定位置开始的一部分数据。
	sample := Dataframe{
		Pair:       df.Pair,
		Close:      df.Close.LastValues(positions),
		Open:       df.Open.LastValues(positions),
		High:       df.High.LastValues(positions),
		Low:        df.Low.LastValues(positions),
		Volume:     df.Volume.LastValues(positions),
		Time:       df.Time[start:],
		LastUpdate: df.LastUpdate,
		Metadata:   make(map[string]Series[float64]),
	}

	// 复制原始数据的元数据，仅包含指定位置开始的一部分数据。
	for key := range df.Metadata {
		sample.Metadata[key] = df.Metadata[key].LastValues(positions)
	}

	return sample
}

// Candle 表示一个K线数据。
type Candle struct {
	Pair      string    // 交易对
	Time      time.Time // 时间
	UpdatedAt time.Time // 更新时间
	Open      float64   // 开盘价
	Close     float64   // 收盘价
	Low       float64   // 最低价
	High      float64   // 最高价
	Volume    float64   // 成交量
	Complete  bool      // 是否完整

	// 来自CSV输入的附加列
	// Aditional collums from CSV inputs
	Metadata map[string]float64 // 元数据
}

// Empty 判断该K线是否为空
func (c Candle) Empty() bool {
	return c.Pair == "" && c.Close == 0 && c.Open == 0 && c.Volume == 0
}

// HeikinAshi 表示一个平均柱（平均柱图）
type HeikinAshi struct {
	PreviousHACandle Candle // 前一个平均柱
}

// NewHeikinAshi 创建一个新的平均柱。
func NewHeikinAshi() *HeikinAshi {
	return &HeikinAshi{}
}

// ToSlice 将平均柱转换为字符串切片。
func (c Candle) ToSlice(precision int) []string {
	return []string{
		fmt.Sprintf("%d", c.Time.Unix()),
		strconv.FormatFloat(c.Open, 'f', precision, 64),
		strconv.FormatFloat(c.Close, 'f', precision, 64),
		strconv.FormatFloat(c.Low, 'f', precision, 64),
		strconv.FormatFloat(c.High, 'f', precision, 64),
		strconv.FormatFloat(c.Volume, 'f', precision, 64),
	}
}

// ToHeikinAshi 将K线转换为平均柱
func (c Candle) ToHeikinAshi(ha *HeikinAshi) Candle {
	haCandle := ha.CalculateHeikinAshi(c)

	return Candle{
		Pair:      c.Pair,
		Open:      haCandle.Open,
		High:      haCandle.High,
		Low:       haCandle.Low,
		Close:     haCandle.Close,
		Volume:    c.Volume,
		Complete:  c.Complete,
		Time:      c.Time,
		UpdatedAt: c.UpdatedAt,
	}
}

// Less 判断当前K线是否比另一个Item（K线）更小。
func (c Candle) Less(j Item) bool {
	diff := j.(Candle).Time.Sub(c.Time)
	if diff < 0 {
		return false
	}
	if diff > 0 {
		return true
	}

	diff = j.(Candle).UpdatedAt.Sub(c.UpdatedAt)
	if diff < 0 {
		return false
	}
	if diff > 0 {
		return true
	}

	return c.Pair < j.(Candle).Pair
}

// Account 表示一个账户
type Account struct {
	Balances []Balance
}

// Balance 返回指定资产的余额。
func (a Account) Balance(assetTick, quoteTick string) (Balance, Balance) {
	var assetBalance, quoteBalance Balance
	var isSetAsset, isSetQuote bool

	for _, balance := range a.Balances {
		switch balance.Asset {
		case assetTick:
			assetBalance = balance
			isSetAsset = true
		case quoteTick:
			quoteBalance = balance
			isSetQuote = true
		}

		if isSetAsset && isSetQuote {
			break
		}
	}

	return assetBalance, quoteBalance
}

// Equity 计算账户的净值
func (a Account) Equity() float64 {
	var total float64

	for _, balance := range a.Balances {
		total += balance.Free
		total += balance.Lock
	}

	return total
}

// CalculateHeikinAshi 计算平均柱。
func (ha *HeikinAshi) CalculateHeikinAshi(c Candle) Candle {
	var hkCandle Candle

	openValue := ha.PreviousHACandle.Open
	closeValue := ha.PreviousHACandle.Close

	// 第一个平均柱使用当前K线计算
	// First HA candle is calculated using current candle
	if ha.PreviousHACandle.Empty() {
		openValue = c.Open
		closeValue = c.Close
	}

	hkCandle.Open = (openValue + closeValue) / 2
	hkCandle.Close = (c.Open + c.High + c.Low + c.Close) / 4
	hkCandle.High = math.Max(c.High, math.Max(hkCandle.Open, hkCandle.Close))
	hkCandle.Low = math.Min(c.Low, math.Min(hkCandle.Open, hkCandle.Close))
	ha.PreviousHACandle = hkCandle

	return hkCandle
}
