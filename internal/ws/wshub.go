package ws

import (
	"encoding/json"
	"fmt"
	"sync"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	"totallyguysproject/internal/models"
	"time"
)

type Hub struct {
	clients map[uint]map[*websocket.Conn]bool
	mu      sync.RWMutex
	db      *gorm.DB
}

func NewHub(db *gorm.DB) *Hub {
	return &Hub{
		clients: make(map[uint]map[*websocket.Conn]bool),
		db:      db,
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

func (h *Hub) Send(userID uint, msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("ws marshal error:", err)
		return
	}

	h.mu.RLock()
	conns, online := h.clients[userID]
	h.mu.RUnlock()
	//if online - send without db caching, else - store in db until target user is online
	if online {
		for conn := range conns {
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				fmt.Println("ws write error:", err)
				h.RemoveClient(userID, conn)
				conn.Close()
			}
		}
	} else {
		notif := models.Notification{
			UserID: userID,
			Type:   msgTypeFrom(msg),
			Data:   string(data),
			Read:   false,
		}
		if err := h.db.Create(&notif).Error; err != nil {
			fmt.Println("failed to save notification:", err)
		}
	}
}

func (h *Hub) SendPendingFromDB(userID uint, conn *websocket.Conn) {
	var notifs []models.Notification
	if err := h.db.Where("user_id = ? AND deleted_at IS NULL", userID).Find(&notifs).Error; err == nil {
		for _, n := range notifs {
			conn.WriteMessage(websocket.TextMessage, []byte(n.Data))
			h.db.Delete(&n) // soft delete, StartNotificationCleanup() for deleting garbage every hour
		}
	}
}

func (h *Hub) SendToMany(userIDs []uint, msg interface{}) {
	for _, id := range userIDs {
		h.Send(id, msg)
	}
}

func (h *Hub) GetClientIDs() []uint {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var ids []uint
	for uid := range h.clients {
		ids = append(ids, uid)
	}
	return ids
}

func StartNotificationCleanup(db *gorm.DB) {
	ticker := time.NewTicker(time.Hour)
	go func() {
		for range ticker.C {
			db.Unscoped().Where("deleted_at IS NOT NULL").Delete(&models.Notification{})
		}
	}()
}

func msgTypeFrom(msg interface{}) string {
	if m, ok := msg.(map[string]interface{}); ok {
		if t, exists := m["type"].(string); exists {
			return t
		}
	}
	return "unknown"
}

