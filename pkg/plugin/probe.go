package plugin

import (
    "context"
    "time"
)

type Metric struct {
    Probe   string
    Time    int64
    Latency float64
}

type Probe interface {
    Name() string
    Interval() time.Duration
    Run(ctx context.Context, out chan<- Metric)
}
