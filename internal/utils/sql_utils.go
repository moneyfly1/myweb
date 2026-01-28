package utils

import (
	"database/sql"
)

func GetNullStringValue(ns sql.NullString) interface{} {
	if ns.Valid {
		return ns.String
	}
	return nil
}

func GetNullInt64Value(ni sql.NullInt64) interface{} {
	if ni.Valid {
		return ni.Int64
	}
	return nil
}

func GetNullFloat64Value(nf sql.NullFloat64) interface{} {
	if nf.Valid {
		return nf.Float64
	}
	return nil
}

func GetNullTimeValue(nt sql.NullTime) interface{} {
	if nt.Valid {
		return nt.Time.Format("2006-01-02 15:04:05")
	}
	return nil
}

func GetStringValue(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}
