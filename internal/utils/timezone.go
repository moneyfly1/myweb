package utils

import (
	"time"
)

var BeijingTZ = time.FixedZone("CST", 8*3600)

func GetBeijingTime() time.Time {
	return time.Now().In(BeijingTZ)
}

func ToBeijingTime(t time.Time) time.Time {
	return t.In(BeijingTZ)
}
