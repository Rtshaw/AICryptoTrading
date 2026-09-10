// Package ws provides a small pub/sub hub that broadcasts JSON messages
// (candle updates, AI signals, orders, positions, balance, auto-trade log
// entries, funding updates, settings changes) to every connected
// /ws/market client.
package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type string `json:"type"` // "candle" | "signal" | "order" | "position" | "balance" | "autotrade_log" | "funding" | "settings"
	Data any    `json:"data"`
}

type Hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]struct{}
	upgrade websocket.Upgrader
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]struct{}),
		upgrade: websocket.Upgrader{
			// Local single-user tool: no cross-origin browser will legitimately
			// hit this socket, so allow any origin rather than force users to
			// configure CORS for their own dev frontend.
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrade.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade failed: %v", err)
		return
	}

	h.mu.Lock()
	h.clients[conn] = struct{}{}
	h.mu.Unlock()

	go func() {
		defer h.remove(conn)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}

func (h *Hub) remove(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
	_ = conn.Close()
}

func (h *Hub) Broadcast(msg Message) {
	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("ws marshal failed: %v", err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			go h.remove(conn)
		}
	}
}
