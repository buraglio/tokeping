package ws

import (
    "fmt"
    "os"
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
    // serve static UI too
    http.Handle("/", http.FileServer(http.Dir("web/static")))
    http.HandleFunc("/ws", w.handleWS)

    fmt.Printf("🌐  HTTP/ws server listening on %s\n", w.addr)
    go func() {
        if err := http.ListenAndServe(w.addr, nil); err != nil {
            fmt.Fprintf(os.Stderr, "❌ HTTP server error: %v\n", err)
        }
    }()
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
