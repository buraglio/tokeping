package file

import (
    "fmt"
    "os"
    "sync"

    "tokeping/pkg/plugin"
)

type FileOutput struct {
    path string
    mu   sync.Mutex
    file *os.File
}

func init() {
    plugin.RegisterOutput("file", New)
}

func New(cfg plugin.OutputConfig) (plugin.Output, error) {
    f, err := os.OpenFile(cfg.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return nil, err
    }
    return &FileOutput{path: cfg.Path, file: f}, nil
}

func (o *FileOutput) Name() string { return "file" }
func (o *FileOutput) Start() error { return nil }
func (o *FileOutput) Send(m plugin.Metric) {
    o.mu.Lock()
    defer o.mu.Unlock()
    line := fmt.Sprintf("%d,%s,%.3f
", m.Time, m.Probe, m.Latency)
    o.file.WriteString(line)
}
func (o *FileOutput) Stop() error {
    return o.file.Close()
}
