package daemon

import (
	"context"
	"fmt"
	"os"

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
			fmt.Fprintf(os.Stderr, "⚠️  output %q failed to register: %v\n", o.Name, err)
			continue
		}
		fmt.Printf("➡️  starting output %q (type=%s)\n", o.Name, o.Type)
		if err := out.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  output %q Start() error: %v\n", o.Name, err)
		}
		outputs = append(outputs, out)
	}

	for _, pCfg := range d.cfg.Probes {
		pr, err := plugin.NewProbe(pCfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️ probe %q failed to register: %v\n", pCfg.Name, err)
			continue
		}
		// <-- add your debug here:
		fmt.Fprintf(os.Stderr,
			"🔍 Loaded probe: name=%q, type=%q, target=%q\n",
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
				out.Send(m)
			}
		}
	}
}

func (d *Daemon) Stop() {
	d.cancel()
}
