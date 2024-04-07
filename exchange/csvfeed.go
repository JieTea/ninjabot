package exchange

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/samber/lo"
	"github.com/xhit/go-str2duration/v2"

	"github.com/rodrigo-brito/ninjabot/model"
)

var ErrInsufficientData = errors.New("insufficient data")

// PairFeed 表示交易对的信息
type PairFeed struct {
	Pair       string // 交易对名称
	File       string // CSV 文件路径
	Timeframe  string // 时间框架
	HeikinAshi bool   // 是否使用平滑后的 HeikinAshi 数据
}

// CSVFeed 管理多个交易对的历史数据
type CSVFeed struct {
	Feeds               map[string]PairFeed       // 交易对到其历史数据的映射
	CandlePairTimeFrame map[string][]model.Candle // 蜡烛图时间框架到对应数据的映射
}

func (c CSVFeed) LastQuote(_ context.Context, _ string) (float64, error) {
	return 0, errors.New("invalid operation")
}

// AssetsInfo 根据交易对名称返回一个默认的资产信息结构体
func (c CSVFeed) AssetsInfo(pair string) model.AssetInfo {
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

// parseHeaders 用于解析 CSV 文件的表头
func parseHeaders(headers []string) (index map[string]int, additional []string, ok bool) {
	headerMap := map[string]int{
		"time": 0, "open": 1, "close": 2, "low": 3, "high": 4, "volume": 5,
	}

	_, err := strconv.Atoi(headers[0])
	if err == nil {
		return headerMap, additional, false
	}

	for index, h := range headers {
		if _, ok := headerMap[h]; !ok {
			additional = append(additional, h)
		}
		headerMap[h] = index
	}

	return headerMap, additional, true
}

// NewCSVFeed 根据给定的时间框架和一组 PairFeed，创建一个 CSVFeed 实例，并从 CSV 文件中读取历史数据
func NewCSVFeed(targetTimeframe string, feeds ...PairFeed) (*CSVFeed, error) {
	csvFeed := &CSVFeed{
		Feeds:               make(map[string]PairFeed),
		CandlePairTimeFrame: make(map[string][]model.Candle),
	}

	for _, feed := range feeds {
		csvFeed.Feeds[feed.Pair] = feed

		csvFile, err := os.Open(feed.File)
		if err != nil {
			return nil, err
		}

		csvLines, err := csv.NewReader(csvFile).ReadAll()
		if err != nil {
			return nil, err
		}

		var candles []model.Candle
		ha := model.NewHeikinAshi()

		// map each header label with its index
		headerMap, additionalHeaders, hasCustomHeaders := parseHeaders(csvLines[0])
		if hasCustomHeaders {
			csvLines = csvLines[1:]
		}

		for _, line := range csvLines {
			timestamp, err := strconv.Atoi(line[headerMap["time"]])
			if err != nil {
				return nil, err
			}

			candle := model.Candle{
				Time:      time.Unix(int64(timestamp), 0).UTC(),
				UpdatedAt: time.Unix(int64(timestamp), 0).UTC(),
				Pair:      feed.Pair,
				Complete:  true,
			}

			candle.Open, err = strconv.ParseFloat(line[headerMap["open"]], 64)
			if err != nil {
				return nil, err
			}

			candle.Close, err = strconv.ParseFloat(line[headerMap["close"]], 64)
			if err != nil {
				return nil, err
			}

			candle.Low, err = strconv.ParseFloat(line[headerMap["low"]], 64)
			if err != nil {
				return nil, err
			}

			candle.High, err = strconv.ParseFloat(line[headerMap["high"]], 64)
			if err != nil {
				return nil, err
			}

			candle.Volume, err = strconv.ParseFloat(line[headerMap["volume"]], 64)
			if err != nil {
				return nil, err
			}

			if hasCustomHeaders {
				candle.Metadata = make(map[string]float64)
				for _, header := range additionalHeaders {
					candle.Metadata[header], err = strconv.ParseFloat(line[headerMap[header]], 64)
					if err != nil {
						return nil, err
					}
				}
			}

			if feed.HeikinAshi {
				candle = candle.ToHeikinAshi(ha)
			}

			candles = append(candles, candle)
		}

		csvFeed.CandlePairTimeFrame[csvFeed.feedTimeframeKey(feed.Pair, feed.Timeframe)] = candles

		err = csvFeed.resample(feed.Pair, feed.Timeframe, targetTimeframe)
		if err != nil {
			return nil, err
		}
	}

	return csvFeed, nil
}

// feedTimeframeKey 生成用于唯一标识交易对和时间框架的键
func (c CSVFeed) feedTimeframeKey(pair, timeframe string) string {
	return fmt.Sprintf("%s--%s", pair, timeframe)
}

// Limit 限制历史数据的时间范围
func (c *CSVFeed) Limit(duration time.Duration) *CSVFeed {
	for pair, candles := range c.CandlePairTimeFrame {
		start := candles[len(candles)-1].Time.Add(-duration)
		c.CandlePairTimeFrame[pair] = lo.Filter(candles, func(candle model.Candle, _ int) bool {
			return candle.Time.After(start)
		})
	}
	return c
}

// isFirstCandlePeriod 和 isLastCandlePeriod 用于检查给定时间是否为指定时间框架的第一个或最后一个时间段
func isFirstCandlePeriod(t time.Time, fromTimeframe, targetTimeframe string) (bool, error) {
	fmt.Println("psram: ", t.String(), fromTimeframe, targetTimeframe)
	fromDuration, err := str2duration.ParseDuration(fromTimeframe)
	if err != nil {
		return false, err
	}

	prev := t.Add(-fromDuration).UTC()

	fmt.Println("prev: ", prev.String(), fromDuration)

	return isLastCandlePeriod(prev, fromTimeframe, targetTimeframe)
}

// isLastCandlePeriod 判断给定时间点是否是指定时间框架的最后一个蜡烛图周期
// t: 要检查的时间点
// fromTimeframe: 当前时间框架，例如："1m"、"5m"、"1h"等
// targetTimeframe: 目标时间框架，需要确定给定时间点是否是其最后一个周期
// 返回值:
//   - bool: 给定时间点是否是目标时间框架的最后一个周期
//   - error: 如果时间框架无效，则返回错误
func isLastCandlePeriod(t time.Time, fromTimeframe, targetTimeframe string) (bool, error) {
	// 如果当前时间框架与目标时间框架相同，则认为是最后一个周期
	if fromTimeframe == targetTimeframe {
		return true, nil
	}

	// 将当前时间框架解析为持续时间
	fromDuration, err := str2duration.ParseDuration(fromTimeframe)
	if err != nil {
		return false, err
	}

	// 计算当前时间点的下一个时间点
	next := t.Add(fromDuration).UTC()

	// 根据目标时间框架进行判断
	switch targetTimeframe {
	case "1m":
		return next.Second()%60 == 0, nil
	case "5m":
		return next.Minute()%5 == 0, nil
	case "10m":
		return next.Minute()%10 == 0, nil
	case "15m":
		return next.Minute()%15 == 0, nil
	case "30m":
		return next.Minute()%30 == 0, nil
	case "1h":
		return next.Minute()%60 == 0, nil
	case "2h":
		return next.Minute() == 0 && next.Hour()%2 == 0, nil
	case "4h":
		return next.Minute() == 0 && next.Hour()%4 == 0, nil
	case "12h":
		return next.Minute() == 0 && next.Hour()%12 == 0, nil
	case "1d":
		return next.Minute() == 0 && next.Hour()%24 == 0, nil
	case "1w":
		return next.Minute() == 0 && next.Hour()%24 == 0 && next.Weekday() == time.Sunday, nil
	}

	return false, fmt.Errorf("invalid timeframe: %s", targetTimeframe)
}

// resample 根据目标时间框架重新采样历史数据
func (c *CSVFeed) resample(pair, sourceTimeframe, targetTimeframe string) error {
	sourceKey := c.feedTimeframeKey(pair, sourceTimeframe)
	targetKey := c.feedTimeframeKey(pair, targetTimeframe)

	var i int
	for ; i < len(c.CandlePairTimeFrame[sourceKey]); i++ {
		if ok, err := isFirstCandlePeriod(c.CandlePairTimeFrame[sourceKey][i].Time, sourceTimeframe,
			targetTimeframe); err != nil {
			return err
		} else if ok {
			break
		}
	}

	candles := make([]model.Candle, 0)
	for ; i < len(c.CandlePairTimeFrame[sourceKey]); i++ {
		candle := c.CandlePairTimeFrame[sourceKey][i]
		if last, err := isLastCandlePeriod(candle.Time, sourceTimeframe, targetTimeframe); err != nil {
			return err
		} else if last {
			candle.Complete = true
		} else {
			candle.Complete = false
		}

		lastIndex := len(candles) - 1
		if lastIndex >= 0 && !candles[lastIndex].Complete {
			candle.Time = candles[lastIndex].Time
			candle.Open = candles[lastIndex].Open
			candle.High = math.Max(candles[lastIndex].High, candle.High)
			candle.Low = math.Min(candles[lastIndex].Low, candle.Low)
			candle.Volume += candles[lastIndex].Volume
		}
		candles = append(candles, candle)
	}

	// remove last candle if not complete
	if !candles[len(candles)-1].Complete {
		candles = candles[:len(candles)-1]
	}

	c.CandlePairTimeFrame[targetKey] = candles

	return nil
}

// CandlesByPeriod 根据指定的时间范围和时间框架，返回历史蜡烛数据
func (c CSVFeed) CandlesByPeriod(_ context.Context, pair, timeframe string,
	start, end time.Time) ([]model.Candle, error) {

	key := c.feedTimeframeKey(pair, timeframe)
	candles := make([]model.Candle, 0)
	for _, candle := range c.CandlePairTimeFrame[key] {
		if candle.Time.Before(start) || candle.Time.After(end) {
			continue
		}
		candles = append(candles, candle)
	}
	return candles, nil
}

// CandlesByLimit 根据指定的时间框架和数量限制，返回历史蜡烛数据
func (c *CSVFeed) CandlesByLimit(_ context.Context, pair, timeframe string, limit int) ([]model.Candle, error) {
	var result []model.Candle
	key := c.feedTimeframeKey(pair, timeframe)
	if len(c.CandlePairTimeFrame[key]) < limit {
		return nil, fmt.Errorf("%w: %s", ErrInsufficientData, pair)
	}
	result, c.CandlePairTimeFrame[key] = c.CandlePairTimeFrame[key][:limit], c.CandlePairTimeFrame[key][limit:]
	return result, nil
}

// CandlesSubscription 返回一个通道，用于订阅指定时间框架下的历史蜡烛数据
func (c CSVFeed) CandlesSubscription(_ context.Context, pair, timeframe string) (chan model.Candle, chan error) {
	ccandle := make(chan model.Candle)
	cerr := make(chan error)
	key := c.feedTimeframeKey(pair, timeframe)
	go func() {
		for _, candle := range c.CandlePairTimeFrame[key] {
			ccandle <- candle
		}
		close(ccandle)
		close(cerr)
	}()
	return ccandle, cerr
}
