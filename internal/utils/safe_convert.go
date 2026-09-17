package utils

import (
	"math"
)

// MustSafeUintToInt64 安全地将 uint 转换为 int64，溢出时返回 0
func MustSafeUintToInt64(u uint) int64 {
	if u > math.MaxInt64 {
		return 0
	}
	return int64(u)
}

// MustSafeIntToUint 安全地将 int 转换为 uint，负数时返回 0
func MustSafeIntToUint(i int) uint {
	if i < 0 {
		return 0
	}
	return uint(i)
}

// MustSafeInt64ToUint 安全地将 int64 转换为 uint，负数或溢出时返回 0
func MustSafeInt64ToUint(i int64) uint {
	if i < 0 || i > math.MaxInt {
		return 0
	}
	return uint(i)
}

// MustSafeInt64ToRune 安全地将 int64 转换为 rune，溢出时返回 0
func MustSafeInt64ToRune(i int64) rune {
	if i < 0 || i > 0x10FFFF { // Unicode 最大码点
		return 0
	}
	return rune(i)
}
