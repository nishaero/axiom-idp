//go:build ignore

package catalog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// WebSocketHub manages WebSocket connections for real-time updates
type WebSocketHub struct {
	mu           sync.RWMutex
	connections  map[*websocket.Conn]bool
	broadcast    chan *Event
	register     chan *websocket.Conn
	unregister   chan *websocket.Conn
	logger       *logrus.Logger
	heartbeats   bool
	maxPingDelay time.Duration
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub(logger *logrus.Logger) *WebSocketHub {
	return &WebSocketHub{
		connections:  make(map[*websocket.Conn]bool),
		broadcast:    make(chan *Event, 256),
		register:     make(chan *websocket.Conn, 256),
		unregister:   make(chan *websocket.Conn, 256),
		logger:       logger,
		heartbeats:   true,
		maxPingDelay: 10 * time.Second,
	}
}

// Start starts the WebSocket hub
func (hub *WebSocketHub) Start() {
	go hub.run()
}

// run is the main loop for processing WebSocket messages
func (hub *WebSocketHub) run() {
	for {
		select {
		case conn := <-hub.register:
			hub.mu.Lock()
			if _, ok := hub.connections[conn]; ok {
				hub.mu.Unlock()
				conn.Close()
				continue
			}
			hub.connections[conn] = true
			hub.mu.Unlock()
			hub.logger.WithField("connection", conn).Debug("WebSocket connection registered")

		case conn := <-hub.unregister:
			hub.mu.Lock()
			if _, ok := hub.connections[conn]; ok {
				delete(hub.connections, conn)
				close(conn)
				hub.logger.WithField("connection", conn).Debug("WebSocket connection unregistered")
			}
			hub.mu.Unlock()

		case event := <-hub.broadcast:
			hub.mu.RLock()
			for conn := range hub.connections {
				select {
				case <-conn.CloseHandler():
					hub.mu.RUnlock()
					return
				default:
					hub.writeMessage(conn, event)
				}
			}
			hub.mu.RUnlock()

		case <-time.After(10 * time.Second):
			hub.mu.RLock()
			for conn := range hub.connections {
				hub.writeMessage(conn, &Event{
					Type:      EventHeartbeat,
					Timestamp: time.Now(),
					Metadata: map[string]interface{}{
						"type": "heartbeat",
					},
				})
			}
			hub.mu.RUnlock()
		}
	}
}

// writeMessage writes an event to a WebSocket connection
func (hub *WebSocketHub) writeMessage(conn *websocket.Conn, event *Event) {
	data, err := json.Marshal(event)
	if err != nil {
		hub.logger.WithError(err).Error("Failed to marshal event")
		conn.Close()
		return
	}

	err = conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		hub.logger.WithError(err).Debug("WebSocket write error")
		conn.Close()
		return
	}
}

// RegisterConnection registers a new WebSocket connection
func (hub *WebSocketHub) RegisterConnection(conn *websocket.Conn) {
	hub.register <- conn
}

// UnregisterConnection removes a WebSocket connection
func (hub *WebSocketHub) UnregisterConnection(conn *websocket.Conn) {
	hub.unregister <- conn
}

// BroadcastEvent broadcasts an event to all connected clients
func (hub *WebSocketHub) BroadcastEvent(event *Event) {
	hub.broadcast <- event
}

// BroadcastEventFilter broadcasts events matching a filter to all connected clients
func (hub *WebSocketHub) BroadcastEventFilter(event *Event, filter func(*Event) bool) {
	if filter(event) {
		hub.broadcast <- event
	}
}

// GetConnectionCount returns the number of connected clients
func (hub *WebSocketHub) GetConnectionCount() int {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	return len(hub.connections)
}

// BroadcastError broadcasts an error event to all connected clients
func (hub *WebSocketHub) BroadcastError(err error) {
	hub.broadcast <- &Event{
		Type:      EventError,
		Timestamp: time.Now(),
		Error:     err,
	}
}

// WebSocketUpgrader is the upgrader for WebSocket connections
var WebSocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins - configure appropriately for production
		return true
	},
}

// HandleWebSocket handles incoming WebSocket connections
func (hub *WebSocketHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := WebSocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "WebSocket upgrade failed", http.StatusInternalServerError)
		return
	}

	hub.logger.WithField("connection", conn).Info("New WebSocket connection")
	hub.RegisterConnection(conn)
	defer hub.UnregisterConnection(conn)

	// Handle WebSocket messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				hub.logger.WithError(err).Error("WebSocket error")
			}
			break
		}

		// Parse incoming message
		var request struct {
			Type     string                 `json:"type"`
			Metadata map[string]interface{} `json:"metadata"`
		}

		if err := json.Unmarshal(message, &request); err != nil {
			hub.logger.WithError(err).Error("Failed to parse WebSocket message")
			continue
		}

		// Handle subscription requests
		if request.Type == "subscribe" {
			hub.handleSubscription(conn, request.Metadata)
		}

		// Handle unsubscribe requests
		if request.Type == "unsubscribe" {
			hub.handleUnsubscription(conn, request.Metadata)
		}
	}
}

// handleSubscription handles subscription requests from clients
func (hub *WebSocketHub) handleSubscription(conn *websocket.Conn, metadata map[string]interface{}) {
	eventType, ok := metadata["eventType"].(string)
	if !ok {
		return
	}

	hub.logger.WithFields(logrus.Fields{
		"connection": conn,
		"eventType":  eventType,
	}).Debug("Client subscribed to event type")
}

// handleUnsubscription handles unsubscription requests from clients
func (hub *WebSocketHub) handleUnsubscription(conn *websocket.Conn, metadata map[string]interface{}) {
	eventType, ok := metadata["eventType"].(string)
	if !ok {
		return
	}

	hub.logger.WithFields(logrus.Fields{
		"connection": conn,
		"eventType":  eventType,
	}).Debug("Client unsubscribed from event type")
}

// Connection represents a WebSocket connection with metadata
type Connection struct {
	ID           string    `json:"id"`
	ConnectedAt  time.Time `json:"connected_at"`
	LastActivity time.Time `json:"last_activity"`
	EventsSent   int       `json:"events_sent"`
}

// GetConnections returns information about all active connections
func (hub *WebSocketHub) GetConnections() []*Connection {
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	connections := make([]*Connection, 0, len(hub.connections))
	for conn := range hub.connections {
		connections = append(connections, &Connection{
			ID:           conn.LocalAddr().String(),
			ConnectedAt:  time.Now(),
			LastActivity: time.Now(),
			EventsSent:   0,
		})
	}

	return connections
}

// EventMessage represents a message sent over WebSocket
type EventMessage struct {
	Type      string            `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     string            `json:"error,omitempty"`
}

// ConvertEventToMessage converts an Event to a WebSocket message
func ConvertEventToMessage(event *Event) *EventMessage {
	message := &EventMessage{
		Type:      eventTypeToString(event.Type),
		Timestamp: event.Timestamp,
		Data:      make(map[string]interface{}),
	}

	if event.Error != nil {
		message.Error = event.Error.Error()
		return message
	}

	// Add container data if present
	if event.Container != nil {
		message.Data["container"] = event.Container
	}

	// Add image data if present
	if event.Image != nil {
		message.Data["image"] = event.Image
	}

	// Add network data if present
	if event.Network != nil {
		message.Data["network"] = event.Network
	}

	// Add pod data if present
	if event.Pod != nil {
		message.Data["pod"] = event.Pod
	}

	// Add deployment data if present
	if event.Deployment != nil {
		message.Data["deployment"] = event.Deployment
	}

	// Add service data if present
	if event.Service != nil {
		message.Data["service"] = event.Service
	}

	// Add additional metadata
	for k, v := range event.Metadata {
		if k != "type" && k != "action" {
			message.Data[k] = v
		}
	}

	return message
}

// eventTypeToString converts an EventType to a string
func eventTypeToString(eventType EventType) string {
	switch eventType {
	case EventContainerStarted:
		return "container.started"
	case EventContainerStopped:
		return "container.stopped"
	case EventContainerHealthChanged:
		return "container.health.changed"
	case EventContainerStatusChanged:
		return "container.status.changed"
	case EventContainerLogs:
		return "container.logs"
	case EventImagePullled:
		return "image.pulled"
	case EventImageDeleted:
		return "image.deleted"
	case EventNetworkCreated:
		return "network.created"
	case EventNetworkDeleted:
		return "network.deleted"
	case EventDiscoveryCompleted:
		return "discovery.completed"
	case EventDiscoveryFailed:
		return "discovery.failed"
	case EventPodAdded:
		return "pod.added"
	case EventPodUpdated:
		return "pod.updated"
	case EventPodDeleted:
		return "pod.deleted"
	case EventDeploymentAdded:
		return "deployment.added"
	case EventDeploymentUpdated:
		return "deployment.updated"
	case EventDeploymentDeleted:
		return "deployment.deleted"
	case EventServiceAdded:
		return "service.added"
	case EventServiceUpdated:
		return "service.updated"
	case EventServiceDeleted:
		return "service.deleted"
	case EventError:
		return "error"
	case EventHeartbeat:
		return "heartbeat"
	default:
		return "unknown"
	}
}

// SubscribeRequest represents a request to subscribe to event types
type SubscribeRequest struct {
	EventTypes []EventType `json:"event_types"`
}

// SubscribeResponse represents a response to a subscription request
type SubscribeResponse struct {
	Success bool     `json:"success"`
	Types   []string `json:"types,omitempty"`
	Error   string   `json:"error,omitempty"`
}

// UnsubscribeRequest represents a request to unsubscribe from event types
type UnsubscribeRequest struct {
	EventTypes []EventType `json:"event_types"`
}

// SubscribeRequest represents a request to subscribe to event types
type SubscriptionRequest struct {
	ID       string          `json:"id"`
	Types    []string        `json:"types"`
	Filter   map[string]interface{} `json:"filter,omitempty"`
}

// SubscriptionResponse represents a response to a subscription request
type SubscriptionResponse struct {
	Success bool   `json:"success"`
	ID      string `json:"id,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ParseSubscriptionRequest parses a subscription request from JSON
func ParseSubscriptionRequest(data []byte) (*SubscriptionRequest, error) {
	var req SubscriptionRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to parse subscription request: %w", err)
	}

	if len(req.Types) == 0 {
		return nil, fmt.Errorf("no event types specified")
	}

	return &req, nil
}

// BuildSubscriptionResponse builds a subscription response
func BuildSubscriptionResponse(success bool, id string, err string) *SubscriptionResponse {
	return &SubscriptionResponse{
		Success: success,
		ID:      id,
		Error:   err,
	}
}
