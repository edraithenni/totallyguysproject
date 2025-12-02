package ws

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	"totallyguysproject/internal/models"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
	sendQueueSize  = 128 // per-client outbound queue
	dbQueueSize    = 1024
)

// If NotifID != nil, the DB stored notification with that ID should be deleted on successful send.
type clientMessage struct {
	Data    []byte
	NotifID *uint
}

type Client struct {
	conn *websocket.Conn
	send chan clientMessage
	// closed flag to avoid double-close
	mu     sync.Mutex
	closed bool
}

func (c *Client) close() {
	c.mu.Lock()
	if !c.closed {
		c.closed = true
		close(c.send) // signal writer to stop
		_ = c.conn.Close()
	}
	c.mu.Unlock()
}

type Hub struct {
	// map[userID]map[*websocket.Conn]*Client
	clients map[uint]map[*websocket.Conn]*Client
	mu      sync.RWMutex
	db      *gorm.DB

	dbQueue chan models.Notification
	
	delivered chan uint

	stop chan struct{}
}

func NewHub(db *gorm.DB) *Hub {
	h := &Hub{
		clients:   make(map[uint]map[*websocket.Conn]*Client),
		db:        db,
		dbQueue:   make(chan models.Notification, dbQueueSize),
		delivered: make(chan uint, 1024),
		stop:      make(chan struct{}),
	}

	go h.dbWriter()
	go h.deliveredHandler()
	return h
}

func (h *Hub) AddClient(userID uint, conn *websocket.Conn) *Client {
	client := &Client{
		conn: conn,
		send: make(chan clientMessage, sendQueueSize),
	}

	conn.SetReadLimit(maxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	h.mu.Lock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*websocket.Conn]*Client)
	}
	h.clients[userID][conn] = client
	h.mu.Unlock()

	go h.writePump(userID, client)

	fmt.Printf("User %d connected\n", userID)
	return client
}

func (h *Hub) RemoveClient(userID uint, conn *websocket.Conn) {
	h.mu.Lock()
	if conns, ok := h.clients[userID]; ok {
		if client, exists := conns[conn]; exists {
			client.close()
			delete(conns, conn)
		}
		if len(conns) == 0 {
			delete(h.clients, userID)
		}
	}
	h.mu.Unlock()
	fmt.Printf("User %d disconnected\n", userID)
}

// If offline it enqueues the notification to dbQueue for async persistence.
func (h *Hub) Send(userID uint, msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("ws marshal error:", err)
		return
	}

	h.mu.RLock()
	conns := h.clients[userID]
	h.mu.RUnlock()

	if len(conns) > 0 {
		for conn, client := range conns {
			select {
			case client.send <- clientMessage{Data: data, NotifID: nil}:
			default:
				fmt.Println("client send queue full, removing client:", userID)
				h.RemoveClient(userID, conn)
			}
		}
		return
	}

	notif := models.Notification{
		UserID: userID,
		Type:   msgTypeFrom(msg),
		Data:   string(data),
		Read:   false,
	}
	select {
	case h.dbQueue <- notif:
	default:
		fmt.Println("dbQueue full; dropping notification for user", userID)
	}
}


// It runs in background and saves notifications to DB. When saved it does NOT delete them until delivered.
func (h *Hub) dbWriter() {
	for {
		select {
		case n := <-h.dbQueue:
			if err := h.db.Create(&n).Error; err != nil {
				fmt.Println("failed to save notification:", err)
			}
		case <-h.stop:
			return
		}
	}
}

// deliveredHandler deletes notifications that have been confirmed delivered.
func (h *Hub) deliveredHandler() {
	for {
		select {
		case id := <-h.delivered:
			// delete by id (soft delete)
			if err := h.db.Delete(&models.Notification{}, id).Error; err != nil {
				fmt.Println("failed to delete delivered notification:", err)
			}
		case <-h.stop:
			return
		}
	}
}

// writePump serializes writes to the websocket, pings periodically, and reports delivered DB IDs.
func (h *Hub) writePump(userID uint, client *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = client.conn.Close()
	}()

	for {
		select {
		case cm, ok := <-client.send:
			_ = client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = client.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}
			if err := client.conn.WriteMessage(websocket.TextMessage, cm.Data); err != nil {
				h.RemoveClient(userID, client.conn)
				return
			}
			if cm.NotifID != nil {
				select {
				case h.delivered <- *cm.NotifID:
				default:
					fmt.Println("delivered channel full, skip ack of notification", *cm.NotifID)
				}
			}
		case <-ticker.C:
			_ = client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				h.RemoveClient(userID, client.conn)
				return
			}
		}
	}
}

// SendPendingFromDB loads DB notifications for a user and enqueues them to the client's send queue.
func (h *Hub) SendPendingFromDB(userID uint, conn *websocket.Conn) {
	h.mu.RLock()
	clientMap := h.clients[userID]
	client, ok := clientMap[conn]
	h.mu.RUnlock()
	if !ok {
		return
	}

	var notifs []models.Notification
	if err := h.db.Where("user_id = ? AND deleted_at IS NULL", userID).Find(&notifs).Error; err != nil {
		fmt.Println("failed to load pending notifications:", err)
		return
	}

	for _, n := range notifs {
		cm := clientMessage{Data: []byte(n.Data), NotifID: &n.ID}
		select {
		case client.send <- cm:
		default:
			fmt.Println("client send queue full when flushing DB notifications for user", userID)
			return
		}
	}
}

func (h *Hub) SendToMany(userIDs []uint, msg interface{}) {
	for _, id := range userIDs {
		h.Send(id, msg)
	}
}

// GetClientIDs returns IDs of currently connected users
func (h *Hub) GetClientIDs() []uint {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var ids []uint
	for uid := range h.clients {
		ids = append(ids, uid)
	}
	return ids
}

func (h *Hub) Stop() {
	close(h.stop)
}
 
func StartNotificationCleanup(db *gorm.DB) {
	ticker := time.NewTicker(time.Hour)
	go func() {
		for range ticker.C {
			db.Unscoped().Where("deleted_at IS NOT NULL").Delete(&models.Notification{})
			db.Unscoped().
				Where("deleted_at IS NULL AND created_at < ?", time.Now().AddDate(0, 0, -30)).
				Delete(&models.Notification{})
		}
	}()
}

func msgTypeFrom(msg interface{}) string {
	switch m := msg.(type) {
	case map[string]interface{}:
		if t, ok := m["type"].(string); ok {
			return t
		}
	case []byte:
		var mm map[string]interface{}
		if err := json.Unmarshal(m, &mm); err == nil {
			if t, ok := mm["type"].(string); ok {
				return t
			}
		}
	default:
		b, err := json.Marshal(msg)
		if err == nil {
			var mm map[string]interface{}
			if err := json.Unmarshal(b, &mm); err == nil {
				if t, ok := mm["type"].(string); ok {
					return t
				}
			}
		}
	}
	return "unknown"
}
