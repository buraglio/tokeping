package dns

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
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

	dp := &DNSProbe{
		name:     cfg.Name,
		target:   cfg.Target,
		interval: cfg.Interval,
		protocol: strings.ToLower(proto),
		resolver: cfg.Resolver,
		dohURL:   cfg.DoHURL,
	}

	// Default UDP resolver
	dp.udpRes = net.DefaultResolver

	// TCP resolver over the configured address
	dp.tcpRes = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 2 * time.Second}
			return d.DialContext(ctx, "tcp", dp.resolver)
		},
	}

	// DoT client with proper SNI
	dp.dotClient = &dns.Client{
		Net:     "tcp-tls",
		Timeout: 5 * time.Second,
		TLSConfig: &tls.Config{
			ServerName:         extractHostname(cfg.Resolver),
			InsecureSkipVerify: false,
		},
	}

	// DoH HTTP client
	dp.httpClient = &http.Client{Timeout: 5 * time.Second}

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

			switch p.protocol {
			case "udp":
				_, err = p.udpRes.LookupHost(ctx, p.target)

			case "tcp":
				_, err = p.tcpRes.LookupHost(ctx, p.target)

			case "dot":
				m := new(dns.Msg)
				m.SetQuestion(dns.Fqdn(p.target), dns.TypeA)
				fmt.Fprintf(os.Stderr, "⏱ DoT query %s → %s\n", p.target, p.resolver)
				_, rtt, err2 := p.dotClient.ExchangeContext(ctx, m, p.resolver)
				if err2 != nil {
					err = err2
					fmt.Fprintf(os.Stderr, "❌ DoT error: %v\n", err)
				} else {
					// use the measured round-trip time
					elapsed := rtt
					out <- plugin.Metric{
						Probe:   p.name,
						Time:    time.Now().Unix(),
						Latency: elapsed.Seconds() * 1000,
					}
					continue
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
					fmt.Fprintf(os.Stderr, "❌ DoH error: %v\n", err)
				} else {
					defer resp.Body.Close()
					var result struct{ Answer []interface{} }
					if err2 = json.NewDecoder(resp.Body).Decode(&result); err2 != nil {
						err = err2
						fmt.Fprintf(os.Stderr, "❌ DoH parse error: %v\n", err)
					}
				}

			default:
				err = fmt.Errorf("unknown DNS protocol: %s", p.protocol)
			}

			// Fallback for udp, tcp, doh, or dot errors
			elapsed := time.Since(start)
			ms := elapsed.Seconds() * 1000
			if err != nil {
				ms = -1
			}

			out <- plugin.Metric{
				Probe:   p.name,
				Time:    time.Now().Unix(),
				Latency: ms,
			}
		}
	}
}
