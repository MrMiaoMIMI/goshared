package datetime

import "time"

func GetCurrentMillisecond() int64 {
	return time.Now().UnixNano() / 1000000
}

func NextNDayTime(n int) time.Time {
	return time.Now().AddDate(0, 0, n)
}

func NextNDayTimeMillisecond(n int) int64 {
	return NextNDayTime(n).UnixMilli()
}
