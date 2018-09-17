package runstats

import (
	"log"
	"os"
	"time"
	"fmt"

	influxDBClient "github.com/influxdata/influxdb/client/v2"
	"github.com/pkg/errors"
	"github.com/sam-kamerer/go-runtime-metrics/collector"
)

const (
	defaultHost               = "http://localhost:8086"
	defaultMeasurement        = "go.runtime"
	defaultDatabase           = "stats"
	defaultCollectionInterval = 10 * time.Second
	defaultBatchInterval      = 60 * time.Second
	defaultPingInterval       = 60 * time.Second
)

type (
	Logger interface {
		Println(v ...interface{})
		Printf(f string, v ...interface{})
	}
	DefaultLogger struct{}

	Config struct {
		// InfluxDb scheme://host:port
		// Default is "http://localhost:8086".
		Addr string

		// Database to write points to.
		// Default is "statsCollector" and is auto created
		Database string

		// Username with privileges on provided database.
		Username string

		// Password for provided user.
		Password string

		// Measurement to write points to.
		// Default is "go.runtime.<hostname>".
		Measurement string

		// Measurement to write points to.
		RetentionPolicy string

		// Interval at which to write batched points to InfluxDB.
		// Default is 60 seconds
		BatchInterval time.Duration

		// Interval at which to check status of cluster
		// Default is 60 seconds
		PingInterval time.Duration

		// Precision in time to write your points in.
		// Default is nanoseconds
		Precision string

		// Interval at which to collect points.
		// Default is 10 seconds
		CollectionInterval time.Duration

		// Disable collecting CPU Statistics. cpu.*
		// Default is false
		DisableCpu bool

		// Disable collecting Memory Statistics. mem.*
		DisableMem bool

		// Default is DefaultLogger which exits when the library encounters a fatal error.
		Logger Logger
	}

	statsSender struct {
		config *Config
		logger Logger
		client influxDBClient.Client
		points influxDBClient.BatchPoints
		pc     chan *influxDBClient.Point
	}
)

func (config *Config) init() {
	if config == nil {
		config = &Config{}
	}

	if config.Database == "" {
		config.Database = defaultDatabase
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

	if config.BatchInterval == 0 {
		config.BatchInterval = defaultBatchInterval
	}

	if config.PingInterval == 0 {
		config.PingInterval = defaultPingInterval
	}

	if config.Logger == nil {
		config.Logger = &DefaultLogger{}
	}
}

// Make client
func newClient(config influxDBClient.HTTPConfig) (client influxDBClient.Client, err error) {
	client, err = influxDBClient.NewHTTPClient(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create influxDB client")
	}

	// Ping InfluxDB to ensure there is a connection
	if _, _, err := client.Ping(5 * time.Second); err != nil {
		err = errors.Wrap(err, "failed to ping influxDB client")
	}
	return
}

func newStatsSender(config *Config) (*statsSender, error) {
	client, err := newClient(influxDBClient.HTTPConfig{
		Addr:     config.Addr,
		Username: config.Username,
		Password: config.Password,
	})
	if err != nil {
		return nil, err
	}

	// Auto create database
	_, err = queryDB(client, fmt.Sprintf("CREATE DATABASE \"%s\"", config.Database))
	if err != nil {
		config.Logger.Println(err)
	}

	sender := &statsSender{
		logger: config.Logger,
		client: client,
		config: config,
		pc:     make(chan *influxDBClient.Point),
	}

	bp, err := sender.newBatch()
	if err != nil {
		client.Close()
		return nil, err
	}
	sender.points = bp

	go func() {
		var lastPingError error
		for range time.NewTicker(sender.config.PingInterval).C {
			if _, _, err := sender.client.Ping(3 * time.Second); err != nil {
				log.Println("ping error:", err)
				lastPingError = err
			} else if lastPingError != nil {
				log.Println("ping ok")
				client, err := newClient(influxDBClient.HTTPConfig{
					Addr:     sender.config.Addr,
					Username: sender.config.Username,
					Password: sender.config.Password,
				})
				if err == nil {
					sender.client.Close()
					sender.client = client
					lastPingError = nil
					log.Println("client reconnected")
				}
			}
		}
	}()

	return sender, nil
}

func RunCollector(config *Config) error {
	config.init()

	sender, err := newStatsSender(config)
	if err != nil {
		return err
	}
	go sender.loop(config.BatchInterval)

	c := collector.New(sender.onNewPoint)
	c.PauseDur = config.CollectionInterval
	c.EnableCPU = !config.DisableCpu
	c.EnableMem = !config.DisableMem

	go c.Run()

	return nil
}

func (r *statsSender) onNewPoint(fields collector.Fields) {
	pt, err := influxDBClient.NewPoint(r.config.Measurement, fields.Tags(), fields.Values(), time.Now())
	if err != nil {
		r.logger.Println(errors.Wrap(err, "error while creating point"))
		return
	}
	r.pc <- pt
}

func (r *statsSender) newBatch() (bp influxDBClient.BatchPoints, err error) {
	bp, err = influxDBClient.NewBatchPoints(influxDBClient.BatchPointsConfig{
		Database:        r.config.Database,
		Precision:       r.config.Precision,
		RetentionPolicy: r.config.RetentionPolicy,
	})
	if err != nil {
		r.logger.Println(errors.Wrap(err, "could not create BatchPoints"))
	}
	return
}

// Write collected points to InfluxDB periodically
func (r *statsSender) loop(interval time.Duration) {
	var err error
	tickCh := time.NewTicker(interval).C
	for {
		select {
		case <-tickCh:
			points := r.points.Points()
			if r.points == nil || len(points) == 0 {
				continue
			}

			if err = r.client.Write(r.points); err != nil {
				r.logger.Println(errors.Wrap(err, "could not write points to InfluxDB"))
				continue
			}

			if r.points, err = r.newBatch(); err != nil {
				continue
			}

		case pt := <-r.pc:
			if r.points != nil {
				r.points.AddPoint(pt)
			}
		}
	}
}

func (*DefaultLogger) Println(v ...interface{}) {
	log.Println(v...)
}
func (*DefaultLogger) Printf(f string, v ...interface{}) {
	log.Printf(f, v...)
}

func queryDB(c influxDBClient.Client, cmd string) (res []influxDBClient.Result, err error) {
	if response, err := c.Query(influxDBClient.Query{Command: cmd}); err == nil {
		if response.Error() != nil {
			return res, response.Error()
		}
		res = response.Results
	}
	return
}
