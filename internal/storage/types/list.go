package types

// List 是一个双向链表，用 slice 模拟
type List []string

// NewList 创建一个新的 List
func NewList() List {
	return make(List, 0)
}

// LPush 从左边插入一个或多个元素
// 返回值：插入后列表的长度
// 注意：LPUSH key a b c => 列表为 [c, b, a]
func (l *List) LPush(values ...string) int {
	if len(values) == 0 {
		return len(*l)
	}
	// 一次性构建：先反转 values，再整体 append
	// 这样只需要一次复制，时间复杂度 O(n)
	for i, j := 0, len(values)-1; i < j; i, j = i+1, j-1 {
		values[i], values[j] = values[j], values[i]
	}
	*l = append(values, *l...)
	return len(*l)
}

// RPush 从右边插入一个或多个元素
// 返回值：插入后列表的长度
func (l *List) RPush(values ...string) int {
	*l = append(*l, values...)
	return len(*l)
}

// LPop 从左边弹出一个元素
// 返回值：弹出的元素和是否成功
func (l *List) LPop() (string, bool) {
	if len(*l) == 0 {
		return "", false
	}
	val := (*l)[0]
	*l = (*l)[1:]
	return val, true
}

// RPop 从右边弹出一个元素
// 返回值：弹出的元素和是否成功
func (l *List) RPop() (string, bool) {
	if len(*l) == 0 {
		return "", false
	}
	idx := len(*l) - 1
	val := (*l)[idx]
	*l = (*l)[:idx]
	return val, true
}

// LRange 返回指定区间的元素（包含 start 和 stop）
// 支持负数索引：-1 表示最后一个元素
// 例如：LRANGE 0 -1 返回全部元素
func (l List) LRange(start, stop int) []string {
	length := len(l)
	if length == 0 {
		return []string{}
	}

	// 处理负数索引
	if start < 0 {
		start = length + start
	}
	if stop < 0 {
		stop = length + stop
	}

	// 边界检查
	if start < 0 {
		start = 0
	}
	if stop >= length {
		stop = length - 1
	}
	if start > stop {
		return []string{}
	}

	return l[start : stop+1]
}

// LLen 返回列表长度
func (l List) LLen() int {
	return len(l)
}

// LIndex 返回指定索引的元素（支持负数）
// 返回值：元素和是否存在
func (l List) LIndex(index int) (string, bool) {
	length := len(l)
	if length == 0 {
		return "", false
	}

	// 处理负数索引
	if index < 0 {
		index = length + index
	}

	// 检查边界
	if index < 0 || index >= length {
		return "", false
	}

	return l[index], true
}
