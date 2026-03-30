package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/afterdarksys/diskimager/pkg/progress"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocketServer manages WebSocket connections
type WebSocketServer struct {
	clients   map[string]map[*websocket.Conn]bool // jobID -> connections
	clientsMu sync.RWMutex
	upgrader  websocket.Upgrader
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer() *WebSocketServer {
	return &WebSocketServer{
		clients: make(map[string]map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
	}
}

// HandleConnection handles a new WebSocket connection
func (ws *WebSocketServer) HandleConnection(c *gin.Context) {
	jobID := c.Param("id")

	conn, err := ws.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Register client
	ws.clientsMu.Lock()
	if ws.clients[jobID] == nil {
		ws.clients[jobID] = make(map[*websocket.Conn]bool)
	}
	ws.clients[jobID][conn] = true
	ws.clientsMu.Unlock()

	// Send initial connection message
	conn.WriteJSON(map[string]interface{}{
		"type":    "connected",
		"job_id":  jobID,
		"message": "WebSocket connection established",
	})

	// Handle incoming messages (for control commands)
	go ws.handleMessages(jobID, conn)
}

// handleMessages handles incoming WebSocket messages
func (ws *WebSocketServer) handleMessages(jobID string, conn *websocket.Conn) {
	defer func() {
		ws.removeClient(jobID, conn)
		conn.Close()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle control messages
		var cmd map[string]interface{}
		if err := json.Unmarshal(message, &cmd); err != nil {
			log.Printf("Invalid message format: %v", err)
			continue
		}

		// Process commands (pause, resume, cancel, etc.)
		if cmdType, ok := cmd["type"].(string); ok {
			switch cmdType {
			case "ping":
				conn.WriteJSON(map[string]interface{}{
					"type": "pong",
				})
			case "cancel":
				// Would trigger job cancellation
				log.Printf("Cancel requested for job %s", jobID)
			}
		}
	}
}

// BroadcastProgress broadcasts progress updates to all clients watching a job
func (ws *WebSocketServer) BroadcastProgress(jobID string, prog progress.Progress) {
	ws.clientsMu.RLock()
	clients := ws.clients[jobID]
	ws.clientsMu.RUnlock()

	if len(clients) == 0 {
		return
	}

	message := map[string]interface{}{
		"type":            "progress",
		"job_id":          jobID,
		"bytes_processed": prog.BytesProcessed,
		"total_bytes":     prog.TotalBytes,
		"phase":           prog.Phase,
		"message":         prog.Message,
		"speed":           prog.Speed,
		"eta":             prog.ETA.Seconds(),
		"percentage":      prog.Percentage,
		"bad_sectors":     prog.BadSectors,
		"errors":          prog.Errors,
		"timestamp":       prog.Timestamp,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal progress: %v", err)
		return
	}

	ws.clientsMu.RLock()
	defer ws.clientsMu.RUnlock()

	for conn := range clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("Failed to send progress: %v", err)
			ws.removeClient(jobID, conn)
		}
	}
}

// removeClient removes a client connection
func (ws *WebSocketServer) removeClient(jobID string, conn *websocket.Conn) {
	ws.clientsMu.Lock()
	defer ws.clientsMu.Unlock()

	if clients, ok := ws.clients[jobID]; ok {
		delete(clients, conn)
		if len(clients) == 0 {
			delete(ws.clients, jobID)
		}
	}
}

// BroadcastJobStatus broadcasts job status changes
func (ws *WebSocketServer) BroadcastJobStatus(jobID string, status string, phase string) {
	ws.clientsMu.RLock()
	clients := ws.clients[jobID]
	ws.clientsMu.RUnlock()

	if len(clients) == 0 {
		return
	}

	message := map[string]interface{}{
		"type":   "status",
		"job_id": jobID,
		"status": status,
		"phase":  phase,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal status: %v", err)
		return
	}

	ws.clientsMu.RLock()
	defer ws.clientsMu.RUnlock()

	for conn := range clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("Failed to send status: %v", err)
			ws.removeClient(jobID, conn)
		}
	}
}

// BroadcastError broadcasts error messages
func (ws *WebSocketServer) BroadcastError(jobID string, err error) {
	ws.clientsMu.RLock()
	clients := ws.clients[jobID]
	ws.clientsMu.RUnlock()

	if len(clients) == 0 {
		return
	}

	message := map[string]interface{}{
		"type":   "error",
		"job_id": jobID,
		"error":  err.Error(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal error: %v", err)
		return
	}

	ws.clientsMu.RLock()
	defer ws.clientsMu.RUnlock()

	for conn := range clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("Failed to send error: %v", err)
			ws.removeClient(jobID, conn)
		}
	}
}
