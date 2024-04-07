package download

import (
	"context"
	"encoding/csv"
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/xhit/go-str2duration/v2"

	"github.com/rodrigo-brito/ninjabot/service"
	"github.com/rodrigo-brito/ninjabot/tools/log"
)

// 下载交易所历史数据的工具，主要用于从交易所获取蜡烛图数据并保存到CSV文件中。

const batchSize = 500

// Downloader 结构体表示一个下载器，用于从交易所下载历史数据
type Downloader struct {
	exchange service.Feeder
}

// NewDownloader 创建一个新的下载器实例
func NewDownloader(exchange service.Feeder) Downloader {
	return Downloader{
		exchange: exchange,
	}
}

// Parameters 结构体用于存储下载参数，包括起始时间和结束时间
type Parameters struct {
	Start time.Time
	End   time.Time
}

// Option 是一个函数类型，用于设置下载参数的选项
type Option func(*Parameters)

// WithInterval 设置下载的时间间隔
func WithInterval(start, end time.Time) Option {
	return func(parameters *Parameters) {
		parameters.Start = start
		parameters.End = end
	}
}

// WithDays 设置下载的时间跨度为几天
func WithDays(days int) Option {
	return func(parameters *Parameters) {
		parameters.Start = time.Now().AddDate(0, 0, -days)
		parameters.End = time.Now()
	}
}

// candlesCount 计算给定时间范围内的蜡烛数量和间隔
func candlesCount(start, end time.Time, timeframe string) (int, time.Duration, error) {
	totalDuration := end.Sub(start)
	interval, err := str2duration.ParseDuration(timeframe)
	if err != nil {
		return 0, 0, err
	}
	return int(totalDuration / interval), interval, nil
}

// Download 实际执行下载操作
func (d Downloader) Download(ctx context.Context, pair, timeframe string, output string, options ...Option) error {
	// 创建CSV文件
	recordFile, err := os.Create(output)
	if err != nil {
		return err
	}

	// 设置默认下载参数
	now := time.Now()
	parameters := &Parameters{
		Start: now.AddDate(0, -1, 0),
		End:   now,
	}

	// 根据选项设置下载参数
	for _, option := range options {
		option(parameters)
	}

	// 将开始时间和结束时间调整到当天的开始和结束
	parameters.Start = time.Date(parameters.Start.Year(), parameters.Start.Month(), parameters.Start.Day(),
		0, 0, 0, 0, time.UTC)

	if now.Sub(parameters.End) > 0 {
		parameters.End = time.Date(parameters.End.Year(), parameters.End.Month(), parameters.End.Day(),
			0, 0, 0, 0, time.UTC)
	} else {
		parameters.End = now
	}

	// 计算总共需要下载的蜡烛数量和间隔
	candlesCount, interval, err := candlesCount(parameters.Start, parameters.End, timeframe)
	if err != nil {
		return err
	}
	candlesCount++

	// 打印下载信息
	log.Infof("Downloading %d candles of %s for %s", candlesCount, timeframe, pair)
	info := d.exchange.AssetsInfo(pair)
	writer := csv.NewWriter(recordFile)

	// 创建进度条
	progressBar := progressbar.Default(int64(candlesCount))
	lostData := 0
	isLastLoop := false

	// 写入CSV文件的表头
	err = writer.Write([]string{
		"time", "open", "close", "low", "high", "volume",
	})
	if err != nil {
		return err
	}

	// 循环下载蜡烛数据
	for begin := parameters.Start; begin.Before(parameters.End); begin = begin.Add(interval * batchSize) {
		end := begin.Add(interval * batchSize)
		if end.Before(parameters.End) {
			end = end.Add(-1 * time.Second)
		} else {
			end = parameters.End
			isLastLoop = true
		}

		candles, err := d.exchange.CandlesByPeriod(ctx, pair, timeframe, begin, end)
		if err != nil {
			return err
		}

		// 将蜡烛数据写入CSV文件
		for _, candle := range candles {
			err := writer.Write(candle.ToSlice(info.QuotePrecision))
			if err != nil {
				return err
			}
		}

		// 更新进度条和丢失数据的统计
		countCandles := len(candles)
		if !isLastLoop {
			lostData += batchSize - countCandles
		}

		if err = progressBar.Add(countCandles); err != nil {
			log.Warnf("update progresbar fail: %s", err.Error())
		}
	}

	// 关闭进度条并输出丢失数据的警告信息
	if err = progressBar.Close(); err != nil {
		log.Warnf("close progresbar fail: %s", err.Error())
	}

	if lostData > 0 {
		log.Warnf("%d missing candles", lostData)
	}

	// 刷新并关闭CSV文件，完成下载
	writer.Flush()
	log.Info("Done!")
	return writer.Error()
}
