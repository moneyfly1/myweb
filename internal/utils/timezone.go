package utils

import (
	"time"
)

// 北京时区
var BeijingTZ = time.FixedZone("CST", 8*3600)

// GetBeijingTime 获取北京时间
func GetBeijingTime() time.Time {
	return time.Now().In(BeijingTZ)
}

// ToBeijingTime 转换为北京时间
func ToBeijingTime(t time.Time) time.Time {
	return t.In(BeijingTZ)
}

