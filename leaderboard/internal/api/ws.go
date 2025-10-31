package api

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	// for dev: allow all origins
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WSHandler upgrades the HTTP connection to a WebSocket connection
// and registers it with the Hub. It also listens for client disconnect.
func (h *Handlers) WSHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.Hub.Register(conn)

	// We don't actually care what the client sends us right now,
	// but we need to read from the socket to detect when it disconnects.
	go func() {
		defer h.Hub.Unregister(conn)
		for {
			if _, _, err := conn.NextReader(); err != nil {
				// client disconnected
				return
			}
		}
	}()
}
