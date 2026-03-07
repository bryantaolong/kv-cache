package types

import (
	"time"
)

// DataType 类型枚举
type DataType int

const (
	TypeString DataType = iota // 字符串
	TypeList                   // 列表
	TypeSet                    // 集合
	TypeZSet                   // 有序集合
	TypeHash                   // 哈希表
)

// Value 是存储的值
type Value struct {
	Type     DataType
	Data     interface{}
	ExpireAt *time.Time
}
