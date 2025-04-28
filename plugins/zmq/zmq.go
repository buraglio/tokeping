package zmq

import (
    "encoding/json"

    zmq "github.com/pebbe/zmq4"
    "tokeping/pkg/plugin"
)

type ZMQOutput struct {
    socket *zmq.Socket
}

func init() {
    plugin.RegisterOutput("zmq", New)
}

func New(cfg plugin.OutputConfig) (plugin.Output, error) {
    sock, err := zmq.NewSocket(zmq.PUB)
    if err != nil {
        return nil, err
    }
    if err := sock.Bind(cfg.Listen); err != nil {
        return nil, err
    }
    return &ZMQOutput{socket: sock}, nil
}

func (o *ZMQOutput) Name() string { return "zmq" }
func (o *ZMQOutput) Start() error { return nil }
func (o *ZMQOutput) Send(m plugin.Metric) {
    if b, err := json.Marshal(m); err == nil {
        o.socket.SendBytes(b, 0)
    }
}
func (o *ZMQOutput) Stop() error {
    return o.socket.Close()
}
