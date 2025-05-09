package mtr

import (
    "bufio"
    "context"
    "fmt"
    "net"
    "os"
    "os/exec"
    "strconv"
    "strings"
    "time"

    "tokeping/pkg/plugin"
)

func init() {
    plugin.RegisterProbe("mtr", New)
}

type MTRProbe struct {
    name     string
    target   string
    interval time.Duration
    count    int
    ipv6     bool
}

func New(cfg plugin.ProbeConfig) (plugin.Probe, error) {
    // default to 5 cycles
    count := 5
    // detect IPv6 by presence of colon or non-IPv4 address
    ipv6 := false
    if ip := net.ParseIP(cfg.Target); ip != nil && ip.To4() == nil {
        ipv6 = true
    }
    return &MTRProbe{
        name:     cfg.Name,
        target:   cfg.Target,
        interval: cfg.Interval,
        count:    count,
        ipv6:     ipv6,
    }, nil
}

func (p *MTRProbe) Name() string {
    return p.name
}

func (p *MTRProbe) Interval() time.Duration {
    return p.interval
}

func (p *MTRProbe) Run(ctx context.Context, out chan<- plugin.Metric) {
    ticker := time.NewTicker(p.interval)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            // build mtr args
            args := []string{"-r", "-c", strconv.Itoa(p.count)}
            if p.ipv6 {
                args = append(args, "-6")
            } else {
                args = append(args, "-4")
            }
            args = append(args, p.target)
            cmd := exec.CommandContext(ctx, "mtr", args...)
            output, err := cmd.Output()
            if err != nil {
                fmt.Fprintf(os.Stderr, "❌ mtr error: %v\n", err)
                out <- plugin.Metric{Probe: p.name, Time: time.Now().Unix(), Latency: -1}
                continue
            }
            scanner := bufio.NewScanner(strings.NewReader(string(output)))
            for scanner.Scan() {
                line := scanner.Text()
                // skip headers and non-data lines
                if strings.Contains(line, "Loss%") || strings.HasPrefix(line, "HOST:") || strings.HasPrefix(line, "Start:") || strings.HasPrefix(line, "My traceroute") || strings.TrimSpace(line) == "" {
                    continue
                }
                fields := strings.Fields(line)
                if len(fields) < 6 {
                    continue
                }
                host := fields[1]
                avgStr := fields[5]
                avg, err := strconv.ParseFloat(avgStr, 64)
                if err != nil {
                    avg = -1
                }
                probeName := fmt.Sprintf("%s/%s", p.name, host)
                out <- plugin.Metric{
                    Probe:   probeName,
                    Time:    time.Now().Unix(),
                    Latency: avg,
                }
            }
        }
    }
}
