package handler

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type WebSocketHandler struct {
	Upgrader websocket.Upgrader
}

func NewWebSocketHandler() WebSocketHandler {
	return WebSocketHandler{
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				log.Printf("Origin: %s", origin)
				return true // Replace this with origin validation logic if needed
			},
		},
	}
}

func (ws WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := ws.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("error %s when upgrading connection to websocket", err)
		return
	}
	defer c.Close()
	for {
        messageType, message, err := c.ReadMessage()
        if err != nil {
            fmt.Println(err)
            break
        }
        fmt.Printf("Received: %s\n", message)

        if err := c.WriteMessage(messageType, message); err != nil {
            fmt.Println(err)
            break
        }
    }
}
