package ws

import (
	"encoding/json"
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
	fmt.Println("Clients after add:", h.clients)
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
	fmt.Println("Clients after add:", h.clients)
}

func (h *Hub) Send(userID uint, msg interface{}) {
	h.mu.RLock()
	conns, ok := h.clients[userID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("ws marshal error:", err)
		return
	}

	for conn := range conns {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			fmt.Println("ws write error:", err)
			h.RemoveClient(userID, conn)
			conn.Close()
		}
	}
}

func (h *Hub) SendToMany(userIDs []uint, msg interface{}) {
	for _, id := range userIDs {
		h.Send(id, msg)
	}
}
