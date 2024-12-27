package routes

import (
	"context"

	"github.com/frhan23/chat-go/handler"
	"github.com/gorilla/mux"
)

func NewRoutes(mux *mux.Router) {
	// Health check
	mux.HandleFunc("/health", handler.HealthCheck)

	ws := handler.NewWebSocketHandler()
	mux.Handle("/ws", ws)

	chatServer := handler.NewChatServer(context.Background())

	mux.Handle("/chat", chatServer)
}
