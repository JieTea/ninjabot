package model

import "sync"

type PriorityQueue struct {
	sync.Mutex
	length          int
	data            []Item
	notifyCallbacks []func(Item)
}

type Item interface {
	Less(Item) bool
}

// NewPriorityQueue 用给定的切片数据创建一个新的优先队列，
// 并根据数据调整队列，以满足优先级队列的性质。
func NewPriorityQueue(data []Item) *PriorityQueue {
	q := &PriorityQueue{}
	q.data = data
	q.length = len(data)
	// 调整队列以满足优先级队列的性质
	if q.length > 0 {
		i := q.length >> 1
		for ; i >= 0; i-- {
			q.down(i)
		}
	}
	return q
}

// Push 向优先队列中添加一个元素，并根据元素的优先级调整队列。
func (q *PriorityQueue) Push(item Item) {
	q.Lock()
	defer q.Unlock()

	// 添加元素到队列末尾
	q.data = append(q.data, item)
	q.length++
	// 调整队列以满足优先级队列的性质
	q.up(q.length - 1)

	// 通知所有注册的回调函数
	for _, notify := range q.notifyCallbacks {
		go notify(item)
	}
}

// PopLock 返回一个只读 channel，用于安全地从优先队列中弹出元素。
func (q *PriorityQueue) PopLock() <-chan Item {
	ch := make(chan Item)
	// 注册一个回调函数，将弹出的元素发送到 channel 中
	q.notifyCallbacks = append(q.notifyCallbacks, func(_ Item) {
		ch <- q.Pop()
	})
	return ch
}

// Pop 从优先队列中弹出优先级最高的元素，并返回该元素。
func (q *PriorityQueue) Pop() Item {
	q.Lock()
	defer q.Unlock()

	if q.length == 0 {
		return nil
	}
	top := q.data[0]
	q.length--
	if q.length > 0 {
		q.data[0] = q.data[q.length]
		// 调整队列以满足优先级队列的性质
		q.down(0)
	}
	q.data = q.data[:len(q.data)-1]
	return top
}

// Peek 返回优先队列中优先级最高的元素，但不将其从队列中移除。
func (q *PriorityQueue) Peek() Item {
	q.Lock()
	defer q.Unlock()

	if q.length == 0 {
		return nil
	}
	return q.data[0]
}

// Len 返回优先队列中元素的数量。
func (q *PriorityQueue) Len() int {
	q.Lock()
	defer q.Unlock()

	return q.length
}

// down 方法用于将位于 pos 位置的元素下沉到合适的位置，以维护优先级队列的性质。
func (q *PriorityQueue) down(pos int) {
	data := q.data
	// 计算非叶子节点的数量
	halfLength := q.length >> 1
	// 暂存待下沉的元素
	item := data[pos]
	// 循环直到当前节点已经是叶子节点为止
	for pos < halfLength {
		// 计算左子节点位置
		left := (pos << 1) + 1
		// 计算右子节点位置
		right := left + 1
		// 获取左右子节点中值较小的节点作为 best
		best := data[left]
		// 如果右子节点存在且右子节点的值小于 best 的值，则更新 best 和 left
		if right < q.length && data[right].Less(best) {
			left = right
			best = data[right]
		}
		// 如果 best 的值不小于 item 的值，则退出循环
		if !best.Less(item) {
			break
		}
		// 将 best 上浮到当前位置 pos，并更新 pos 为 left
		data[pos] = best
		pos = left
	}
	// 将最初暂存的 item 放置在最终确定的位置上
	data[pos] = item
}

// 将最后一个元素（通常是新插入的元素）上浮到合适的位置，以满足优先级队列的性质。
func (q *PriorityQueue) up(pos int) {
	data := q.data
	item := data[pos] // 待上浮的元素暂存为 item
	for pos > 0 {     // 循环: 直到当前元素的位置不是根节点（即位置不是0）
		parent := (pos - 1) >> 1 // 计算当前节点的父节点位置
		current := data[parent]  // 获取父节点的值
		// 如果 item 的值大于 current 的值，，停止上浮跳出循环
		if !item.Less(current) {
			break
		}
		// 如果 item 的值小于 current 的值，将 current 下沉到当前位置
		data[pos] = current
		pos = parent
	}
	// 将 item 放置在最终确定的位置上。
	data[pos] = item
}
