package dns

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"tokeping/pkg/plugin"

	"github.com/miekg/dns"
)

// extractHostname strips the “:port” so TLS ServerName is correct
func extractHostname(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

type DNSProbe struct {
	name       string
	target     string
	interval   time.Duration
	protocol   string
	resolver   string
	dohURL     string
	udpRes     *net.Resolver
	tcpRes     *net.Resolver
	dotClient  *dns.Client
	httpClient *http.Client
}

func init() {
	plugin.RegisterProbe("dns", New)
}

func New(cfg plugin.ProbeConfig) (plugin.Probe, error) {
	proto := cfg.Protocol
	if proto == "" {
		proto = "udp"
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	dp := &DNSProbe{
		name:     cfg.Name,
		target:   cfg.Target,
		interval: cfg.Interval,
		protocol: strings.ToLower(proto),
		resolver: cfg.Resolver,
		dohURL:   cfg.DoHURL,
	}

	dp.udpRes = net.DefaultResolver

	dp.tcpRes = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: timeout}
			return d.DialContext(ctx, "tcp", dp.resolver)
		},
	}

	dp.dotClient = &dns.Client{
		Net:     "tcp-tls",
		Timeout: timeout,
		TLSConfig: &tls.Config{
			ServerName:         extractHostname(cfg.Resolver),
			InsecureSkipVerify: false,
		},
	}

	dp.httpClient = &http.Client{Timeout: timeout}

	return dp, nil
}

func (p *DNSProbe) Name() string            { return p.name }
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
			var err error
			var latencyMs float64

			switch p.protocol {
			case "udp":
				_, err = p.udpRes.LookupHost(ctx, p.target)
				latencyMs = time.Since(start).Seconds() * 1000

			case "tcp":
				_, err = p.tcpRes.LookupHost(ctx, p.target)
				latencyMs = time.Since(start).Seconds() * 1000

			case "dot":
				m := new(dns.Msg)
				m.SetQuestion(dns.Fqdn(p.target), dns.TypeA)
				_, rtt, err2 := p.dotClient.ExchangeContext(ctx, m, p.resolver)
				err = err2
				if err == nil {
					latencyMs = rtt.Seconds() * 1000
				}

			case "doh":
				req, _ := http.NewRequestWithContext(ctx, "GET", p.dohURL, nil)
				q := req.URL.Query()
				q.Set("name", p.target)
				q.Set("type", "A")
				req.URL.RawQuery = q.Encode()
				req.Header.Set("Accept", "application/dns-json")

				resp, err2 := p.httpClient.Do(req)
				if err2 != nil {
					err = err2
				} else {
					defer resp.Body.Close()
					var result struct{ Answer []interface{} }
					if err2 = json.NewDecoder(resp.Body).Decode(&result); err2 != nil {
						err = err2
					}
					latencyMs = time.Since(start).Seconds() * 1000
				}

			default:
				err = fmt.Errorf("unknown DNS protocol: %s", p.protocol)
			}

			if err != nil {
				continue
			}

			out <- plugin.Metric{
				Probe:   p.name,
				Time:    time.Now().Unix(),
				Latency: latencyMs,
			}
		}
	}
}
