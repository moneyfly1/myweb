package database

import (
	"database/sql"
	"time"
)

func NullString(s string) sql.NullString {
	return sql.NullString{
		String: s,
		Valid:  s != "",
	}
}

func NullInt64(i int64) sql.NullInt64 {
	return sql.NullInt64{
		Int64: i,
		Valid: true,
	}
}

func NullFloat64(f float64) sql.NullFloat64 {
	return sql.NullFloat64{
		Float64: f,
		Valid:   true,
	}
}

func NullTime(t time.Time) sql.NullTime {
	return sql.NullTime{
		Time:  t,
		Valid: !t.IsZero(),
	}
}
