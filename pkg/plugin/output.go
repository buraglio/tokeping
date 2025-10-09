package plugin

import "context"

type Output interface {
	Name() string
	Start() error
	Send(ctx context.Context, m Metric)
	Stop() error
}
