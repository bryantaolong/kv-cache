package types

import (
	"strconv"
)

// String 类型直接使用 string，这里提供辅助方法

// StringIncr 将字符串解析为整数并 +1
// 返回值：新值的字符串形式和错误
func StringIncr(s string) (string, error) {
	num, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return "", err
	}
	num++
	return strconv.FormatInt(num, 10), nil
}

// StringIncrBy 将字符串解析为整数并增加 delta
func StringIncrBy(s string, delta int64) (string, error) {
	num, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return "", err
	}
	num += delta
	return strconv.FormatInt(num, 10), nil
}

// StringDecr 将字符串解析为整数并 -1
func StringDecr(s string) (string, error) {
	return StringIncrBy(s, -1)
}

// StringDecrBy 将字符串解析为整数并减少 delta
func StringDecrBy(s string, delta int64) (string, error) {
	num, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return "", err
	}
	num -= delta
	return strconv.FormatInt(num, 10), nil
}
