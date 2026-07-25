package ws

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"heavy-vehicle-routing/backend/internal/queue"
)

// ChatGateway pushes live chat messages to a connected client. Sending a message
// still goes through POST /api/v1/chats/{driverId}/messages (persisted via
// store.ChatMessageStore first); this only forwards the resulting event to
// whichever side is currently connected - REST/ListThread remains the durable
// source of truth for anything sent while offline.
type ChatGateway struct {
	Queue *queue.Client
}

func NewChat(q *queue.Client) *ChatGateway {
	return &ChatGateway{Queue: q}
}

func (g *ChatGateway) HandleChatStream(w http.ResponseWriter, r *http.Request, driverID, counterpartID int64) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws: chat upgrade failed for %d<->%d: %v", driverID, counterpartID, err)
		return
	}
	defer conn.Close()

	routingKey := queue.ChatRoutingKey(driverID, counterpartID)
	deliveries, closeConsumer, err := g.Queue.ConsumeChatEphemeral(routingKey)
	if err != nil {
		log.Printf("ws: chat consume failed for %s: %v", routingKey, err)
		return
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for d := range deliveries {
			if err := conn.WriteMessage(websocket.TextMessage, d.Body); err != nil {
				return
			}
		}
	}()

	// The client never sends anything meaningful over this socket (new messages
	// go through the REST POST above) - we just block on reads to detect when it
	// disconnects, then tear down the RabbitMQ consumer to stop the writer goroutine.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
	closeConsumer()
	<-done
}
