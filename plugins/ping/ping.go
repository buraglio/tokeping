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
	pinger   *ping.Pinger
}

func init() {
	plugin.RegisterProbe("ping", New)
}

func New(cfg plugin.ProbeConfig) (plugin.Probe, error) {
	pr, err := ping.NewPinger(cfg.Target)
	if err != nil {
		return nil, err
	}
	pr.Count = 1
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	pr.Timeout = timeout
	return &PingProbe{
		name:     cfg.Name,
		target:   cfg.Target,
		interval: cfg.Interval,
		pinger:   pr,
	}, nil
}

func (p *PingProbe) Name() string            { return p.name }
func (p *PingProbe) Interval() time.Duration { return p.interval }
func (p *PingProbe) Run(ctx context.Context, out chan<- plugin.Metric) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	defer p.pinger.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := p.pinger.Run()
			if err != nil {
				continue
			}
			stats := p.pinger.Statistics()
			out <- plugin.Metric{
				Probe:   p.name,
				Time:    time.Now().Unix(),
				Latency: stats.AvgRtt.Seconds() * 1000,
			}
		}
	}
}
