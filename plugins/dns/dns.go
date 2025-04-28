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
    resolver *net.Resolver
}

func init() {
    plugin.RegisterProbe("dns", New)
}

func New(cfg plugin.ProbeConfig) (plugin.Probe, error) {
    // default to system resolver
    r := net.DefaultResolver

    // if cfg.Resolver is set, dial that address over UDP
    if cfg.Resolver != "" {
        r = &net.Resolver{
            PreferGo: true,
            Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
                d := net.Dialer{Timeout: 2 * time.Second}
                return d.DialContext(ctx, "udp", cfg.Resolver)
            },
        }
    }

    return &DNSProbe{
        name:     cfg.Name,
        target:   cfg.Target,
        interval: cfg.Interval,
        resolver: r,
    }, nil
}

func (p *DNSProbe) Name() string           { return p.name }
func (p *DNSProbe) Interval() time.Duration { return p.interval }

func (p *DNSProbe) Run(ctx context.Context, out chan<- plugin.Metric) {
    ticker := time.NewTicker(p.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            start := time.Now()
            _, err := p.resolver.LookupHost(ctx, p.target)
            elapsed := time.Since(start).Seconds() * 1000 // ms

            if err != nil {
                // on failure, record a negative latency
                elapsed = -1
            }

            out <- plugin.Metric{
                Probe:   p.name,
                Time:    time.Now().Unix(),
                Latency: elapsed,
            }
        }
    }
}
