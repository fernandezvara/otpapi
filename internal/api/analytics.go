package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"otp/internal/realtime"
)

// AnalyticsStream streams realtime events for the authenticated customer via SSE.
func AnalyticsStream(c *gin.Context) {
	customerID := c.GetString("customer_id")
	if customerID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// SSE headers
	w := c.Writer
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable proxy buffering if any
	flusher, ok := w.(http.Flusher)
	if !ok {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	ch, unsubscribe := realtime.SubscribeDefault(customerID)
	defer unsubscribe()

	ctx := c.Request.Context()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	// Send an initial event to confirm stream
	init := realtime.Event{Type: "hello", Timestamp: time.Now(), Data: map[string]any{"status": "ok"}}
	if b, err := json.Marshal(init); err == nil {
		fmt.Fprintf(w, "event: %s\n", init.Type)
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-ch:
			b, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: %s\n", ev.Type)
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		case <-heartbeat.C:
			ev := realtime.Event{Type: "heartbeat", Timestamp: time.Now(), Data: map[string]any{"t": time.Now().Unix()}}
			b, _ := json.Marshal(ev)
			fmt.Fprintf(w, "event: %s\n", ev.Type)
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		}
	}
}
