package fluxer

import (
	"strconv"
	"time"
)

const SnowflakeEpoch = 1420070400000

func ParseSnowflake(id string) (uint64, error) {
	return strconv.ParseUint(id, 10, 64)
}

func SnowflakeTime(id string) (time.Time, error) {
	v, err := ParseSnowflake(id)
	if err != nil {
		return time.Time{}, err
	}
	return time.UnixMilli(int64(v>>22) + SnowflakeEpoch).UTC(), nil
}

func SnowflakeAt(t time.Time) string {
	ms := t.UnixMilli() - SnowflakeEpoch
	if ms < 0 {
		return "0"
	}
	return strconv.FormatUint(uint64(ms)<<22, 10)
}

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

func ShardFor(guildID string, count int) int {
	if count <= 1 {
		return 0
	}
	v, _ := ParseSnowflake(guildID)
	return int((v >> 22) % uint64(count))
}
