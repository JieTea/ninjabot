package indicator

import "github.com/markcheno/go-talib"

// SuperTrend 计算超级趋势指标
// 根据给定的最高价、最低价和收盘价数据，以及 ATR 计算周期和因子，计算出相应的超级趋势指标值。
// 输入参数：
//   - high: 最高价数据
//   - low: 最低价数据
//   - close: 收盘价数据
//   - atrPeriod: ATR 计算周期
//   - factor: ATR 倍数因子
//
// 返回值：
//   - []float64: 超级趋势指标值
func SuperTrend(high, low, close []float64, atrPeriod int, factor float64) []float64 {
	// talib.Atr 计算平均真实波幅（ATR）
	atr := talib.Atr(high, low, close, atrPeriod)
	basicUpperBand := make([]float64, len(atr))
	basicLowerBand := make([]float64, len(atr))
	finalUpperBand := make([]float64, len(atr))
	finalLowerBand := make([]float64, len(atr))
	superTrend := make([]float64, len(atr))

	// 根据 ATR 和因子计算基本上下线
	for i := 1; i < len(basicLowerBand); i++ {
		basicUpperBand[i] = (high[i]+low[i])/2.0 + atr[i]*factor
		basicLowerBand[i] = (high[i]+low[i])/2.0 - atr[i]*factor

		// 根据基本上下线计算最终上下线
		if basicUpperBand[i] < finalUpperBand[i-1] ||
			close[i-1] > finalUpperBand[i-1] {
			finalUpperBand[i] = basicUpperBand[i]
		} else {
			finalUpperBand[i] = finalUpperBand[i-1]
		}

		if basicLowerBand[i] > finalLowerBand[i-1] ||
			close[i-1] < finalLowerBand[i-1] {
			finalLowerBand[i] = basicLowerBand[i]
		} else {
			finalLowerBand[i] = finalLowerBand[i-1]
		}

		// 根据最终上下线和收盘价计算超级趋势指标值
		if finalUpperBand[i-1] == superTrend[i-1] {
			if close[i] > finalUpperBand[i] {
				superTrend[i] = finalLowerBand[i]
			} else {
				superTrend[i] = finalUpperBand[i]
			}
		} else {
			if close[i] < finalLowerBand[i] {
				superTrend[i] = finalUpperBand[i]
			} else {
				superTrend[i] = finalLowerBand[i]
			}
		}
	}

	return superTrend
}
