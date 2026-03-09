package utils

import (
	"errors"
	"math"
)

// SafeUintToInt64 安全地将 uint 转换为 int64，检查溢出
func SafeUintToInt64(u uint) (int64, error) {
	if u > math.MaxInt64 {
		return 0, errors.New("uint value overflow int64")
	}
	return int64(u), nil
}

// SafeUint64ToInt 安全地将 uint64 转换为 int，检查溢出
func SafeUint64ToInt(u uint64) (int, error) {
	if u > math.MaxInt {
		return 0, errors.New("uint64 value overflow int")
	}
	return int(u), nil
}

// SafeUintToInt 安全地将 uint 转换为 int，检查溢出
func SafeUintToInt(u uint) (int, error) {
	if u > math.MaxInt {
		return 0, errors.New("uint value overflow int")
	}
	return int(u), nil
}

// SafeInt64ToInt 安全地将 int64 转换为 int，检查溢出
func SafeInt64ToInt(i int64) (int, error) {
	if i > math.MaxInt || i < math.MinInt {
		return 0, errors.New("int64 value overflow int")
	}
	return int(i), nil
}

// MustSafeUintToInt64 安全地将 uint 转换为 int64，溢出时返回 0
func MustSafeUintToInt64(u uint) int64 {
	if u > math.MaxInt64 {
		return 0
	}
	return int64(u)
}

// MustSafeUint64ToInt 安全地将 uint64 转换为 int，溢出时返回 0
func MustSafeUint64ToInt(u uint64) int {
	if u > math.MaxInt {
		return 0
	}
	return int(u)
}

// MustSafeUintToInt 安全地将 uint 转换为 int，溢出时返回 0
func MustSafeUintToInt(u uint) int {
	if u > math.MaxInt {
		return 0
	}
	return int(u)
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
