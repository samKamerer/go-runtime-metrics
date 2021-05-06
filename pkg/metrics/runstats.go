package metrics

import (
	"crypto/tls"
	"os"
	"time"

	"github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/sam-kamerer/go-runtime-metrics/v2/pkg/collector"
)

const (
	defaultHost                    = "http://localhost:8086"
	defaultMeasurement             = "go.runtime"
	defaultBucket                  = "stats"
	defaultCollectionInterval      = 10 * time.Second
	defaultFlushInterval      uint = 60000 // in ms
)

type (
	Config struct {
		// InfluxDb scheme://host:port
		// Default is "http://localhost:8086".
		Addr string

		AuthToken string

		// Organization
		Org string

		// Bucket to write points to.
		// Default is "statsCollector" and is auto created
		Bucket string

		// Measurement to write points to.
		// Default is "go.runtime.<hostname>".
		Measurement string

		// Flush interval in ms
		FlushInterval uint

		// Interval at which to collect points.
		// Default is 10 seconds
		CollectionInterval time.Duration

		// Disable collecting CPU Statistics. cpu.*
		// Default is false
		DisableCpu bool

		// Disable collecting Memory Statistics. mem.*
		DisableMem bool
	}

	statsSender struct {
		config   *Config
		client   influxdb2.Client
		writeAPI api.WriteAPI
		pc       chan *write.Point
	}
)

func (config *Config) init() {
	if config == nil {
		*config = Config{}
	}

	if config.Bucket == "" {
		config.Bucket = defaultBucket
	}

	if config.Addr == "" {
		config.Addr = defaultHost
	}

	if config.Measurement == "" {
		config.Measurement = defaultMeasurement

		if hn, err := os.Hostname(); err != nil {
			config.Measurement += ".unknown"
		} else {
			config.Measurement += "." + hn
		}
	}

	if config.CollectionInterval == 0 {
		config.CollectionInterval = defaultCollectionInterval
	}

	if config.FlushInterval == 0 {
		config.FlushInterval = defaultFlushInterval
	}
}

func newStatsSender(config *Config) *statsSender {
	clientOptions := influxdb2.DefaultOptions().
		SetFlushInterval(config.FlushInterval).
		SetUseGZip(true).
		SetTLSConfig(&tls.Config{InsecureSkipVerify: true})

	sender := &statsSender{
		client: influxdb2.NewClientWithOptions(config.Addr, config.AuthToken, clientOptions),
		config: config,
		pc:     make(chan *write.Point),
	}
	sender.writeAPI = sender.client.WriteAPI(config.Org, config.Bucket)

	return sender
}

func RunCollector(config *Config) error {
	config.init()

	c := collector.New(newStatsSender(config).onNewPoint)
	c.PauseDur = config.CollectionInterval
	c.EnableCPU = !config.DisableCpu
	c.EnableMem = !config.DisableMem

	go c.Run()

	return nil
}

func (r *statsSender) onNewPoint(fields collector.Fields) {
	p := influxdb2.NewPointWithMeasurement(r.config.Measurement)
	for k, v := range fields.Tags() {
		p.AddTag(k, v)
	}
	for k, v := range fields.Values() {
		p.AddField(k, v)
	}
	p.SetTime(time.Now())
	r.writeAPI.WritePoint(p)
}
