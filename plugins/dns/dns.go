package dns

import (
    "context"
    "crypto/tls"
    "encoding/json"
    "net"
    "net/http"
    "time"

    "github.com/miekg/dns"
    "tokeping/pkg/plugin"
)

type DNSProbe struct {
    name     string
    target   string
    interval time.Duration
    protocol string
    resolver string
    dohURL   string
    udpRes   *net.Resolver
    tcpRes   *net.Resolver
    dotClient *dns.Client
    httpClient *http.Client
}

func init() {
    plugin.RegisterProbe("dns", New)
}

func New(cfg plugin.ProbeConfig) (plugin.Probe, error) {
    // default to UDP
    proto := cfg.Protocol
    if proto == "" {
        proto = "udp"
    }

    dp := &DNSProbe{
        name:     cfg.Name,
        target:   cfg.Target,
        interval: cfg.Interval,
        protocol: proto,
        resolver: cfg.Resolver,
        dohURL:   cfg.DoHURL,
    }

    // set up UDP and TCP resolvers
    dp.udpRes = net.DefaultResolver
    dp.tcpRes = &net.Resolver{
        PreferGo: true,
        Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
            // network will be "tcp" here
            d := net.Dialer{Timeout: 2 * time.Second}
            return d.DialContext(ctx, "tcp", dp.resolver)
        },
    }

    // set up DoT client
    dp.dotClient = &dns.Client{
        Net:       "tcp-tls",
        Timeout:   5 * time.Second,
        TLSConfig: &tls.Config{ServerName: cfg.Resolver}, // SNI
    }

    // set up DoH HTTP client (JSON API)
    dp.httpClient = &http.Client{Timeout: 5 * time.Second}

    return dp, nil
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
            var err error

            switch p.protocol {
            case "udp":
                _, err = p.udpRes.LookupHost(ctx, p.target)
            case "tcp":
                _, err = p.tcpRes.LookupHost(ctx, p.target)
            case "dot":
                // build DNS message
                m := new(dns.Msg)
                m.SetQuestion(dns.Fqdn(p.target), dns.TypeA)
                _, _, err = p.dotClient.ExchangeContext(ctx, m, p.resolver)
            case "doh":
                // DoH JSON API: e.g. https://cloudflare-dns.com/dns-query?name=example.com&type=A
                req, _ := http.NewRequestWithContext(ctx, "GET", p.dohURL, nil)
                q := req.URL.Query()
                q.Set("name", p.target)
                q.Set("type", "A")
                req.URL.RawQuery = q.Encode()
                req.Header.Set("Accept", "application/dns-json")

                resp, e := p.httpClient.Do(req)
                if e == nil {
                    defer resp.Body.Close()
                    // ignore parsing details—just unmarshal to ensure valid JSON
                    var result struct{ Answer []interface{} }
                    e = json.NewDecoder(resp.Body).Decode(&result)
                }
                err = e
            default:
                err = context.DeadlineExceeded
            }

            elapsed := time.Since(start).Seconds() * 1000 // ms
            if err != nil {
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
