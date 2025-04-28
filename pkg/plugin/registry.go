package plugin

import (
    "fmt"
    "tokeping/pkg/config"
)

type ProbeConfig = config.ProbeConfig
type OutputConfig = config.OutputConfig

var (
    probeFactories  = make(map[string]func(ProbeConfig) (Probe, error))
    outputFactories = make(map[string]func(OutputConfig) (Output, error))
)

func RegisterProbe(typ string, factory func(ProbeConfig) (Probe, error)) {
    probeFactories[typ] = factory
}

func NewProbe(cfg ProbeConfig) (Probe, error) {
    f, ok := probeFactories[cfg.Type]
    if !ok {
        return nil, fmt.Errorf("unknown probe type: %s", cfg.Type)
    }
    return f(cfg)
}

func RegisterOutput(typ string, factory func(OutputConfig) (Output, error)) {
    outputFactories[typ] = factory
}

func NewOutput(cfg OutputConfig) (Output, error) {
    f, ok := outputFactories[cfg.Type]
    if !ok {
        return nil, fmt.Errorf("unknown output type: %s", cfg.Type)
    }
    return f(cfg)
}
