package plugin

type Output interface {
    Name() string
    Start() error
    Send(m Metric)
    Stop() error
}
