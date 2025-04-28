package ping

import (
    "context"
    "time"

    "github.com/go-ping/ping"
    "tokeping/pkg/plugin"
)

type PingProbe struct {
    name     string
    target   string
    interval time.Duration
}

func init() {
    plugin.RegisterProbe("ping", New)
}

func New(cfg plugin.ProbeConfig) (plugin.Probe, error) {
    return &PingProbe{cfg.Name, cfg.Target, cfg.Interval}, nil
}

func (p *PingProbe) Name() string           { return p.name }
func (p *PingProbe) Interval() time.Duration { return p.interval }
func (p *PingProbe) Run(ctx context.Context, out chan<- plugin.Metric) {
    ticker := time.NewTicker(p.interval)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            pr, err := ping.NewPinger(p.target)
            if err != nil {
                continue
            }
            pr.Count = 1
            pr.Run()
            stats := pr.Statistics()
            out <- plugin.Metric{
                Probe:   p.name,
                Time:    time.Now().Unix(),
                Latency: stats.AvgRtt.Seconds() * 1000,
            }
        }
    }
}
