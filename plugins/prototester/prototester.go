package prototester

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"tokeping/pkg/plugin"
)

type ProtoTesterProbe struct {
	name       string
	target     string
	interval   time.Duration
	protoPath  string
	timeout    time.Duration
	emitBoth   bool
	emitScore  bool
	errorCount int
	maxErrors  int
}

type ProtoTesterResult struct {
	Mode       string `json:"mode"`
	Protocol   string `json:"protocol"`
	Comparison struct {
		DNSV4Stats struct {
			Sent        int     `json:"sent"`
			Received    int     `json:"received"`
			Lost        int     `json:"lost"`
			MinMS       float64 `json:"min_ms"`
			MaxMS       float64 `json:"max_ms"`
			AvgMS       float64 `json:"avg_ms"`
			StddevMS    float64 `json:"stddev_ms"`
			JitterMS    float64 `json:"jitter_ms"`
			SuccessRate float64 `json:"success_rate"`
		} `json:"dns_v4_stats"`
		DNSV6Stats struct {
			Sent        int     `json:"sent"`
			Received    int     `json:"received"`
			Lost        int     `json:"lost"`
			MinMS       float64 `json:"min_ms"`
			MaxMS       float64 `json:"max_ms"`
			AvgMS       float64 `json:"avg_ms"`
			StddevMS    float64 `json:"stddev_ms"`
			JitterMS    float64 `json:"jitter_ms"`
			SuccessRate float64 `json:"success_rate"`
		} `json:"dns_v6_stats"`
		IPv4Score float64 `json:"ipv4_score"`
		IPv6Score float64 `json:"ipv6_score"`
		Winner    string  `json:"winner"`
		Protocol  string  `json:"protocol"`
		Timestamp string  `json:"timestamp"`
	} `json:"comparison"`
	Timestamp string `json:"timestamp"`
}

func init() {
	plugin.RegisterProbe("prototester", New)
}

func New(cfg plugin.ProbeConfig) (plugin.Probe, error) {
	protoExe, err := exec.LookPath("prototester")
	if err != nil {
		return nil, fmt.Errorf("prototester not found in PATH: %v", err)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &ProtoTesterProbe{
		name:       cfg.Name,
		target:     cfg.Target,
		interval:   cfg.Interval,
		protoPath:  protoExe,
		timeout:    timeout,
		emitBoth:   true,
		emitScore:  true,
		errorCount: 0,
		maxErrors:  5,
	}, nil
}

func (p *ProtoTesterProbe) Name() string {
	return p.name
}

func (p *ProtoTesterProbe) Interval() time.Duration {
	return p.interval
}

func (p *ProtoTesterProbe) Run(ctx context.Context, out chan<- plugin.Metric) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			testCtx, cancel := context.WithTimeout(ctx, p.timeout)
			cmd := exec.CommandContext(testCtx, p.protoPath, "-compare", p.target, "-json")
			output, err := cmd.Output()
			cancel()

			if err != nil {
				p.errorCount++
				fmt.Fprintf(os.Stderr, "prototester error for %q (%d/%d): %v\n", p.name, p.errorCount, p.maxErrors, err)
				if p.errorCount >= p.maxErrors {
					fmt.Fprintf(os.Stderr, "prototester probe %q: max errors reached, skipping metric emission\n", p.name)
				}
				continue
			}
			p.errorCount = 0

			var result ProtoTesterResult
			if err := json.Unmarshal(output, &result); err != nil {
				fmt.Fprintf(os.Stderr, "prototester JSON parse error for %q: %v\n", p.name, err)
				continue
			}

			now := time.Now().Unix()

			if p.emitBoth {
				out <- plugin.Metric{
					Probe:   p.name + "_ipv4",
					Time:    now,
					Latency: result.Comparison.DNSV4Stats.AvgMS,
				}

				out <- plugin.Metric{
					Probe:   p.name + "_ipv6",
					Time:    now,
					Latency: result.Comparison.DNSV6Stats.AvgMS,
				}
			}

			if p.emitScore {
				out <- plugin.Metric{
					Probe:   p.name + "_ipv4_score",
					Time:    now,
					Latency: result.Comparison.IPv4Score,
				}

				out <- plugin.Metric{
					Probe:   p.name + "_ipv6_score",
					Time:    now,
					Latency: result.Comparison.IPv6Score,
				}
			}
		}
	}
}
