package terminal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"wws/api/internal/db"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // TODO: Implement proper origin checking
	},
}

// TerminalSession represents a shared terminal session
type TerminalSession struct {
	ID          int       `db:"id" json:"id"`
	WorkspaceID int       `db:"workspace_id" json:"workspace_id"`
	SessionType string    `db:"session_type" json:"session_type"` // "tmux", "direct"
	IsActive    bool      `db:"is_active" json:"is_active"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// TerminalParticipant represents a user participating in a terminal session
type TerminalParticipant struct {
	ID             int          `db:"id" json:"id"`
	SessionID      int          `db:"session_id" json:"session_id"`
	UserID         int          `db:"user_id" json:"user_id"`
	Username       string       `db:"username" json:"username"`
	Permission     string       `db:"permission" json:"permission"` // "viewer" or "editor"
	Connected      bool         `db:"connected" json:"connected"`
	ConnectedAt    sql.NullTime `db:"connected_at" json:"connected_at"`
	DisconnectedAt sql.NullTime `db:"disconnected_at" json:"disconnected_at"`
}

// TerminalMessage represents a message in the terminal session
type TerminalMessage struct {
	Type      string    `json:"type"` // "input", "output", "user-join", "user-leave", "cursor"
	SessionID int       `json:"session_id"`
	UserID    int       `json:"user_id,omitempty"`
	Username  string    `json:"username,omitempty"`
	Data      string    `json:"data,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Client represents a WebSocket client connected to a terminal session
type Client struct {
	ID         int
	SessionID  int
	UserID     int
	Username   string
	Permission string
	Conn       *websocket.Conn
	Send       chan []byte
	Hub        *TerminalHub
}

// TerminalHub manages WebSocket connections for terminal sessions
type TerminalHub struct {
	clients    map[int]*Client
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

var hub = &TerminalHub{
	clients:    make(map[int]*Client),
	broadcast:  make(chan []byte, 256),
	register:   make(chan *Client),
	unregister: make(chan *Client),
}

// StartHub starts the terminal hub goroutine
func StartHub() {
	go hub.run()
}

func (h *TerminalHub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.mu.Unlock()

			// Notify other clients
			hub.broadcastUserJoin(client)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Send)
				hub.broadcastUserLeave(client)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client.ID)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *TerminalHub) broadcastUserJoin(client *Client) {
	msg := TerminalMessage{
		Type:      "user-join",
		SessionID: client.SessionID,
		UserID:    client.UserID,
		Username:  client.Username,
		Timestamp: time.Now(),
	}
	data, _ := json.Marshal(msg)
	h.broadcast <- data
}

func (h *TerminalHub) broadcastUserLeave(client *Client) {
	msg := TerminalMessage{
		Type:      "user-leave",
		SessionID: client.SessionID,
		UserID:    client.UserID,
		Username:  client.Username,
		Timestamp: time.Now(),
	}
	data, _ := json.Marshal(msg)
	h.broadcast <- data
}

// GetSessionParticipants retrieves all participants for a terminal session
func GetSessionParticipants(ctx context.Context, sessionID int) ([]TerminalParticipant, error) {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT id, session_id, user_id, username, permission, connected, connected_at, disconnected_at
		FROM terminal_participants WHERE session_id = ?
		ORDER BY connected_at DESC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query participants: %w", err)
	}
	defer rows.Close()

	var participants []TerminalParticipant
	for rows.Next() {
		var p TerminalParticipant
		if err := rows.Scan(&p.ID, &p.SessionID, &p.UserID, &p.Username,
			&p.Permission, &p.Connected, &p.ConnectedAt, &p.DisconnectedAt); err != nil {
			return nil, fmt.Errorf("failed to scan participant: %w", err)
		}
		participants = append(participants, p)
	}

	return participants, nil
}

// CreateTerminalSession creates a new shared terminal session
func CreateTerminalSession(ctx context.Context, workspaceID int, sessionType string) (*TerminalSession, error) {
	session := &TerminalSession{
		WorkspaceID: workspaceID,
		SessionType: sessionType,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO terminal_sessions (workspace_id, session_type, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := db.DB.ExecContext(ctx, query,
		session.WorkspaceID, session.SessionType, session.IsActive,
		session.CreatedAt, session.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	sessionID, _ := result.LastInsertId()
	session.ID = int(sessionID)

	return session, nil
}

// AddParticipant adds a user to a terminal session
func AddParticipant(ctx context.Context, sessionID, userID int, username, permission string) (*TerminalParticipant, error) {
	participant := &TerminalParticipant{
		SessionID:   sessionID,
		UserID:      userID,
		Username:    username,
		Permission:  permission,
		Connected:   true,
		ConnectedAt: sql.NullTime{Time: time.Now(), Valid: true},
	}

	query := `
		INSERT INTO terminal_participants (session_id, user_id, username, permission, connected, connected_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := db.DB.ExecContext(ctx, query,
		participant.SessionID, participant.UserID, participant.Username,
		participant.Permission, participant.Connected, participant.ConnectedAt.Time)
	if err != nil {
		return nil, fmt.Errorf("failed to add participant: %w", err)
	}

	participantID, _ := result.LastInsertId()
	participant.ID = int(participantID)

	return participant, nil
}

// RemoveParticipant removes a user from a terminal session
func RemoveParticipant(ctx context.Context, sessionID, userID int) error {
	_, err := db.DB.ExecContext(ctx, `
		UPDATE terminal_participants 
		SET connected = FALSE, disconnected_at = ?
		WHERE session_id = ? AND user_id = ?
	`, time.Now(), sessionID, userID)
	return err
}

// GetSessionParticipantsByUser retrieves all active sessions for a user
func GetSessionParticipantsByUser(ctx context.Context, userID int) ([]TerminalParticipant, error) {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT tp.id, tp.session_id, tp.user_id, tp.username, tp.permission, 
		       tp.connected, tp.connected_at, tp.disconnected_at, ts.workspace_id
		FROM terminal_participants tp
		JOIN terminal_sessions ts ON tp.session_id = ts.id
		WHERE tp.user_id = ? AND tp.connected = TRUE
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user sessions: %w", err)
	}
	defer rows.Close()

	var participants []TerminalParticipant
	for rows.Next() {
		var p TerminalParticipant
		if err := rows.Scan(&p.ID, &p.SessionID, &p.UserID, &p.Username,
			&p.Permission, &p.Connected, &p.ConnectedAt, &p.DisconnectedAt); err != nil {
			return nil, fmt.Errorf("failed to scan participant: %w", err)
		}
		participants = append(participants, p)
	}

	return participants, nil
}

// HandleWebSocket handles WebSocket connections for terminal sessions
func HandleWebSocket(w http.ResponseWriter, r *http.Request, sessionID, userID int, username, permission string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	client := &Client{
		ID:         userID,
		SessionID:  sessionID,
		UserID:     userID,
		Username:   username,
		Permission: permission,
		Conn:       conn,
		Send:       make(chan []byte, 256),
		Hub:        hub,
	}

	hub.register <- client

	// Add participant to database
	AddParticipant(r.Context(), sessionID, userID, username, permission)
	defer RemoveParticipant(r.Context(), sessionID, userID)

	// Start client goroutines
	go client.writePump()
	go client.readPump()
}

func (c *Client) writePump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	for message := range c.Send {
		c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(65536) // 64KB max message size
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Broadcast message to all clients in the session
		hub.broadcast <- message
	}
}

// BroadcastInput broadcasts terminal input from an editor to all participants
func BroadcastInput(sessionID, userID int, data string) {
	msg := TerminalMessage{
		Type:      "input",
		SessionID: sessionID,
		UserID:    userID,
		Data:      data,
		Timestamp: time.Now(),
	}
	msgData, _ := json.Marshal(msg)
	hub.broadcast <- msgData
}

// BroadcastOutput broadcasts terminal output to all participants
func BroadcastOutput(sessionID int, data string) {
	msg := TerminalMessage{
		Type:      "output",
		SessionID: sessionID,
		Data:      data,
		Timestamp: time.Now(),
	}
	msgData, _ := json.Marshal(msg)
	hub.broadcast <- msgData
}

// BroadcastCursor broadcasts cursor position updates
func BroadcastCursor(sessionID, userID int, x, y int) {
	msg := TerminalMessage{
		Type:      "cursor",
		SessionID: sessionID,
		UserID:    userID,
		Data:      fmt.Sprintf("{\"x\":%d,\"y\":%d}", x, y),
		Timestamp: time.Now(),
	}
	msgData, _ := json.Marshal(msg)
	hub.broadcast <- msgData
}
