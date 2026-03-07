package types

// Hash 是一个 field -> value 的映射
type Hash map[string]string

// NewHash 创建一个新的 Hash
func NewHash() Hash {
	return make(Hash)
}

// HSet 设置 field 的值
// 返回值：1 表示新增 field，0 表示更新已有 field
func (h Hash) HSet(field, value string) int {
	_, exists := h[field]
	h[field] = value // 无论是否存在都要赋值
	if exists {
		return 0
	}
	return 1
}

// HGet 获取 field 的值
// 返回值：value 和是否存在
func (h Hash) HGet(field string) (string, bool) {
	if value, ok := h[field]; ok {
		return value, true
	}
	return "", false
}

// HDel 删除一个或多个 field
// 返回值：成功删除的 field 数量
func (h Hash) HDel(fields ...string) int {
	// 遍历 fields，删除存在的 field，计数
	count := 0
	for _, field := range fields {
		if _, ok := h[field]; ok {
			delete(h, field)
			count++
		}
	}
	return count
}

// HExists 判断 field 是否存在
func (h Hash) HExists(field string) bool {
	_, ok := h[field]
	return ok
}

// HLen 返回 field 的数量
func (h Hash) HLen() int {
	return len(h)

}

// HKeys 返回所有 field 的列表
func (h Hash) HKeys() []string {
	// 遍历 map，收集所有 key
	keys := make([]string, 0, len(h))
	for key := range h {
		keys = append(keys, key)
	}
	return keys

}

// HVals 返回所有 value 的列表
func (h Hash) HVals() []string {
	// 遍历 map，收集所有 value
	values := make([]string, 0, len(h))
	for _, value := range h {
		values = append(values, value)
	}
	return values
}

// HGetAll 返回所有 field 和 value 的列表 [f1, v1, f2, v2, ...]
func (h Hash) HGetAll() []string {
	fields := make([]string, 0, len(h)*2)
	// 遍历 map，交替追加 field 和 value
	for field, value := range h {
		fields = append(fields, field, value)
	}

	return fields
}
