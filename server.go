package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
	"strings"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512

	SendMessageAction     = "send-message"
	JoinRoomAction        = "join-room"
	LeaveRoomAction       = "leave-room"
	JoinRoomPrivateAction = "join-room-private"
	RoomJoinedAction      = "room-joined"

	welcomeMessage = "%s joined the room"
)

var (
	newline = []byte{'\n'}
)

type Message struct {
	Type    string `json:"type"`
	Data    string `json:"data"`
	Target  string `json:"target"`
	Sender  string `json:"sender"`
}

func (message *Message) encode() []byte {
	data, _ := json.Marshal(message)
	return data
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Subscriber struct {
	Channel     chan []byte
	Unsubscribe chan bool
}

type Broker interface {
	Subscribe(ctx context.Context, channels ...string) *Subscriber
	Unsubscribe(ctx context.Context, sub *Subscriber, channels ...string)
	Publish(ctx context.Context, topic string, message []byte) error
	Close()
}

type MemoryBroker struct {
	subscribers map[string][]*Subscriber
	mutex       sync.Mutex
}

func NewMemoryBroker() Broker {
	return &MemoryBroker{subscribers: make(map[string][]*Subscriber)}
}

func (b *MemoryBroker) Subscribe(ctx context.Context, channels ...string) *Subscriber {
	sub := &Subscriber{
		Channel:     make(chan []byte, 1),
		Unsubscribe: make(chan bool),
	}
	b.mutex.Lock()
	defer b.mutex.Unlock()
	for _, t := range channels {
		b.subscribers[t] = append(b.subscribers[t], sub)
	}
	return sub
}

func (b *MemoryBroker) Unsubscribe(ctx context.Context, sub *Subscriber, channels ...string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	close(sub.Channel)
	for _, t := range channels {
		if subscribers, found := b.subscribers[t]; found {
			var newSubscribers []*Subscriber
			for _, subscriber := range subscribers {
				if subscriber != sub {
					newSubscribers = append(newSubscribers, subscriber)
				}
			}
			b.subscribers[t] = newSubscribers
		}
	}
}

func (b *MemoryBroker) Publish(ctx context.Context, channel string, msg []byte) error {
	b.mutex.Lock()
	if subscribers, found := b.subscribers[channel]; found {
		for _, sub := range subscribers {
			select {
			case sub.Channel <- msg:
			case <-time.After(time.Second):
				fmt.Printf("Subscriber slow. Unsubscribing from channel: %s\n", channel)
				defer b.Unsubscribe(ctx, sub, channel)
			}
		}
	}
	defer b.mutex.Unlock()
	return nil
}

func (b *MemoryBroker) Close() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	for _, subscribers := range b.subscribers {
		for _, subscriber := range subscribers {
			close(subscriber.Channel)
		}
	}
}

type Room struct {
	Name       string `json:"name"`
	server     *Server
	clients    []*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Message
	close      chan interface{}
	game 	   *Game
}

func NewRoom(name string, server *Server) *Room {
	return &Room{
		Name:       name,
		server:     server,
		clients:    make([]*Client, 0),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message),
		close:      make(chan interface{}),
		game: 		NewGame(),
	}
}

func (room *Room) Run() {
	
	//p1 := NewPlayer(cli, cards...)
	//game := NewGame(p1)
	//go game.Run()

	ch := room.server.broker.Subscribe(context.TODO(), room.Name)
	go room.subscribeToRoomMessages(ch)
	for {
		select {
		case client := <-room.register:
			room.registerClientInRoom(client)
		case client := <-room.unregister:
			room.unregisterClientInRoom(client)
		case message := <-room.broadcast:
			room.publishRoomMessage(message.encode())
		case <-room.close:
			room.server.broker.Unsubscribe(context.TODO(), ch, room.Name)
			return
		}
	}
}

func (room *Room) Close() {
	room.close <- struct{}{}
}

func (room *Room) registerClientInRoom(client *Client) {
	room.notifyClientJoined(client)
	room.clients = append(room.clients, client)

	client.player = NewPlayer(client)
	room.game.AddPlayer(client.player)
}

func (room *Room) unregisterClientInRoom(client *Client) {
	for i, c := range room.clients {
		if c == client {
			room.clients = append(room.clients[:i], room.clients[i+1:]...)
			break
		}
	}
}

func (room *Room) broadcastToClientsInRoom(message []byte) {
	for _, client := range room.clients {
		client.send <- message
	}
}

func (room *Room) publishRoomMessage(message []byte) {
	if err := room.server.broker.Publish(context.TODO(), room.Name, message); err != nil {
		log.Println(err)
	}
}

func (room *Room) subscribeToRoomMessages(ch *Subscriber) {
	for msg := range ch.Channel {
		room.broadcastToClientsInRoom(msg)
	}
}

func (room *Room) notifyClientJoined(client *Client) {
	message := Message{
		Type:  SendMessageAction,
		Target:  room.Name,
		Data: fmt.Sprintf(welcomeMessage, client.Name),
	}
	room.publishRoomMessage(message.encode())
}

type Client struct {
	Name   string `json:"name"`
	conn   *websocket.Conn
	server *Server
	send   chan []byte
	receive chan int
	room   *Room
	player *Player
}

func newClient(conn *websocket.Conn, server *Server, name string) *Client {
	return &Client{
		Name:   name,
		conn:   conn,
		server: server,
		send:   make(chan []byte, 256),
		receive: make(chan int, 1),
	}
}

func (c *Client) prompt(action string, choices []string) (int, error) {
	m := Message{Type: action, Data: strings.Join(choices, ","), Target: c.room.Name}
	c.send <- m.encode()
	choice := <-c.receive
	// TODO: handle timeout or other errors
	return choice, nil
}

func (c *Client) Card(player *Player, options []*CardInstance) (int, error) {
	if len(options) == 0 { return -1, nil }
	choices := []string{}
	for _, c := range options {
		choices = append(choices, c.Id.String())
	}
	return c.prompt("game.card", choices)
}

func (c *Client) Field(player *Player, options []int) (int, error) {
	if len(options) == 0 { return -1, nil }
	choices := []string{}
	for _, c := range options {
		choices = append(choices, fmt.Sprintf("%d", c))
	}
	return c.prompt("game.field", choices)
}

func (c *Client) Ability(player *Player, options []*Activated, card *CardInstance) (int, error) {
	if len(options) == 0 { return -1, nil }
	choices := []string{}
	for i, a := range card.GetActivatedAbilities() {
		for _, c := range options {
			if a == c {
				choices = append(choices, fmt.Sprintf("%s:%d", card.Id.String(), i))
				break
			}
		}
	}
	return c.prompt("game.ability", choices)
}

func (c *Client) Target(player *Player, options []*CardInstance, num int) ([]int, error) {
	if len(options) == 0 { return nil, nil }
	opts := []string{}
	for _, c := range options {
		opts = append(opts, c.Id.String())
	}
	choices := []int{}
	for i := 0; i < num; i++ {
		c, err := c.prompt("game.target", opts)
		if err != nil {
			return nil, err
		}
		choices = append(choices, c)
		opts = append(opts[:c], opts[c+1:]...)
	}
	return choices, nil
}

func (c *Client) Discard(player *Player, options []*CardInstance, num int) ([]int, error) {
	if len(options) == 0 || num == 0 { return nil, nil }
	opts := []string{}
	for _, c := range options {
		opts = append(opts, c.Id.String())
	}
	choices := []int{}
	for i := 0; i < num; i++ {
		c, err := c.prompt("game.discard", opts)
		if err != nil {
			return nil, err
		}
		choices = append(choices, c)
		opts = append(opts[:c], opts[c+1:]...)
	}
	return choices, nil
}

func (client *Client) readPump() {
	defer func() {
		client.disconnect()
	}()
	client.conn.SetReadLimit(maxMessageSize)
	client.conn.SetReadDeadline(time.Now().Add(pongWait))
	client.conn.SetPongHandler(func(string) error { client.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, jsonMessage, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("unexpected close error: %v", err)
			}
			break
		}
		client.handleNewMessage(jsonMessage)
	}
}

func (client *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()
	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := client.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Attach queued messages to the current websocket message.
			n := len(client.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-client.send)
			}
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (client *Client) disconnect() {
	client.server.unregister <- client
	if client.room != nil {
		client.room.unregister <- client
	}
	close(client.send)
	client.conn.Close()
}

func ServeWs(wsServer *Server, w http.ResponseWriter, r *http.Request) {
	userCtxValue := r.Context().Value(UserContextKey)
	if userCtxValue == nil {
		log.Println("Not authenticated")
		return
	}
	user := userCtxValue.(string)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := newClient(conn, wsServer, user)

	go client.writePump()
	go client.readPump()

	wsServer.register <- client
}

func (client *Client) handleNewMessage(jsonMessage []byte) {
	var message Message
	if err := json.Unmarshal(jsonMessage, &message); err != nil {
		log.Printf("Error on unmarshal JSON message %s", err)
		return
	}
	message.Sender = client.Name
	switch message.Type {
	case SendMessageAction:
		if room := client.server.rooms[message.Target]; room != nil {
			room.broadcast <- &message
		}
	case JoinRoomAction:
		client.handleJoinRoomMessage(message)
	case LeaveRoomAction:
		client.handleLeaveRoomMessage(message)
	case JoinRoomPrivateAction:
		client.handleJoinRoomPrivateMessage(message)
	}
}

func (client *Client) handleJoinRoomMessage(message Message) {
	client.joinRoom(message.Data, "")
}

func (client *Client) handleLeaveRoomMessage(message Message) {
	if room := client.server.rooms[message.Data]; room != nil {
		client.room = nil
		room.unregister <- client
	}
}

func (client *Client) handleJoinRoomPrivateMessage(message Message) {
	if target := client.server.repository.FindUserByName(message.Data); target != nil {
		roomName := message.Data + client.Name
		if joinedRoom := client.joinRoom(roomName, target.Name); joinedRoom != nil {
			client.inviteTargetUser(target, joinedRoom)
		}
	}
}

func (client *Client) joinRoom(name, sender string) (room *Room) {
	room, ok := client.server.rooms[name]
	if !ok {
		room = NewRoom(name, client.server)
		go room.Run()
		client.server.rooms[name] = room
	}
	if client.room != room {
		if client.room != nil {
			client.room.unregister <- client
		}
		client.room = room
		room.register <- client
		m := Message{Type: RoomJoinedAction, Target: room.Name, Sender: sender}
		client.send <- m.encode()
	}
	return
}

func (client *Client) inviteTargetUser(target *User, room *Room) {
	if c := client.server.clients[target.Name]; c != nil {
		c.joinRoom(room.Name, client.Name)
	}
}

type Server struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	rooms      map[string]*Room
	repository *Repository
	broker     Broker
}

func NewWebsocketServer(broker Broker, repository *Repository) *Server {
	return &Server{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		rooms:      make(map[string]*Room),
		repository: repository,
		broker:     broker,
	}
}

func (server *Server) Run() {
	for {
		select {
		case client := <-server.register:
			server.registerClient(client)
		case client := <-server.unregister:
			server.unregisterClient(client)
		}
	}
}

func (server *Server) registerClient(client *Client) {
	server.clients[client.Name] = client
}

func (server *Server) unregisterClient(client *Client) {
	delete(server.clients, client.Name)
}
