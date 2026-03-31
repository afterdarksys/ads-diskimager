package api

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for WebSocket
		// In production, implement proper origin checking
		return true
	},
}

// handleJobStream handles WebSocket streaming for job progress
func (s *Server) handleJobStream(c *gin.Context) {
	jobID := c.Param("jobId")

	// Get the job
	job, err := s.jobQueue.GetJob(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: fmt.Sprintf("Job not found: %s", jobID),
		})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket: %v", err)
		return
	}
	defer conn.Close()

	// Subscribe to job progress
	progressChan := job.Subscribe()
	defer job.Unsubscribe(progressChan)

	// Send initial job status
	if err := conn.WriteJSON(map[string]interface{}{
		"type":   "status",
		"job_id": job.ID,
		"status": job.Status,
	}); err != nil {
		log.Printf("Failed to send initial status: %v", err)
		return
	}

	// Setup ping/pong to keep connection alive
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Channel to signal connection close
	done := make(chan struct{})

	// Handle incoming messages (primarily for pong responses)
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	// Stream progress updates
	for {
		select {
		case progress, ok := <-progressChan:
			if !ok {
				// Channel closed, job completed
				if err := conn.WriteJSON(map[string]interface{}{
					"type":   "complete",
					"job_id": job.ID,
					"status": job.Status,
				}); err != nil {
					log.Printf("Failed to send completion message: %v", err)
				}
				return
			}

			// Send progress update
			if err := conn.WriteJSON(map[string]interface{}{
				"type":     "progress",
				"job_id":   job.ID,
				"progress": progress,
			}); err != nil {
				log.Printf("Failed to send progress: %v", err)
				return
			}

		case <-ticker.C:
			// Send ping to keep connection alive
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Failed to send ping: %v", err)
				return
			}

		case <-done:
			// Client disconnected
			return
		}
	}
}

// StreamProgress is a helper to stream progress updates via WebSocket
func StreamProgress(conn *websocket.Conn, jobID string, progress *JobProgress) error {
	return conn.WriteJSON(map[string]interface{}{
		"type":     "progress",
		"job_id":   jobID,
		"progress": progress,
	})
}

// StreamError sends an error message via WebSocket
func StreamError(conn *websocket.Conn, jobID string, err error) error {
	return conn.WriteJSON(map[string]interface{}{
		"type":    "error",
		"job_id":  jobID,
		"error":   err.Error(),
	})
}

// StreamComplete sends a completion message via WebSocket
func StreamComplete(conn *websocket.Conn, jobID string, result *JobResult) error {
	return conn.WriteJSON(map[string]interface{}{
		"type":    "complete",
		"job_id":  jobID,
		"result":  result,
	})
}
