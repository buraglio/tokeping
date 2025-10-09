package daemon

import (
	"context"
	"fmt"
	"os"
	"time"

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
	bufferSize := cfg.MetricBuffer
	if bufferSize == 0 {
		bufferSize = 100
	}
	return &Daemon{
		cfg:    cfg,
		outCh:  make(chan plugin.Metric, bufferSize),
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
			fmt.Fprintf(os.Stderr, "output %q failed to register: %v\n", o.Name, err)
			continue
		}
		fmt.Printf("starting output %q (type=%s)\n", o.Name, o.Type)
		if err := out.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "output %q Start() error: %v\n", o.Name, err)
		}
		outputs = append(outputs, out)
	}

	for _, pCfg := range d.cfg.Probes {
		pr, err := plugin.NewProbe(pCfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "probe %q failed to register: %v\n", pCfg.Name, err)
			continue
		}
		fmt.Fprintf(os.Stderr,
			"loaded probe: name=%q, type=%q, target=%q\n",
			pr.Name(), pCfg.Type, pCfg.Target,
		)

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
				go func(o plugin.Output, metric plugin.Metric) {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					o.Send(ctx, metric)
				}(out, m)
			}
		}
	}
}

func (d *Daemon) Stop() {
	d.cancel()
}
