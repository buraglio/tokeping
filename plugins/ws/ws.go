package ws

import (
    "net/http"
    "sync"

    "github.com/gorilla/websocket"
    "tokeping/pkg/plugin"
)

type WSOutput struct {
    addr     string
    clients  map[*websocket.Conn]bool
    mu       sync.Mutex
    upgrader websocket.Upgrader
}

func init() {
    plugin.RegisterOutput("ws", New)
}

func New(cfg plugin.OutputConfig) (plugin.Output, error) {
    return &WSOutput{
        addr:    cfg.Listen,
        clients: make(map[*websocket.Conn]bool),
        upgrader: websocket.Upgrader{
            CheckOrigin: func(r *http.Request) bool { return true },
        },
    }, nil
}

func (w *WSOutput) Name() string { return "ws" }
func (w *WSOutput) Start() error {
    http.HandleFunc("/ws", w.handleWS)
    go http.ListenAndServe(w.addr, nil)
    return nil
}
func (w *WSOutput) handleWS(rw http.ResponseWriter, req *http.Request) {
    conn, err := w.upgrader.Upgrade(rw, req, nil)
    if err != nil {
        return
    }
    w.mu.Lock()
    w.clients[conn] = true
    w.mu.Unlock()
}
func (w *WSOutput) Send(m plugin.Metric) {
    w.mu.Lock()
    defer w.mu.Unlock()
    for c := range w.clients {
        c.WriteJSON(m)
    }
}
func (w *WSOutput) Stop() error {
    w.mu.Lock()
    defer w.mu.Unlock()
    for c := range w.clients {
        c.Close()
    }
    return nil
}
