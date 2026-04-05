package http

import (
	"fmt"

	"github.com/gin-gonic/gin"
	redisadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/redis"
)

func StatusStreamHandler(rdb *redisadapter.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prepare SSE Headers
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

		ctx := c.Request.Context()
		pubsub := rdb.Subscribe(ctx, "status_updates")
		defer pubsub.Close()

		ch := pubsub.Channel()

		// Send an initial connected ping
		c.Writer.Write([]byte("event: ping\ndata: connected\n\n"))
		c.Writer.Flush()

		for {
			select {
			case <-ctx.Done():
				return // Client disconnected
			case msg, ok := <-ch:
				if !ok {
					return
				}
				// Format as Server-Sent Event
				// "data: {JSON_PAYLOAD}\n\n"
				payload := fmt.Sprintf("data: %s\n\n", msg.Payload)
				if _, err := c.Writer.Write([]byte(payload)); err != nil {
					return // Error writing to stream, typically client disjointed
				}
				c.Writer.Flush()
			}
		}
	}
}
