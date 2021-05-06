package influxdb

import (
	"encoding/json"
	"expvar"
	"runtime"
	"testing"
)

func TestMetrics(t *testing.T) {
	point := &Point{}
	name := "test"

	json.Unmarshal([]byte(Metrics(name).String()), point)

	expKeys := []string{
		"cpu.goroutines",
		"mem.lookups",
		"mem.gc.count",
	}

	for _, expKey := range expKeys {
		if _, ok := point.Values.Values()[expKey]; !ok {
			t.Errorf("expected key (%s) not found", expKey)
		}
	}

	if result := point.Name; result != "test" {
		t.Errorf("expected name (%s) got (%s)", name, result)
	}
}

func BenchmarkMetrics(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Metrics("some_metric").String()
		}
	})
}

func BenchmarkMemStat(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			expvar.Func(memStats).String()
		}
	})
}

func memStats() interface{} {
	stats := &runtime.MemStats{}
	runtime.ReadMemStats(stats)
	return *stats
}
