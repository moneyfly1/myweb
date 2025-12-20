package utils

import (
	"database/sql"
)

// GetNullStringValue 处理NullString
func GetNullStringValue(ns sql.NullString) interface{} {
	if ns.Valid {
		return ns.String
	}
	return nil
}

// GetNullInt64Value 处理NullInt64
func GetNullInt64Value(ni sql.NullInt64) interface{} {
	if ni.Valid {
		return ni.Int64
	}
	return nil
}

// GetNullFloat64Value 处理NullFloat64
func GetNullFloat64Value(nf sql.NullFloat64) interface{} {
	if nf.Valid {
		return nf.Float64
	}
	return nil
}

// GetNullTimeValue 处理NullTime
func GetNullTimeValue(nt sql.NullTime) interface{} {
	if nt.Valid {
		return nt.Time.Format("2006-01-02 15:04:05")
	}
	return nil
}

// GetStringValue safely gets string value from pointer
func GetStringValue(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}
