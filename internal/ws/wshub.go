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

// Tunables (adjust to your needs)
const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
	sendQueueSize  = 128 // per-client outbound queue
	dbQueueSize    = 1024
)

// clientMessage is an envelope sent to a client's writer. NotifID is nil for non-DB messages.
// If NotifID != nil, the DB stored notification with that ID should be deleted on successful send.
type clientMessage struct {
	Data    []byte
	NotifID *uint
}

// Client wraps a websocket.Conn and serializes writes via a single writer goroutine.
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

// Hub manages clients and notification persistence/delivery.
type Hub struct {
	// map[userID]map[*websocket.Conn]*Client
	clients map[uint]map[*websocket.Conn]*Client
	mu      sync.RWMutex
	db      *gorm.DB

	// async DB writer queue (persist notifications for offline users)
	dbQueue chan models.Notification

	// delivered channel: writePump reports delivered notification IDs here so hub can delete them.
	delivered chan uint

	// stop channel for worker goroutines (optional, useful for tests/graceful shutdown)
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
	// start background workers
	go h.dbWriter()
	go h.deliveredHandler()
	return h
}

// AddClient registers the connection and starts its writer (writePump).
func (h *Hub) AddClient(userID uint, conn *websocket.Conn) *Client {
	client := &Client{
		conn: conn,
		send: make(chan clientMessage, sendQueueSize),
	}
	// set read limits & pong handler
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

// RemoveClient removes and closes a client.
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

// Send sends msg to a user. If user is online it enqueues to their clients' send queues.
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
			// non-blocking enqueue: if client's send buffer is full, remove client to protect server.
			select {
			case client.send <- clientMessage{Data: data, NotifID: nil}:
			default:
				// too slow - remove this client
				fmt.Println("client send queue full, removing client:", userID)
				h.RemoveClient(userID, conn)
			}
		}
		return
	}

	// offline: persist notification asynchronously
	notif := models.Notification{
		UserID: userID,
		Type:   msgTypeFrom(msg),
		Data:   string(data),
		Read:   false,
	}
	select {
	case h.dbQueue <- notif:
		// queued for persistence
	default:
		// DB queue full — log & drop to avoid blocking. You can choose to fallback to direct write here.
		fmt.Println("dbQueue full; dropping notification for user", userID)
	}
}

// dbWriter persists notifications enqueued for offline users.
// It runs in background and saves notifications to DB. When saved it does NOT delete them until delivered.
func (h *Hub) dbWriter() {
	for {
		select {
		case n := <-h.dbQueue:
			if err := h.db.Create(&n).Error; err != nil {
				fmt.Println("failed to save notification:", err)
				// on failure you might retry, but be careful to avoid tight loops
			}
		case <-h.stop:
			return
		}
	}
}

// deliveredHandler deletes notifications that have been confirmed delivered.
// writePump will send NotifID into h.delivered after successful socket write.
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
				// channel closed
				_ = client.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}
			if err := client.conn.WriteMessage(websocket.TextMessage, cm.Data); err != nil {
				// on write error, remove the client (RemoveClient closes resources)
				h.RemoveClient(userID, client.conn)
				return
			}
			// if this message corresponds to a persisted notification, report delivered
			if cm.NotifID != nil {
				select {
				case h.delivered <- *cm.NotifID:
				default:
					// if delivered channel were blocked (unlikely), log and continue
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
// It only enqueues messages; actual deletion occurs when writePump confirms delivery via h.delivered.
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
		// Prepare envelope with NotifID to delete after confirmed send
		cm := clientMessage{Data: []byte(n.Data), NotifID: &n.ID}
		select {
		case client.send <- cm:
			// enqueued; deletion will happen after writer confirms
		default:
			// client's queue full; stop flushing now to avoid hogging memory.
			fmt.Println("client send queue full when flushing DB notifications for user", userID)
			return
		}
	}
}

// SendToMany convenience
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

// Stop gracefully shuts down workers (useful for tests or graceful shutdown)
func (h *Hub) Stop() {
	close(h.stop)
}
 
// StartNotificationCleanup remains similar; keep or adapt TTL as needed.
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

// msgTypeFrom same helpful function as before.
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