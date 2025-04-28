package daemon

import (
    "context"

    "tokeping/pkg/config"
    "tokeping/pkg/plugin"
)

type Daemon struct {
    cfg    *config.Config
    outCh  chan plugin.Metric
    ctx    context.Context
    cancel context.CancelFunc
}

func New(cfg *config.Config) (*Daemon, error) {
    ctx, cancel := context.WithCancel(context.Background())
    return &Daemon{
        cfg:    cfg,
        outCh:  make(chan plugin.Metric, 100),
        ctx:    ctx,
        cancel: cancel,
    }, nil
}

func (d *Daemon) Run(parent context.Context) {
    go func() {
        <-parent.Done()
        d.cancel()
    }()

    outputs := []plugin.Output{}
    for _, o := range d.cfg.Outputs {
        out, err := plugin.NewOutput(o)
        if err != nil {
            continue
        }
        out.Start()
        outputs = append(outputs, out)
    }

    for _, p := range d.cfg.Probes {
        pr, err := plugin.NewProbe(p)
        if err != nil {
            continue
        }
        go pr.Run(d.ctx, d.outCh)
    }

    for {
        select {
        case <-d.ctx.Done():
            for _, out := range outputs {
                out.Stop()
            }
            return
        case m := <-d.outCh:
            for _, out := range outputs {
                out.Send(m)
            }
        }
    }
}

func (d *Daemon) Stop() {
    d.cancel()
}
