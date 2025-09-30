package ws

import (
	"fmt"
	"sync"
	"github.com/gorilla/websocket"
)

type Hub struct {
	clients map[uint]map[*websocket.Conn]bool
	mu      sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[uint]map[*websocket.Conn]bool),
	}
}

func (h *Hub) AddClient(userID uint, conn *websocket.Conn) {
	h.mu.Lock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*websocket.Conn]bool)
	}
	h.clients[userID][conn] = true
	h.mu.Unlock()
	fmt.Printf("User %d connected\n", userID)
}

func (h *Hub) RemoveClient(userID uint, conn *websocket.Conn) {
	h.mu.Lock()
	if conns, ok := h.clients[userID]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(h.clients, userID)
		}
	}
	h.mu.Unlock()
	fmt.Printf("User %d disconnected\n", userID)
}

func (h *Hub) Send(userID uint, msg string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if conns, ok := h.clients[userID]; ok {
		for conn := range conns {
			conn.WriteJSON(map[string]string{"message": msg})
		}
	}
}

func (h *Hub) SendToMany(userIDs []uint, msg string) {
	for _, id := range userIDs {
		h.Send(id, msg)
	}
}
