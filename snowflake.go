package fluxer

import (
	"strconv"
	"time"
)

// SnowflakeEpoch is the Unix time in milliseconds that snowflake timestamps
// count from, the start of 2015 in UTC.
const SnowflakeEpoch = 1420070400000

// ParseSnowflake parses a snowflake ID from its decimal string form.
func ParseSnowflake(id string) (uint64, error) {
	return strconv.ParseUint(id, 10, 64)
}

// SnowflakeTime returns the creation time encoded in a snowflake ID, in UTC.
func SnowflakeTime(id string) (time.Time, error) {
	v, err := ParseSnowflake(id)
	if err != nil {
		return time.Time{}, err
	}
	return time.UnixMilli(int64(v>>22) + SnowflakeEpoch).UTC(), nil
}

// SnowflakeAt returns the smallest snowflake for time t, which works as a
// Before or After bound when paging by time. Times before SnowflakeEpoch give
// "0".
func SnowflakeAt(t time.Time) string {
	ms := t.UnixMilli() - SnowflakeEpoch
	if ms < 0 {
		return "0"
	}
	return strconv.FormatUint(uint64(ms)<<22, 10)
}

// CompareSnowflakes compares two snowflake IDs numerically and returns -1, 0
// or +1. An ID that does not parse counts as 0.
func CompareSnowflakes(a, b string) int {
	x, _ := ParseSnowflake(a)
	y, _ := ParseSnowflake(b)
	switch {
	case x < y:
		return -1
	case x > y:
		return 1
	}
	return 0
}

// ShardFor returns the shard, out of count shards, that receives the events of
// guildID. It returns 0 when count is 1 or less.
func ShardFor(guildID string, count int) int {
	if count <= 1 {
		return 0
	}
	v, _ := ParseSnowflake(guildID)
	return int((v >> 22) % uint64(count))
}
