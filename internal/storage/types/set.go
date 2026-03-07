package types

// Set 是一个无序不重复集合，用 map[string]struct{} 实现
type Set map[string]struct{}

// NewSet 创建一个新的 Set
func NewSet() Set {
	return make(Set)
}

// SAdd 添加一个或多个成员
// 返回值：成功添加的成员数量（已存在的不会重复添加）
func (s Set) SAdd(members ...string) int {
	count := 0
	for _, m := range members {
		if _, ok := s[m]; !ok {
			s[m] = struct{}{}
			count++
		}
	}
	return count
}

// SRem 移除一个或多个成员
// 返回值：成功移除的成员数量
func (s Set) SRem(members ...string) int {
	count := 0
	for _, m := range members {
		if _, ok := s[m]; ok {
			delete(s, m)
			count++
		}
	}
	return count
}

// SIsMember 判断成员是否存在
func (s Set) SIsMember(member string) bool {
	_, ok := s[member]
	return ok
}

// SMembers 返回所有成员
func (s Set) SMembers() []string {
	members := make([]string, 0, len(s))
	for m := range s {
		members = append(members, m)
	}
	return members
}

// SCard 返回集合的成员数量
func (s Set) SCard() int {
	return len(s)
}

// SPop 随机弹出指定数量的成员
// 返回值：被弹出的成员列表
func (s Set) SPop(count int) []string {
	if count <= 0 || len(s) == 0 {
		return []string{}
	}

	// 简单实现：遍历取前 count 个
	result := make([]string, 0, count)
	for m := range s {
		result = append(result, m)
		delete(s, m)
		if len(result) >= count {
			break
		}
	}
	return result
}

// SUnion 返回两个集合的并集（不修改原集合）
func (s Set) SUnion(other Set) Set {
	result := NewSet()
	for m := range s {
		result[m] = struct{}{}
	}
	for m := range other {
		result[m] = struct{}{}
	}
	return result
}

// SInter 返回两个集合的交集（不修改原集合）
func (s Set) SInter(other Set) Set {
	result := NewSet()
	for m := range s {
		if _, ok := other[m]; ok {
			result[m] = struct{}{}
		}
	}
	return result
}

// SDiff 返回两个集合的差集（不修改原集合）
func (s Set) SDiff(other Set) Set {
	result := NewSet()
	for m := range s {
		if _, ok := other[m]; !ok {
			result[m] = struct{}{}
		}
	}
	return result
}
