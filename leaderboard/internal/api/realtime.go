package api

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type LeaderboardUpdate struct {
	Username string `json:"username"`
	Score    int64  `json:"score"`
}

// tracks active websocket clients and lets us broadcast them
type Hub struct {
	mu        sync.Mutex
	clients   map[*websocket.Conn]bool
	broadcast chan LeaderboardUpdate
}

func NewHub() *Hub {
	return &Hub{
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan LeaderboardUpdate, 128),
	}
}

func (h *Hub) Run() {
	for msg := range h.broadcast {
		h.mu.Lock()
		for c := range h.clients {
			if err := c.WriteJSON(msg); err != nil {
				// client is dead, drop it
				log.Println("ws write failed, removing client:", err)
				c.Close()
				delete(h.clients, c)
			}
		}
		h.mu.Unlock()
	}
}

// Register adds a client to the hub
func (h *Hub) Register(conn *websocket.Conn) {
	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()
}

func (h *Hub) Unregister(conn *websocket.Conn) {
	h.mu.Lock()
	if _, ok := h.clients[conn]; ok {
		conn.Close()
		delete(h.clients, conn)
	}
	h.mu.Unlock()
}

// Broadcast queues a message to everyone
func (h *Hub) Broadcast(u LeaderboardUpdate) {
	select {
	case h.broadcast <- u:
	default:
		// channel full, drop instead of blocking the server
		log.Println("broadcast channel full, dropping message")
	}
}
