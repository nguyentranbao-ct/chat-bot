package server

import (
	"context"
	"log"
	"strings"

	socketio "github.com/googollee/go-socket.io"
	"github.com/labstack/echo/v4"

	"github.com/nguyentranbao-ct/chat-bot/internal/models"
	"github.com/nguyentranbao-ct/chat-bot/internal/usecase"
)

// Ensure SocketHandler implements SocketBroadcaster interface
var _ usecase.SocketBroadcaster = (*SocketHandler)(nil)

type SocketHandler struct {
	server      *socketio.Server
	authUsecase *usecase.AuthUseCase
}

func NewSocketHandler(authUsecase *usecase.AuthUseCase) (*SocketHandler, error) {
	server := socketio.NewServer(nil)

	handler := &SocketHandler{
		server:      server,
		authUsecase: authUsecase,
	}

	// Configure socket events
	handler.setupEvents()

	return handler, nil
}

func (h *SocketHandler) setupEvents() {
	// Handle connection
	h.server.OnConnect("/", func(s socketio.Conn) error {
		log.Printf("Socket connected: %s", s.ID())

		// Authenticate connection
		token := h.extractTokenFromAuth(s)
		if token == "" {
			log.Printf("Socket %s: No auth token provided", s.ID())
			return s.Close()
		}

		user, err := h.authUsecase.ValidateToken(context.Background(), token)
		if err != nil {
			log.Printf("Socket %s: Invalid token: %v", s.ID(), err)
			return s.Close()
		}

		// Store user info in connection
		s.SetContext(map[string]interface{}{
			"user_id": user.ID.Hex(),
			"user":    user,
		})

		log.Printf("Socket %s authenticated for user %s", s.ID(), user.Email)
		return nil
	})

	// Handle disconnection
	h.server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		log.Printf("Socket %s disconnected: %s", s.ID(), reason)
	})

	// Handle join channel
	h.server.OnEvent("/", "join_channel", func(s socketio.Conn, channelID string) {
		userID := h.getUserIDFromSocket(s)
		if userID == "" {
			return
		}

		// Join the channel room
		s.Join(channelID)
		log.Printf("User %s joined channel %s", userID, channelID)

		// Notify others in the channel
		h.server.BroadcastToRoom("/", channelID, "user_joined", map[string]interface{}{
			"user_id":    userID,
			"channel_id": channelID,
		})
	})

	// Handle leave channel
	h.server.OnEvent("/", "leave_channel", func(s socketio.Conn, channelID string) {
		userID := h.getUserIDFromSocket(s)
		if userID == "" {
			return
		}

		// Leave the channel room
		s.Leave(channelID)
		log.Printf("User %s left channel %s", userID, channelID)

		// Notify others in the channel
		h.server.BroadcastToRoom("/", channelID, "user_left", map[string]interface{}{
			"user_id":    userID,
			"channel_id": channelID,
		})
	})

	// Handle typing status
	h.server.OnEvent("/", "typing", func(s socketio.Conn, data map[string]interface{}) {
		userID := h.getUserIDFromSocket(s)
		if userID == "" {
			return
		}

		channelID, ok := data["channel_id"].(string)
		if !ok {
			return
		}

		isTyping, ok := data["is_typing"].(bool)
		if !ok {
			return
		}

		eventType := "user_typing_stop"
		if isTyping {
			eventType = "user_typing_start"
		}

		// Broadcast typing status to other users in the channel
		h.server.BroadcastToRoom("/", channelID, eventType, map[string]interface{}{
			"user_id":    userID,
			"channel_id": channelID,
			"is_typing":  isTyping,
		})
	})

	// Handle errors
	h.server.OnError("/", func(s socketio.Conn, e error) {
		log.Printf("Socket error on %s: %v", s.ID(), e)
	})
}

func (h *SocketHandler) extractTokenFromAuth(s socketio.Conn) string {
	// Try to get token from handshake auth
	if auth := s.RemoteHeader().Get("Authorization"); auth != "" {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	// Try to get from query params during handshake
	url := s.URL()
	query := url.Query()
	if token := query.Get("token"); token != "" {
		return token
	}

	return ""
}

func (h *SocketHandler) getUserIDFromSocket(s socketio.Conn) string {
	ctx := s.Context()
	if ctx == nil {
		return ""
	}

	contextMap, ok := ctx.(map[string]interface{})
	if !ok {
		return ""
	}

	userID, ok := contextMap["user_id"].(string)
	if !ok {
		return ""
	}

	return userID
}

// Broadcast message to channel
func (h *SocketHandler) BroadcastMessage(channelID string, message *models.ChatMessage) {
	h.server.BroadcastToRoom("/", channelID, "message_received", message)
}

// Broadcast message sent confirmation to sender
func (h *SocketHandler) BroadcastMessageSent(userID string, message *models.ChatMessage) {
	// Find connections for this user and emit to them
	h.server.ForEach("/", "", func(conn socketio.Conn) {
		if h.getUserIDFromSocket(conn) == userID {
			conn.Emit("message_sent", message)
		}
	})
}

// Broadcast typing indicator
func (h *SocketHandler) BroadcastTyping(channelID, userID string, isTyping bool) {
	eventType := "user_typing_stop"
	if isTyping {
		eventType = "user_typing_start"
	}

	h.server.BroadcastToRoom("/", channelID, eventType, map[string]interface{}{
		"user_id":    userID,
		"channel_id": channelID,
		"is_typing":  isTyping,
	})
}

// Get the socket.io server for integration with Echo
func (h *SocketHandler) GetServer() *socketio.Server {
	return h.server
}

// Middleware for Echo integration
func (h *SocketHandler) Handler() echo.HandlerFunc {
	return echo.WrapHandler(h.server)
}