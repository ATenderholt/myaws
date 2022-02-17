package types

import "time"

const timeFormat = "2006-01-02T15:04:05.999-0700"

func timeMillisToString(ms int64) string {
	return time.UnixMilli(ms).Format(timeFormat)
}
