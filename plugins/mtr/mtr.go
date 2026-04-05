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

// MTRProbe performs traceroute-like latency measurements using the mtr binary.
// Emits one metric per hop as "{probeName}_{hopHost}" with average latency in ms.

type MTRProbe struct {
	name       string
	target     string
	interval   time.Duration
	count      int
	ipv6       bool
	mtrPath    string
	errorCount int
	maxErrors  int
}

func init() {
	plugin.RegisterProbe("mtr", New)
}

// New creates a new MTRProbe. It defaults to IPv6, falling back to IPv4 if no IPv6 addresses are found.
func New(cfg plugin.ProbeConfig) (plugin.Probe, error) {
	// Default to IPv6
	ipv6 := true
	// Check if target is a literal IP
	if ip := net.ParseIP(cfg.Target); ip != nil {
		if ip.To4() != nil {
			ipv6 = false
		}
	} else {
		// Domain name: lookup addresses
		addrs, err := net.LookupIP(cfg.Target)
		if err == nil {
			has6, has4 := false, false
			for _, addr := range addrs {
				if addr.To4() != nil {
					has4 = true
				} else {
					has6 = true
				}
			}
			if has6 {
				ipv6 = true
			} else if has4 {
				ipv6 = false
			}
		}
	}

	// Locate mtr binary
	mtrExe, err := exec.LookPath("mtr")
	if err != nil {
		return nil, fmt.Errorf("mtr not found in PATH: %v", err)
	}

	count := cfg.Count
	if count == 0 {
		count = 5
	}
	return &MTRProbe{
		name:       cfg.Name,
		target:     cfg.Target,
		interval:   cfg.Interval,
		count:      count,
		ipv6:       ipv6,
		mtrPath:    mtrExe,
		errorCount: 0,
		maxErrors:  5,
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
			fmt.Fprintf(os.Stderr, "running MTR probe %q -> %s (ipv6=%t)\n", p.name, p.target, p.ipv6)

			// Build arguments
			args := []string{"-r", "-c", strconv.Itoa(p.count)}
			if p.ipv6 {
				args = append(args, "-6")
			} else {
				args = append(args, "-4")
			}
			args = append(args, p.target)

			// Execute mtr
			evalCtx, cancel := context.WithTimeout(ctx, p.interval)
			cmd := exec.CommandContext(evalCtx, p.mtrPath, args...)
			output, err := cmd.CombinedOutput()
			cancel()
			if err != nil {
				p.errorCount++
				fmt.Fprintf(os.Stderr, "❌ mtr error for %q (%d/%d): %v\nOutput: %s\n", p.name, p.errorCount, p.maxErrors, err, output)
				if p.ipv6 {
					fmt.Fprintf(os.Stderr, "⚠️ falling back to IPv4 for %q next tick\n", p.name)
					p.ipv6 = false
				}
				if p.errorCount >= p.maxErrors {
					fmt.Fprintf(os.Stderr, "mtr probe %q: max errors reached, skipping metric emission\n", p.name)
				} else {
					out <- plugin.Metric{Probe: p.name, Time: time.Now().Unix(), Latency: -1}
				}
				continue
			}
			p.errorCount = 0

			// Parse output
			scanner := bufio.NewScanner(strings.NewReader(string(output)))
			emitted := 0
			for scanner.Scan() {
				line := scanner.Text()
				trimmed := strings.TrimSpace(line)
				if trimmed == "" ||
					strings.HasPrefix(trimmed, "HOST:") ||
					strings.HasPrefix(trimmed, "Start:") ||
					strings.Contains(trimmed, "Loss%") ||
					strings.HasPrefix(trimmed, "My traceroute") {
					continue
				}
				fields := strings.Fields(line)
				if len(fields) < 6 {
					continue
				}
				hop := fields[1]
				avgStr := fields[5]
				avg, err := strconv.ParseFloat(avgStr, 64)
				if err != nil {
					continue
				}
				safeHop := strings.ReplaceAll(hop, "/", "_")
				tag := fmt.Sprintf("%s_%s", p.name, safeHop)
				out <- plugin.Metric{Probe: tag, Time: time.Now().Unix(), Latency: avg}
				emitted++
			}
			if err := scanner.Err(); err != nil {
				fmt.Fprintf(os.Stderr, "scan error for %q: %v\n", p.name, err)
			}
			if emitted == 0 {
				fmt.Fprintf(os.Stderr, "no hops parsed for %q\n", p.name)
			}
		}
	}
}
