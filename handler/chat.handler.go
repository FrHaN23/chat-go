package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Room struct {
	Name    string
	Clients map[*Client]bool
}

type Client struct {
	ID         string
	Username   string
	Connection *websocket.Conn
	Send       chan []byte
	Room       string
}

type ChatServer struct {
	Rooms      map[string]*Room
	Clients    map[*Client]bool
	Broadcast  chan Message
	Register   chan *Client
	Unregister chan *Client
	Mutex      sync.Mutex
	cancelFunc context.CancelFunc
}

type Message struct {
	SenderID  string    `json:"sender_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	System    bool      `json:"system"`
	Room      string    `json:"room"`
	TimeStamp time.Time `json:"timestamp"`
}

// ServeHTTP handles WebSocket connections
func (cs *ChatServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	vars := mux.Vars(r)
	roomName := vars["room"]

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
		log.Printf("Error upgrading connection: %v", err)
		return
	}
	defer conn.Close()

	client, err := cs.handshake(conn, roomName)
	if err != nil {
		log.Printf("Handshake failed: %v", err)
		return
	}

	cs.Register <- client
	go cs.handleMessages(client)
	cs.handleWrites(client)
}

// NewChatServer creates a new ChatServer
func NewChatServer(ctx context.Context) *ChatServer {
	ctx, cancel := context.WithCancel(ctx)
	cs := &ChatServer{
		Rooms:      make(map[string]*Room),
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan Message, 100),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		cancelFunc: cancel,
	}
	go cs.run(ctx)
	return cs
}

// Stop gracefully shuts down the ChatServer
func (cs *ChatServer) Stop() {
	cs.cancelFunc()
}

// run starts the server loop to handle clients and messages
func (cs *ChatServer) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down ChatServer...")
			return

		// New client registration
		case client := <-cs.Register:
			cs.Mutex.Lock()
			room, exists := cs.Rooms[client.Room]
			if !exists {
				room = &Room{
					Name:    client.Room,
					Clients: make(map[*Client]bool),
				}
				cs.Rooms[client.Room] = room
			}
			// Register client in the room
			room.Clients[client] = true
			cs.Clients[client] = true
			cs.Mutex.Unlock()
			log.Printf("New client connected: %s (%s) to room %s", client.Username, client.ID, client.Room)
			cs.broadcastSystemMessage(client.Room, client.Username+" has joined the room")

		// Client unregistering (disconnection)
		case client := <-cs.Unregister:
			cs.Mutex.Lock()
			if room, exists := cs.Rooms[client.Room]; exists {
				delete(room.Clients, client)
				close(client.Send)
				log.Printf("Client disconnected: %s (%s) from room %s", client.Username, client.ID, client.Room)
				cs.broadcastSystemMessage(client.Room, client.Username+" has left the room")
			}
			cs.Mutex.Unlock()

		// Message broadcasting
		case message := <-cs.Broadcast:
			cs.Mutex.Lock()
			log.Printf("Broadcasting message: %v", message)
			if room, exists := cs.Rooms[message.Room]; exists {
				log.Printf("Clients in room %s: %d", message.Room, len(room.Clients))
				for client := range room.Clients {
					log.Printf("Sending message to client: %s", client.Username)
					data, err := json.Marshal(message)
					if err != nil {
						log.Printf("Error marshaling message: %v", err)
						continue
					}
					log.Printf("Serialized message: %s", string(data))

					// Attempt to send message
					select {
					case client.Send <- data:
						log.Printf("Message sent to client: %s", client.Username)
					default:
						log.Printf("Client send buffer full, unregistering: %s", client.Username)
						close(client.Send)
						delete(room.Clients, client)
					}
				}
			}
			cs.Mutex.Unlock()
		}
	}
}

// handshake performs the initial handshake and returns a new client
func (cs *ChatServer) handshake(conn *websocket.Conn, room string) (*Client, error) {
	_, msg, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	var handshake struct {
		Username string `json:"username"`
	}
	if err := json.Unmarshal(msg, &handshake); err != nil || handshake.Username == "" {
		conn.WriteMessage(websocket.TextMessage, []byte("Invalid username"))
		return nil, err
	}

	return &Client{
		ID:         uuid.New().String(),
		Username:   handshake.Username,
		Connection: conn,
		Room:       room,
		Send:       make(chan []byte, 256),
	}, nil
}

// handleMessages handles incoming messages from a client
func (cs *ChatServer) handleMessages(client *Client) {
	defer func() {
		cs.Unregister <- client
		client.Connection.Close()
	}()
	for {
		_, msg, err := client.Connection.ReadMessage()
		if err != nil {
			log.Printf("Error reading message from %s (%s): %v", client.Username, client.ID, err)
			break
		}

		var incoming Message
		if err := json.Unmarshal(msg, &incoming); err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}

		cs.Broadcast <- Message{
			SenderID:  client.ID,
			Username:  client.Username,
			Content:   incoming.Content,
			Room:      client.Room,
			TimeStamp: time.Now(),
		}
	}
}

// handleWrites sends messages to a client
func (cs *ChatServer) handleWrites(client *Client) {
	defer client.Connection.Close()
	for msg := range client.Send {
		log.Printf("Sending to %s (%s): %s", client.Username, client.ID, string(msg))
		if err := client.Connection.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("Error writing to client %s (%s): %v", client.Username, client.ID, err)
			break
		}
	}
}

func (cs *ChatServer) broadcastSystemMessage(roomName, content string) {
	cs.Broadcast <- Message{
		Username:  "System",
		Content:   content,
		Room:      roomName,
		System:    true,
		TimeStamp: time.Now(),
	}
}
