package influxdb

import (
    "context"
    "time"

    influxdb2 "github.com/influxdata/influxdb-client-go/v2"
    api "github.com/influxdata/influxdb-client-go/v2/api"
    "tokeping/pkg/plugin"
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
func (o *InfluxOutput) Send(m plugin.Metric) {
    p := influxdb2.NewPointWithMeasurement("latency").
        AddTag("probe", m.Probe).
        AddField("value", m.Latency).
        SetTime(time.Unix(m.Time, 0))
    o.writeAPI.WritePoint(context.Background(), p)
}
func (o *InfluxOutput) Stop() error {
    o.client.Close()
    return nil
}
