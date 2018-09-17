package collector

import (
	"runtime"
	"time"
)

type (
	// CollectStatsCallback represents a callback after successfully gathering statistics
	CollectStatsCallback func(Fields)

	// Collector implements the periodic grabbing of informational data from the
	// runtime package and outputting the values to a GaugeFunc.
	Collector struct {
		// PauseDur represents the interval in-between each set of stats output.
		// Defaults to 10 seconds.
		PauseDur time.Duration

		// EnableCPU determines whether CPU statistics will be output. Defaults to true.
		EnableCPU bool

		// EnableMem determines whether memory statistics will be output. Defaults to true.
		EnableMem bool

		// Done, when closed, is used to signal Collector that is should stop collecting
		// statistics and the Run function should return.
		Done <-chan struct{}

		collectStatsCallback CollectStatsCallback
	}

	Fields struct {
		// CPU
		NumCpu       int   `json:"cpu.count"`
		NumGoroutine int   `json:"cpu.goroutines"`
		NumCgoCall   int64 `json:"cpu.cgo_calls"`

		// General
		Alloc      int64 `json:"mem.alloc"`
		TotalAlloc int64 `json:"mem.total"`
		Sys        int64 `json:"mem.sys"`
		Lookups    int64 `json:"mem.lookups"`
		Mallocs    int64 `json:"mem.malloc"`
		Frees      int64 `json:"mem.frees"`

		// Heap
		HeapAlloc    int64 `json:"mem.heap.alloc"`
		HeapSys      int64 `json:"mem.heap.sys"`
		HeapIdle     int64 `json:"mem.heap.idle"`
		HeapInuse    int64 `json:"mem.heap.inuse"`
		HeapReleased int64 `json:"mem.heap.released"`
		HeapObjects  int64 `json:"mem.heap.objects"`

		// Stack
		StackInuse  int64 `json:"mem.stack.inuse"`
		StackSys    int64 `json:"mem.stack.sys"`
		MSpanInuse  int64 `json:"mem.stack.mspan_inuse"`
		MSpanSys    int64 `json:"mem.stack.mspan_sys"`
		MCacheInuse int64 `json:"mem.stack.mcache_inuse"`
		MCacheSys   int64 `json:"mem.stack.mcache_sys"`
		OtherSys    int64 `json:"mem.othersys"`

		// GC
		GCSys         int64   `json:"mem.gc.sys"`
		NextGC        int64   `json:"mem.gc.next"`
		LastGC        int64   `json:"mem.gc.last"`
		PauseTotalNs  int64   `json:"mem.gc.pause_total"`
		PauseNs       int64   `json:"mem.gc.pause"`
		NumGC         int32   `json:"mem.gc.count"`
		GCCPUFraction float64 `json:"mem.gc.cpu_fraction"`

		Goarch  string `json:"-"`
		Goos    string `json:"-"`
		Version string `json:"-"`
	}
)

// New creates a new Collector that will periodically output statistics to collectStatsCallback. It
// will also set the values of the exported fields to the described defaults. The values
// of the exported defaults can be changed at any point before Run is called.
func New(callback CollectStatsCallback) *Collector {
	if callback == nil {
		callback = func(Fields) {}
	}

	return &Collector{
		PauseDur:             10 * time.Second,
		EnableCPU:            true,
		EnableMem:            true,
		collectStatsCallback: callback,
	}
}

// Run gathers statistics then outputs them to the configured PointFunc every
// PauseDur. Unlike OneOff, this function will return until Done has been closed
// (or never if Done is nil), therefore it should be called in its own go routine.
func (c *Collector) Run() {
	//c.collectStatsCallback(c.CollectStats())
	tickCh := time.NewTicker(c.PauseDur).C
	for {
		select {
		case <-c.Done:
			return
		case <-tickCh:
			c.collectStatsCallback(c.CollectStats())
		}
	}
}

func (c *Collector) CollectStats() (fields Fields) {
	if c.EnableMem {
		collectMemStats(&fields)
	}

	if c.EnableCPU {
		collectCPUStats(&fields)
	}

	fields.Goos = runtime.GOOS
	fields.Goarch = runtime.GOARCH
	fields.Version = runtime.Version()

	return fields
}

func collectCPUStats(f *Fields) {
	f.NumCpu = runtime.NumCPU()
	f.NumGoroutine = runtime.NumGoroutine()
	f.NumCgoCall = runtime.NumCgoCall()
}

func collectMemStats(f *Fields) {
	m := &runtime.MemStats{}
	runtime.ReadMemStats(m)
	f.Alloc = int64(m.Alloc)
	f.TotalAlloc = int64(m.TotalAlloc)
	f.Sys = int64(m.Sys)
	f.Lookups = int64(m.Lookups)
	f.Mallocs = int64(m.Mallocs)
	f.Frees = int64(m.Frees)
	f.HeapAlloc = int64(m.HeapAlloc)
	f.HeapSys = int64(m.HeapSys)
	f.HeapIdle = int64(m.HeapIdle)
	f.HeapInuse = int64(m.HeapInuse)
	f.HeapReleased = int64(m.HeapReleased)
	f.HeapObjects = int64(m.HeapObjects)
	f.StackInuse = int64(m.StackInuse)
	f.StackSys = int64(m.StackSys)
	f.MSpanInuse = int64(m.MSpanInuse)
	f.MSpanSys = int64(m.MSpanSys)
	f.MCacheInuse = int64(m.MCacheInuse)
	f.MCacheSys = int64(m.MCacheSys)
	f.OtherSys = int64(m.OtherSys)
	f.GCSys = int64(m.GCSys)
	f.NextGC = int64(m.NextGC)
	f.LastGC = int64(m.LastGC)
	f.PauseTotalNs = int64(m.PauseTotalNs)
	f.PauseNs = int64(m.PauseNs[(m.NumGC+255)%256])
	f.NumGC = int32(m.NumGC)
	f.GCCPUFraction = m.GCCPUFraction
}

func (f *Fields) Tags() map[string]string {
	return map[string]string{
		"go.os":      f.Goos,
		"go.arch":    f.Goarch,
		"go.version": f.Version,
	}
}

func (f *Fields) Values() map[string]interface{} {
	return map[string]interface{}{
		"cpu.count":      f.NumCpu,
		"cpu.goroutines": f.NumGoroutine,
		"cpu.cgo_calls":  f.NumCgoCall,

		"mem.alloc":   f.Alloc,
		"mem.total":   f.TotalAlloc,
		"mem.sys":     f.Sys,
		"mem.lookups": f.Lookups,
		"mem.malloc":  f.Mallocs,
		"mem.frees":   f.Frees,

		"mem.heap.alloc":    f.HeapAlloc,
		"mem.heap.sys":      f.HeapSys,
		"mem.heap.idle":     f.HeapIdle,
		"mem.heap.inuse":    f.HeapInuse,
		"mem.heap.released": f.HeapReleased,
		"mem.heap.objects":  f.HeapObjects,

		"mem.stack.inuse":        f.StackInuse,
		"mem.stack.sys":          f.StackSys,
		"mem.stack.mspan_inuse":  f.MSpanInuse,
		"mem.stack.mspan_sys":    f.MSpanSys,
		"mem.stack.mcache_inuse": f.MCacheInuse,
		"mem.stack.mcache_sys":   f.MCacheSys,
		"mem.othersys":           f.OtherSys,

		"mem.gc.pause":        f.PauseNs,
		"mem.gc.pause_total":  f.PauseTotalNs,
		"mem.gc.sys":          f.GCSys,
		"mem.gc.next":         f.NextGC,
		"mem.gc.last":         f.LastGC,
		"mem.gc.count":        f.NumGC,
		"mem.gc.cpu_fraction": f.GCCPUFraction,
	}
}
