package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/oklog/ulid/v2"

	"github.com/SvenDH/go-card-engine/game"
)

type MemoryBroker struct {
	subscribers map[string][]*Subscriber
	mutex       sync.Mutex
}

func NewMemoryBroker() *MemoryBroker {
	return &MemoryBroker{
		subscribers: make(map[string][]*Subscriber),
	}
}

type Subscriber struct {
	Channel     chan []byte
	Unsubscribe chan bool
}

func (b *MemoryBroker) Subscribe(ctx context.Context, channels ...string) *Subscriber {
	sub := &Subscriber{
		Channel:     make(chan []byte, 100),
		Unsubscribe: make(chan bool),
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()

	for _, channel := range channels {
		b.subscribers[channel] = append(b.subscribers[channel], sub)
	}

	return sub
}

func (b *MemoryBroker) Unsubscribe(ctx context.Context, sub *Subscriber, channels ...string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	for _, channel := range channels {
		subs, exists := b.subscribers[channel]
		if !exists {
			continue
		}

		for i, s := range subs {
			if s == sub {
				// Remove the subscriber from the slice
				subs = append(subs[:i], subs[i+1:]...)
				break
			}
		}

		if len(subs) == 0 {
			delete(b.subscribers, channel)
		} else {
			b.subscribers[channel] = subs
		}
	}
}

func (b *MemoryBroker) Publish(ctx context.Context, channel string, message []byte) error {
	b.mutex.Lock()
	subs, exists := b.subscribers[channel]
	b.mutex.Unlock()

	if !exists {
		return nil
	}

	for _, sub := range subs {
		select {
		case sub.Channel <- message:
		default:
			log.Println("dropping message, channel full")
		}
	}

	return nil
}

type Server struct {
	clients    map[string]*Client
	rooms      map[string]*Room
	register   chan *Client
	unregister chan *Client
	repository *Repository
	broker     *MemoryBroker
	mutex      sync.Mutex
}

func NewWebsocketServer(broker *MemoryBroker, repository *Repository) *Server {
	return &Server{
		clients:    make(map[string]*Client),
		rooms:      make(map[string]*Room),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		repository: repository,
		broker:     broker,
	}
}

func (s *Server) Run() {
	for {
		select {
		case client := <-s.register:
			s.mutex.Lock()
			s.clients[client.Name] = client
			s.mutex.Unlock()

		case client := <-s.unregister:
			s.mutex.Lock()
			if _, ok := s.clients[client.Name]; ok {
				delete(s.clients, client.Name)
				close(client.send)
			}
			s.mutex.Unlock()
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Client struct {
	Name    string
	conn    *websocket.Conn
	server  *Server
	send    chan []byte
	receive chan any
	room    *Room
	player  *game.Player
	mu      sync.Mutex
}

func newClient(conn *websocket.Conn, server *Server, name string) *Client {
	return &Client{
		Name:    name,
		conn:    conn,
		server:  server,
		send:    make(chan []byte, 256),
		receive: make(chan any, 256),
	}
}

func (c *Client) GetName() string {
	return c.Name
}

func ServeWs(server *Server, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// Get username from query parameters or generate one
	username := r.URL.Query().Get("username")
	if username == "" {
		username = "user" + ulid.Make().String()
	}

	client := newClient(conn, server, username)

	// Register the client
	server.register <- client

	// Start the client's read and write pumps
	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.server.unregister <- c
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		c.handleNewMessage(message)
	}
}

func (c *Client) writePump() {
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// The server closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.mu.Lock()
			err := c.conn.WriteMessage(websocket.TextMessage, message)
			c.mu.Unlock()

			if err != nil {
				log.Println("write error:", err)
				return
			}
		}
	}
}

func (c *Client) handleNewMessage(jsonMessage []byte) {
	// Handle incoming WebSocket messages
	// This is a simplified version - you'll need to implement the actual message handling
}

type Room struct {
	Name       string
	server     *Server
	clients    []*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Message
	close      chan interface{}
	game       *game.Game
	running    bool
}

type Message struct {
	Type   string `json:"type"`
	Data   any    `json:"data,omitempty"`
	Target string `json:"target,omitempty"`
	Sender string `json:"sender,omitempty"`
}

func (m *Message) encode() []byte {
	data, _ := json.Marshal(m)
	return data
}
