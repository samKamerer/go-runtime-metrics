package expvar

import (
	"expvar"
	"os"

	"github.com/sam-kamerer/go-runtime-metrics/v2/pkg/influxdb"
)

const defaultMeasurement = "go_runtime_metrics"

func init() {
	expvar.Publish(os.Args[0], influxdb.Metrics(defaultMeasurement))
}
