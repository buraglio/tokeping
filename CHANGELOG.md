# Changelog

## Latest Updates

### Performance and Reliability Improvements (Lots of not great code in there, should be fixed. yay!)

#### Fixed Resource Leaks
- **Ping Probe**: Fixed resource leak where pinger instances were recreated every interval
  - Now reuses single pinger instance per probe
  - Proper cleanup with deferred Stop() call
  - Eliminates goroutine and file descriptor leaks

#### Improved Error Handling
- **MTR Probe**: Added error counter with max threshold (5 errors)
  - Stops emitting invalid metrics after repeated failures
  - Clean error reporting without polluting metric streams

- **DNS Probe**: Errors no longer emit metrics (previously sent -1 latency)
  - Failed queries are now silently skipped
  - Consistent timing methodology across all protocols (UDP/TCP/DoT/DoH)

#### Non-Blocking Architecture
- **Daemon Output Delivery**: Metrics now sent to outputs in separate goroutines
  - 5-second timeout per output prevents blocking
  - Slow or failing outputs no longer impact other outputs or probes
  - Significantly improves reliability under load

#### Configurable Parameters
- **Metric Buffer**: New `metric_buffer` config option (default: 100)
  - Prevents probe blocking when outputs are slow
  - Configurable per deployment needs

- **Probe Timeouts**: All probes now support `timeout` parameter
  - Ping: default 5s
  - DNS: default 5s
  - ProtoTester: default 10s

- **MTR Count**: Configurable probe count via `count` parameter (default: 5)

### New Features

#### ProtoTester Integration
- New plugin for IPv4/IPv6 comparison testing
- Integrates with [prototester](https://github.com/buraglio/prototester)
- Emits 4 metrics per test:
  - `{name}_ipv4` - IPv4 average latency
  - `{name}_ipv6` - IPv6 average latency
  - `{name}_ipv4_score` - IPv4 performance score
  - `{name}_ipv6_score` - IPv6 performance score
- Ideal for dual-stack deployment validation

### Code Quality
- Removed all emoji from log messages
- Fixed YAML config duplicate keys
- Added comprehensive error handling
- Improved code formatting and consistency
- Updated README with complete configuration documentation

### Breaking Changes
- Output plugin interface now requires `context.Context` parameter
  - Old: `Send(m Metric)`
  - New: `Send(ctx context.Context, m Metric)`
- DNS probe no longer emits -1 latency on errors (skips metric emission instead)
- MTR probe stops emitting metrics after 5 consecutive errors

### Migration Notes
If you have custom output plugins, update the `Send` method signature:
```go
// Old
func (o *CustomOutput) Send(m plugin.Metric) {
    // ...
}

// New
func (o *CustomOutput) Send(ctx context.Context, m plugin.Metric) {
    // ...
}
```

All built-in plugins have been updated automatically.
