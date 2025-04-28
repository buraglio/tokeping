package dns

import (
    "context"
    "net"
    "time"

    "tokeping/pkg/plugin"
)

type DNSProbe struct {
    name     string
    target   string
    interval time.Duration
}

func init() {
    plugin.RegisterProbe("dns", New)
}

func New(cfg plugin.ProbeConfig) (plugin.Probe, error) {
    return &DNSProbe{
        name:     cfg.Name,
        target:   cfg.Target,   // e.g. "example.com"
        interval: cfg.Interval, // e.g. 5s
    }, nil
}

func (p *DNSProbe) Name() string {
    return p.name
}

func (p *DNSProbe) Interval() time.Duration {
    return p.interval
}

func (p *DNSProbe) Run(ctx context.Context, out chan<- plugin.Metric) {
    ticker := time.NewTicker(p.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            start := time.Now()
            // use default resolver; for custom, you could use a dns.Client
            _, err := net.DefaultResolver.LookupHost(context.Background(), p.target)
            duration := time.Since(start).Seconds() * 1000 // ms

            // if you want to signal failures, you could set Latency = -1
            if err != nil {
                duration = -1
            }

            out <- plugin.Metric{
                Probe:   p.name,
                Time:    time.Now().Unix(),
                Latency: duration,
            }
        }
    }
}
