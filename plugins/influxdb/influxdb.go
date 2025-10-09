package influxdb

import (
	"context"
	"fmt"
	"os"
	"time"

	"tokeping/pkg/plugin"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	api "github.com/influxdata/influxdb-client-go/v2/api"
)

type InfluxOutput struct {
	client   influxdb2.Client
	writeAPI api.WriteAPIBlocking
}

func init() {
	plugin.RegisterOutput("influxdb", New)
}

func New(cfg plugin.OutputConfig) (plugin.Output, error) {
	client := influxdb2.NewClient(cfg.URL, cfg.Token)
	writeAPI := client.WriteAPIBlocking(cfg.Org, cfg.Bucket)
	return &InfluxOutput{client: client, writeAPI: writeAPI}, nil
}

func (o *InfluxOutput) Name() string { return "influxdb" }
func (o *InfluxOutput) Start() error { return nil }
func (o *InfluxOutput) Send(ctx context.Context, m plugin.Metric) {
	point := influxdb2.NewPointWithMeasurement("latency").
		AddTag("probe", m.Probe).
		AddField("value", m.Latency).
		SetTime(time.Unix(m.Time, 0))

	if err := o.writeAPI.WritePoint(ctx, point); err != nil {
		fmt.Fprintf(os.Stderr, "influx write error: %v\n", err)
	}
}

func (o *InfluxOutput) Stop() error {
	o.client.Close()
	return nil
}
