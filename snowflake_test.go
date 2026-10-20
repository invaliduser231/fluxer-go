package fluxer_test

import (
	"testing"
	"time"

	"github.com/invaliduser231/fluxer-go"
)

func TestSnowflakeTime(t *testing.T) {
	at := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	id := fluxer.SnowflakeAt(at)
	got, err := fluxer.SnowflakeTime(id)
	if err != nil || !got.Equal(at) {
		t.Errorf("got %v, %v from %s", got, err, id)
	}
	if got, _ := fluxer.SnowflakeTime("1189375284394692608"); got.Year() != 2023 {
		t.Errorf("got %v", got)
	}
	if _, err := fluxer.SnowflakeTime("abc"); err == nil {
		t.Error("expected error")
	}
	if fluxer.SnowflakeAt(time.Unix(0, 0)) != "0" {
		t.Error("time before the epoch")
	}
}

func TestCompareSnowflakes(t *testing.T) {
	if fluxer.CompareSnowflakes("9", "10") != -1 || fluxer.CompareSnowflakes("10", "9") != 1 || fluxer.CompareSnowflakes("7", "7") != 0 {
		t.Error("compared as text")
	}
}

func TestShardFor(t *testing.T) {
	id := "1189375284394692608"
	if got := fluxer.ShardFor(id, 1); got != 0 {
		t.Errorf("got %d", got)
	}
	v, _ := fluxer.ParseSnowflake(id)
	if got, want := fluxer.ShardFor(id, 16), int((v>>22)%16); got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}
