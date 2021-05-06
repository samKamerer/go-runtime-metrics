# go-runtime-metrics

Collect golang runtime metrics, pushing to [InfluxDB V2](https://www.influxdata.com/products/influxdb/) or pulling
with [Telegraf](https://www.influxdata.com/time-series-platform/telegraf/). Inspired
by https://github.com/bmhatfield/go-runtime-metrics

## Installation

    go get -u github.com/sam-kamerer/go-runtime-metrics/v2

## Push Usage

This library can be configured to push metrics directly to InfluxDB.

```go
package main

import (
	"github.com/sam-kamerer/go-runtime-metrics/v2/pkg/metrics"
)

func main() {
	err := metrics.RunCollector(&metrics.Config{})

	if err != nil {
		// handle error
	}
}

```

Once imported and running, you can expect a number of Go runtime metrics to be sent to InfluxDB. An example of what this
looks like when configured to work with [Grafana](http://grafana.org/):

![](/grafana.png)

[Download Dashboard](https://grafana.net/dashboards/1144)

## Pull Usage via [expvar](https://golang.org/pkg/expvar/)

Package [expvar](https://golang.org/pkg/expvar/) provides a standardized interface to public variables. This library
provides an exported InfluxDB formatted variable with a few other benefits:

* Metric names are easily parsed by regexp.
* Lighter than the standard library memstat expvar
* Includes stats for `cpu.cgo_calls`, `cpu.goroutines` and timing of the last GC pause with `mem.gc.pause`.
* Works out the box with
  Telegraf's [InfluxDB input plugin](https://github.com/influxdata/telegraf/tree/master/plugins/inputs/influxdb)

Import this library's expvar package with `import _ "github.com/sam-kamerer/go-runtime-metrics/v2/expvar"` to export a
variable with default configurations.

```json
{
  "/go/bin/binary": {
    "name": "go_runtime_metrics",
    "tags": {
      "go.arch": "amd64",
      "go.os": "darwin",
      "go.version": "go1.7.4"
    },
    "values": {
      "cpu.count": 4,
      "cpu.cgo_calls": 1,
      "cpu.goroutines": 2,
      "mem.alloc": 667576,
      "mem.frees": 104,
      "mem.gc.count": 0,
      "mem.gc.last": 0,
      "mem.gc.next": 4194304,
      "mem.gc.pause": 0,
      "mem.gc.pause_total": 0,
      "mem.gc.sys": 65536,
      "mem.heap.alloc": 667576,
      "mem.heap.idle": 475136,
      "mem.heap.inuse": 1327104,
      "mem.heap.objects": 5227,
      "mem.heap.released": 0,
      "mem.heap.sys": 1802240,
      "mem.lookups": 3,
      "mem.malloc": 5331,
      "mem.othersys": 820558,
      "mem.stack.inuse": 294912,
      "mem.stack.mcache_inuse": 4800,
      "mem.stack.mcache_sys": 16384,
      "mem.stack.mspan_inuse": 14160,
      "mem.stack.mspan_sys": 16384,
      "mem.stack.sys": 294912,
      "mem.sys": 3018752,
      "mem.total": 667576
    }
  }
}
```

#### Configuring with [Telegraf](https://www.influxdata.com/time-series-platform/telegraf/)

Your program must import `_ "github.com/sam-kamerer/go-runtime-metrics/v2/expvar` in order for an InfluxDB formatted
variable to be exported via `/debug/vars`.

1. [Install Telegraf](https://github.com/influxdata/telegraf#installation)

2. Make a config file utilizing the influxdb input plugin and an output plugin of your choice.

    ```toml
    [[inputs.influxdb]]
      urls = ["http://localhost:6060/debug/vars"]
    
    [[outputs.influxdb]]
      urls = ["http://localhost:8086"]
      ## The target database for metrics (telegraf will create it if not exists).
      database = "stats" # required
      
    ## [[outputs.file]]
    ##   files = ["stdout"]
    ##   data_format = "json"
    ```

3. Start the Telegraf agent with `telegraf -config config.conf`

#### Benchmarks

Benchmark against standard library memstat expvar:


```
goos: darwin
goarch: amd64
pkg: github.com/sam-kamerer/go-runtime-metrics/v2/pkg/influxdb
cpu: Intel(R) Core(TM) i7-4870HQ CPU @ 2.50GHz

BenchmarkMetrics     	   44070	     27732 ns/op	    3024 B/op	      19 allocs/op
BenchmarkMetrics-2   	   27457	     42724 ns/op	    3025 B/op	      19 allocs/op
BenchmarkMetrics-4   	   30907	     38912 ns/op	    3028 B/op	      19 allocs/op
BenchmarkMetrics-8   	   29810	     40576 ns/op	    3033 B/op	      19 allocs/op
BenchmarkMemStat     	   17384	     74193 ns/op	   25531 B/op	       3 allocs/op
BenchmarkMemStat-2   	   18729	     64245 ns/op	   25658 B/op	       3 allocs/op
BenchmarkMemStat-4   	   15324	     71106 ns/op	   25779 B/op	       3 allocs/op
BenchmarkMemStat-8   	   14280	     76296 ns/op	   26104 B/op	       3 allocs/op
```
