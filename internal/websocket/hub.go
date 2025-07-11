package websocket

import (
	"encoding/json"
	"log"
	"net/http"

	"synctunes/internal/room"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

type Hub struct {
	rooms      map[string]*RoomHub
	roomManager *room.Manager
	register   chan *Client
	unregister chan *Client
}

type RoomHub struct {
	roomID     string
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

type Client struct {
	hub    *RoomHub
	conn   *websocket.Conn
	send   chan []byte
	userID string
	roomID string
	role   string // "host" or "listener"
}

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type RoomUpdateMessage struct {
	Type string      `json:"type"`
	Room interface{} `json:"room"`
}

type UserJoinMessage struct {
	Type     string `json:"type"`
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	Role     string `json:"role"`
}

func NewHub(roomManager *room.Manager) *Hub {
	return &Hub{
		rooms:       make(map[string]*RoomHub),
		roomManager: roomManager,
		register:    make(chan *Client),
		unregister:  make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.handleRegister(client)
		case client := <-h.unregister:
			h.handleUnregister(client)
		}
	}
}

func (h *Hub) handleRegister(client *Client) {
	roomHub, exists := h.rooms[client.roomID]
	if !exists {
		roomHub = &RoomHub{
			roomID:     client.roomID,
			clients:    make(map[*Client]bool),
			broadcast:  make(chan []byte, 256),
			register:   make(chan *Client),
			unregister: make(chan *Client),
		}
		h.rooms[client.roomID] = roomHub
		go roomHub.run()
	}
	
	roomHub.register <- client
}

func (h *Hub) handleUnregister(client *Client) {
	if roomHub, exists := h.rooms[client.roomID]; exists {
		roomHub.unregister <- client
	}
}

func (h *Hub) BroadcastToRoom(roomID string, message []byte) {
	if roomHub, exists := h.rooms[roomID]; exists {
		select {
		case roomHub.broadcast <- message:
		default:
			// Room hub is full, skip message
		}
	}
}

func (h *Hub) BroadcastToHosts(roomID string, message []byte) {
	if roomHub, exists := h.rooms[roomID]; exists {
		for client := range roomHub.clients {
			if client.role == "host" {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(roomHub.clients, client)
				}
			}
		}
	}
}

func (h *Hub) BroadcastToListeners(roomID string, message []byte) {
	if roomHub, exists := h.rooms[roomID]; exists {
		for client := range roomHub.clients {
			if client.role == "listener" {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(roomHub.clients, client)
				}
			}
		}
	}
}

func (rh *RoomHub) run() {
	for {
		select {
		case client := <-rh.register:
			rh.clients[client] = true
			log.Printf("Client connected to room %s as %s", rh.roomID, client.role)
			
		case client := <-rh.unregister:
			if _, ok := rh.clients[client]; ok {
				delete(rh.clients, client)
				close(client.send)
				log.Printf("Client (%s) disconnected from room %s", client.role, rh.roomID)
			}
			
		case message := <-rh.broadcast:
			for client := range rh.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(rh.clients, client)
				}
			}
		}
	}
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request, roomID, userID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	
	// Determine user role
	room, exists := h.roomManager.GetRoom(roomID)
	var role string = "listener"
	if exists && room.IsHost(userID) {
		role = "host"
	}
	
	client := &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
		roomID: roomID,
		role:   role,
	}
	
	h.register <- client
	
	go client.writePump()
	go client.readPump(h)
}

func (c *Client) readPump(hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()
	
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
		
		// Handle incoming messages (for future features like chat)
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}
		
		// Process message based on type
		switch msg.Type {
		case "ping":
			c.send <- []byte(`{"type":"pong"}`)
		}
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()
	
	for message := range c.send {
		c.conn.WriteMessage(websocket.TextMessage, message)
	}
}
