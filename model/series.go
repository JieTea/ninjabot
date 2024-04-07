package model

import (
	"strconv"
	"strings"

	"golang.org/x/exp/constraints"
)

// Series 类型，用于表示一系列时间序列的值。该类型使用了泛型，可以存储任何有序类型的数据。
// Series is a time series of values
type Series[T constraints.Ordered] []T

// Values returns the values of the series
// 返回时间序列的所有值
func (s Series[T]) Values() []T {
	return s
}

// Length returns the number of values in the series
// 返回时间序列的长度（即值的个数）
func (s Series[T]) Length() int {
	return len(s)
}

// Last returns the last value of the series given a past index position
// 返回时间序列倒数第 position 个位置的值
func (s Series[T]) Last(position int) T {
	return s[len(s)-1-position]
}

// LastValues returns the last values of the series given a size
// 返回时间序列最后 size 个值
func (s Series[T]) LastValues(size int) []T {
	if l := len(s); l > size {
		return s[l-size:]
	}
	return s
}

// Crossover returns true if the last value of the series is greater than the last value of the reference series
// 判断时间序列的最后一个值是否大于参考序列的最后一个值，并且前一个值小于等于参考序列的前一个值，则返回 true
func (s Series[T]) Crossover(ref Series[T]) bool {
	return s.Last(0) > ref.Last(0) && s.Last(1) <= ref.Last(1)
}

// Crossunder returns true if the last value of the series is less than the last value of the reference series
// 判断时间序列的最后一个值是否小于等于参考序列的最后一个值，并且前一个值大于参考序列的前一个值，则返回 true
func (s Series[T]) Crossunder(ref Series[T]) bool {
	return s.Last(0) <= ref.Last(0) && s.Last(1) > ref.Last(1)
}

// Cross returns true if the last value of the series is greater than the last value of the
// reference series or less than the last value of the reference series
// 判断时间序列的最后一个值是否大于参考序列的最后一个值，或者小于等于参考序列的最后一个值。则 Cross 返回 true
func (s Series[T]) Cross(ref Series[T]) bool {
	return s.Crossover(ref) || s.Crossunder(ref)
}

// NumDecPlaces returns the number of decimal places of a float64
// NumDecPlaces 用于计算一个浮点数的小数位数。
// 这个函数会将浮点数转换为字符串，并找到小数点的位置，从而确定小数位数。
func NumDecPlaces(v float64) int64 {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	i := strings.IndexByte(s, '.')
	if i > -1 {
		return int64(len(s) - i - 1)
	}
	return 0
}
