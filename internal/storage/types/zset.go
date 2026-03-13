package types

import "sort"

// ZSetMember 有序集合的成员
type ZSetMember struct {
	Member string
	Score  float64
}

// ZSet 是有序集合，需要同时支持按 member 查 score 和按 score 排序
type ZSet struct {
	dict   map[string]float64 // member -> score，用于快速查 score
	sorted []ZSetMember       // 按 score 排序的列表，用于范围查询
}

// NewZSet 创建一个新的 ZSet
func NewZSet() *ZSet {
	return &ZSet{
		dict:   make(map[string]float64),
		sorted: make([]ZSetMember, 0),
	}
}

// ZAdd 添加成员，如果 member 已存在则更新 score
// 返回值：1 表示新增，0 表示更新
func (z *ZSet) ZAdd(score float64, member string) int {
	oldScore, exists := z.dict[member]
	z.dict[member] = score

	if exists {
		// 更新：先删除旧位置
		z.removeFromSorted(oldScore, member)
	}
	// 插入新位置（保持有序）
	z.insertToSorted(score, member)

	if exists {
		return 0
	}
	return 1
}

// insertToSorted 将成员插入到 sorted 的正确位置（保持有序）
func (z *ZSet) insertToSorted(score float64, member string) {
	// 二分查找插入位置
	idx := sort.Search(len(z.sorted), func(i int) bool {
		if z.sorted[i].Score == score {
			return z.sorted[i].Member >= member // score相同按member字典序
		}
		return z.sorted[i].Score >= score
	})

	// 插入
	z.sorted = append(z.sorted, ZSetMember{})
	copy(z.sorted[idx+1:], z.sorted[idx:])
	z.sorted[idx] = ZSetMember{Member: member, Score: score}
}

// removeFromSorted 从 sorted 中移除指定 score 和 member 的元素
func (z *ZSet) removeFromSorted(score float64, member string) {
	// 二分查找定位到 score
	idx := sort.Search(len(z.sorted), func(i int) bool {
		return z.sorted[i].Score >= score
	})

	// 在 score 相同的范围内找 member
	for idx < len(z.sorted) && z.sorted[idx].Score == score {
		if z.sorted[idx].Member == member {
			z.sorted = append(z.sorted[:idx], z.sorted[idx+1:]...)
			return
		}
		idx++
	}
}

// ZRem 移除一个或多个成员
// 返回值：成功移除的数量
func (z *ZSet) ZRem(members ...string) int {
	count := 0
	for _, member := range members {
		if score, ok := z.dict[member]; ok {
			z.removeFromSorted(score, member)
			delete(z.dict, member)
			count++
		}
	}
	return count
}

// ZScore 返回成员的 score
// 返回值：score 和是否存在
func (z *ZSet) ZScore(member string) (float64, bool) {
	score, ok := z.dict[member]
	return score, ok
}

// ZRank 返回成员的排名（从 0 开始，按 score 升序）
// 返回值：排名和是否存在
func (z *ZSet) ZRank(member string) (int, bool) {
	score, ok := z.dict[member]
	if !ok {
		return 0, false
	}

	// 在 sorted 中找到位置
	idx := sort.Search(len(z.sorted), func(i int) bool {
		if z.sorted[i].Score == score {
			return z.sorted[i].Member >= member
		}
		return z.sorted[i].Score >= score
	})

	// 确认找到
	if idx < len(z.sorted) && z.sorted[idx].Member == member {
		return idx, true
	}
	return 0, false
}

// ZRevRank 返回成员的倒序排名（从 0 开始，score 大的排名小）
func (z *ZSet) ZRevRank(member string) (int, bool) {
	rank, ok := z.ZRank(member)
	if !ok {
		return 0, false
	}
	return len(z.sorted) - 1 - rank, true
}

// ZRange 按排名范围返回成员（包含 start 和 stop）
// 支持负数：-1 表示最后一个
// 返回值：成员列表（带 score）
func (z *ZSet) ZRange(start, stop int) []ZSetMember {
	length := len(z.sorted)
	if length == 0 {
		return []ZSetMember{}
	}

	// 处理负数
	if start < 0 {
		start = length + start
	}
	if stop < 0 {
		stop = length + stop
	}

	// 边界
	if start < 0 {
		start = 0
	}
	if stop >= length {
		stop = length - 1
	}
	if start > stop {
		return []ZSetMember{}
	}

	return z.sorted[start : stop+1]
}

// ZRevRange 按排名范围返回成员（倒序）
func (z *ZSet) ZRevRange(start, stop int) []ZSetMember {
	length := len(z.sorted)
	if length == 0 {
		return []ZSetMember{}
	}

	// 转换倒序索引为正序索引
	if start < 0 {
		start = length + start
	}
	if stop < 0 {
		stop = length + stop
	}

	// 转换为正序的范围
	newStart := length - 1 - stop
	newStop := length - 1 - start

	if newStart < 0 {
		newStart = 0
	}
	if newStop >= length {
		newStop = length - 1
	}
	if newStart > newStop {
		return []ZSetMember{}
	}

	// 倒序输出
	result := z.sorted[newStart : newStop+1]
	reversed := make([]ZSetMember, len(result))
	for i, m := range result {
		reversed[len(result)-1-i] = m
	}
	return reversed
}

// ZRangeByScore 按 score 范围返回成员 [min, max]
func (z *ZSet) ZRangeByScore(min, max float64) []ZSetMember {
	if len(z.sorted) == 0 {
		return []ZSetMember{}
	}

	// 找到起始位置
	start := sort.Search(len(z.sorted), func(i int) bool {
		return z.sorted[i].Score >= min
	})

	// 找到结束位置
	stop := sort.Search(len(z.sorted), func(i int) bool {
		return z.sorted[i].Score > max
	})

	if start >= stop {
		return []ZSetMember{}
	}
	return z.sorted[start:stop]
}

// ZCard 返回成员数量
func (z *ZSet) ZCard() int {
	return len(z.dict)
}

// Members 返回所有成员（用于内存估算）
func (z *ZSet) Members() []string {
	members := make([]string, 0, len(z.dict))
	for m := range z.dict {
		members = append(members, m)
	}
	return members
}

// ZCount 返回 score 在 [min, max] 范围内的成员数量
func (z *ZSet) ZCount(min, max float64) int {
	return len(z.ZRangeByScore(min, max))
}

// ZIncrBy 将成员的 score 增加 delta
// 返回值：新的 score
func (z *ZSet) ZIncrBy(member string, delta float64) (float64, error) {
	score, exists := z.dict[member]
	if !exists {
		// 不存在则添加
		z.ZAdd(delta, member)
		return delta, nil
	}

	newScore := score + delta
	z.ZAdd(newScore, member)
	return newScore, nil
}
