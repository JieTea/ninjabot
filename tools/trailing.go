package tools

// TrailingStop 实现移动止损的工具类
type TrailingStop struct {
	current float64 // 当前价格
	stop    float64 // 止损价格
	active  bool    // 是否处于激活状态
}

// NewTrailingStop 创建一个新的 TrailingStop 实例。
func NewTrailingStop() *TrailingStop {
	return &TrailingStop{}
}

// Start 启动移动止损，设置当前价格和止损价格。
func (t *TrailingStop) Start(current, stop float64) {
	t.stop = stop
	t.current = current
	t.active = true
}

// Stop 停止移动止损。
func (t *TrailingStop) Stop() {
	t.active = false
}

// Active 返回移动止损是否处于激活状态。
func (t TrailingStop) Active() bool {
	return t.active
}

// Update 根据当前价格更新移动止损的状态，并返回是否触发止损。
func (t *TrailingStop) Update(current float64) bool {
	if !t.active {
		return false
	}

	if current > t.current {
		t.stop = t.stop + (current - t.current)
		t.current = current
		return false
	}

	t.current = current
	return current <= t.stop
}
