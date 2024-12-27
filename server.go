package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/oklog/ulid/v2"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512

	SendMessageAction     = "room.message"
	JoinRoomAction        = "room.join"
	LeaveRoomAction       = "room.leave"
	JoinRoomPrivateAction = "room.join-private"
	RoomJoinedAction      = "room.joined"
	SinglePlayerAction    = "room.join-npc"

	GameReadyAction   = "game.ready"
	GameEventAction   = "game.event"
	GameInfoAction    = "game.info"
	GamePromptAction  = "game.prompt"
	GameChoiceAction  = "game.choice"
	
	GameCardAction    = "card"
	GameFieldAction   = "field"
	GameAbilityAction = "ability"
	GameTargetAction  = "target"
	GameDiscardAction = "discard"

	welcomeMessage = "%s joined the room"
)

//go:embed cards.txt
var cardText []byte
var cards = []*Card{}

var parser = NewCardParser()

type CardPlayer interface {
	GetName() string
}

type Bot struct {
	Name string
}

func (c *Bot) GetName() string { return c.Name }

func (c *Bot) Card(player *Player, options []*CardInstance) (int, error) {
	if len(options) == 0 { return SkipCode, nil }
	return 0, nil
}

func (c *Bot) Field(player *Player, options []int) (int, error) {
	if len(options) == 0 { return SkipCode, nil }
	return 0, nil
}

func (c *Bot) Ability(player *Player, options []*Activated, card *CardInstance) (int, error) {
	if len(options) == 0 { return SkipCode, nil }
	return 0, nil
}

func (c *Bot) Target(player *Player, options []*CardInstance, num int) ([]int, error) {
	if len(options) == 0 { return []int{SkipCode}, nil }
	opts := []int{}
	for i := range options { opts = append(opts, i) }
	return opts[:num], nil
}

func (c *Bot) Discard(player *Player, options []*CardInstance, num int) ([]int, error) {
	if len(options) == 0 || num == 0 { return []int{SkipCode}, nil }
	opts := []int{}
	for i := range options { opts = append(opts, i) }
	return opts[:num], nil
}

func init() {
	for _, txt := range strings.Split(string(cardText), "\n\n") {
		card, err := parser.Parse(txt)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println(card)
		cards = append(cards, card)
	}
}

type Message struct {
	Type   string `json:"type"`
	Data   any    `json:"data,omitempty"`
	Target string `json:"target,omitempty"`
	Sender string `json:"sender,omitempty"`
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
	Name       string
	server     *Server
	clients    []*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Message
	close      chan interface{}
	mutex      sync.Mutex
	game       *Game
	running    bool
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
		game:       NewGame(),
	}
}

func (room *Room) Run() {
	room.game.On(AllEvents, room.eventHandler)
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

type GameInfo struct {
	Players map[string]string `json:"players,omitempty"`
	Cards []string          `json:"cards,omitempty"`
	Seen  map[string]string `json:"seen,omitempty"`
}

type GameEvent struct {
	Event      string `json:"event"`
	Subject    string `json:"subject,omitempty"`
	Source     string `json:"source,omitempty"`
	Controller string `json:"controller,omitempty"`
	Args 	   []any  `json:"args,omitempty"`
}

type GameRequest struct {
	Action  string   `json:"action"`
	Options []string `json:"options,omitempty"`
	Num     int      `json:"num,omitempty"`
	Card    string   `json:"card,omitempty"`
}

func (room *Room) eventHandler(event *Event) {
	card, isCard := event.Subject.(*CardInstance)
	if event.Event == EventOnDraw || event.Event == EventOnEnterBoard {
		for _, client := range room.clients {
			if client.player == card.Owner || event.Event == EventOnEnterBoard {
				client.sendInfo(&GameInfo{
					Seen:  map[string]string{card.GetId().String(): card.Card.Name},
					Cards: []string{card.Card.Text},
				})
			}
		}
	}

	e := GameEvent{Event: event.Event.String(), Args: event.Args}
	if event.Subject != nil {
		e.Subject = event.Subject.GetId().String()
	}
	if event.Source != nil {
		e.Source = event.Source.Source.Id.String()
		e.Controller = event.Source.Controller.GetId().String()
	} else if isCard {
		e.Controller = card.Controller.GetId().String()
	} else {
		e.Controller = event.Subject.GetId().String()
	}
	m := Message{Type: GameEventAction, Data: &e}
	room.broadcastToClientsInRoom(m.encode())
}

func (room *Room) Close() {
	room.close <- struct{}{}
}

func (room *Room) registerClientInRoom(client *Client) {
	room.notifyClientJoined(client)
	room.mutex.Lock()
	defer room.mutex.Unlock()
	room.clients = append(room.clients, client)
}

func (room *Room) unregisterClientInRoom(client *Client) {
	room.mutex.Lock()
	defer room.mutex.Unlock()
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
		Type:   SendMessageAction,
		Target: room.Name,
		Data:   fmt.Sprintf(welcomeMessage, client.Name),
	}
	room.publishRoomMessage(message.encode())
}

type Client struct {
	Name    string
	conn    *websocket.Conn
	server  *Server
	send    chan []byte
	receive chan any
	room    *Room
	player  *Player
	mu      sync.Mutex
}

func newClient(conn *websocket.Conn, server *Server, name string) *Client {
	c := &Client{
		Name:    name,
		conn:    conn,
		server:  server,
		send:    make(chan []byte, 256),
		receive: make(chan any, 1),
	}
	c.player = NewPlayer(c, cards...)
	return c
}

func (client *Client) GetName() string { return client.Name }

func (client *Client) sendInfo(info *GameInfo) {
	m := Message{Type: GameInfoAction, Data: info}
	client.send <- m.encode()
}

func (c *Client) prompt(action string, choices []string, num int, card string) ([]int, error) {
	m := Message{
		Type:   GamePromptAction,
		Data:   &GameRequest{action, choices, num, card},
	}
	c.send <- m.encode()
	choice := <-c.receive
	if choice == nil {
		return []int{SkipCode}, nil
	}
	selectedIds, ok := choice.([]any)
	if !ok {
		return nil, fmt.Errorf("invalid choice type")
	}
	// TODO: handle timeout or other errors
	selected := []int{}
	for _, c := range selectedIds {
		for i, o := range choices {
			if c == o {
				selected = append(selected, i)
				break
			}
		}
	}
	return selected, nil
}

func (c *Client) Card(player *Player, options []*CardInstance) (int, error) {
	if len(options) == 0 {
		return SkipCode, nil
	}
	choices := []string{}
	for _, c := range options {
		choices = append(choices, c.Id.String())
	}
	result, err := c.prompt(GameCardAction, choices, 0, "")
	return result[0], err
}

func (c *Client) Field(player *Player, options []int) (int, error) {
	if len(options) == 0 {
		return SkipCode, nil
	}
	choices := []string{}
	for _, c := range options {
		choices = append(choices, fmt.Sprintf("%d", c))
	}
	result, err := c.prompt(GameFieldAction, choices, 0, "")
	return result[0], err
}

func (c *Client) Ability(player *Player, options []*Activated, card *CardInstance) (int, error) {
	if len(options) == 0 {
		return SkipCode, nil
	}
	choices := []string{}
	for i, a := range card.GetActivatedAbilities() {
		for _, c := range options {
			if a == c {
				choices = append(choices, fmt.Sprintf("%d", i))
				break
			}
		}
	}
	result, err := c.prompt(GameAbilityAction, choices, 0, card.Id.String())
	return result[0], err
}

func (c *Client) Target(player *Player, options []*CardInstance, num int) ([]int, error) {
	if len(options) == 0 {
		return []int{SkipCode}, nil
	}
	opts := []string{}
	for _, c := range options {
		opts = append(opts, c.Id.String())
	}
	return c.prompt(GameTargetAction, opts, num, "")
}

func (c *Client) Discard(player *Player, options []*CardInstance, num int) ([]int, error) {
	if len(options) == 0 || num == 0 {
		return []int{SkipCode}, nil
	}
	opts := []string{}
	for _, c := range options {
		opts = append(opts, c.Id.String())
	}
	return c.prompt(GameDiscardAction, opts, num, "")
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
				w.Write([]byte{'\n'})
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
	client.mu.Lock()
	defer client.mu.Unlock()
	switch message.Type {
	case SendMessageAction:
		client.server.mutex.Lock()
		room := client.server.rooms[message.Target]
		client.server.mutex.Unlock()
		if room != nil {
			room.broadcast <- &message
		}
	case JoinRoomAction:
		client.handleJoinRoomMessage(&message)
	case LeaveRoomAction:
		client.handleLeaveRoomMessage(&message)
	case JoinRoomPrivateAction:
		client.handleJoinRoomPrivateMessage(&message)
	case SinglePlayerAction:
		client.handleAddNpc(&message)
	case GameReadyAction:
		client.startGame(&message)
	case GameChoiceAction:
		client.receive <- message.Data
	}
}

func (client *Client) handleJoinRoomMessage(message *Message) {
	client.joinRoom(message.Data.(string), "")
}

func (client *Client) handleLeaveRoomMessage(message *Message) {
	if room := client.server.rooms[message.Data.(string)]; room != nil {
		client.room = nil
		room.unregister <- client
	}
}

func (client *Client) handleJoinRoomPrivateMessage(message *Message) {
	if target := client.server.repository.FindUserByName(message.Data.(string)); target != nil {
		roomName := ulid.Make().String()
		if joinedRoom := client.joinRoom(roomName, target.Name); joinedRoom != nil {
			client.inviteTargetUser(target, joinedRoom)
		}
	}
}

func (client *Client) joinRoom(name, sender string) *Room {
	client.server.mutex.Lock()
	room, ok := client.server.rooms[name]
	if !ok {
		room = NewRoom(name, client.server)

		go room.Run()

		client.server.rooms[name] = room
	}
	client.server.mutex.Unlock()

	if client.room != room {
		if client.room != nil {
			client.room.unregister <- client
		}
		client.room = room
		room.register <- client
		m := Message{Type: RoomJoinedAction, Target: room.Name, Sender: sender}
		client.send <- m.encode()
	}
	return room
}

func (client *Client) handleAddNpc(message *Message) {
	roomName := ulid.Make().String()
	room := client.joinRoom(roomName, message.Sender)

	room.mutex.Lock()
	defer room.mutex.Unlock()

	room.game.AddPlayer(NewPlayer(&Bot{message.Data.(string)}, cards...))
}

func (client *Client) startGame(message *Message) {
	room := client.room
	room.mutex.Lock()
	defer room.mutex.Unlock()

	room.game.AddPlayer(client.player)

	if !room.running && len(room.game.players) >= 2 {
		log.Printf("Starting game in room %s", room.Name)
		room.running = true

		playerIds := map[string]string{}
		for _, p := range room.game.players {
			playerIds[p.GetId().String()] = p.cmdi.(CardPlayer).GetName()
		}
		for _, c := range room.clients {
			c.sendInfo(&GameInfo{Players: playerIds})
		}

		go room.game.Run()
	}
}

func (client *Client) inviteTargetUser(target *User, room *Room) {
	client.server.mutex.Lock()
	c := client.server.clients[target.Name]
	client.server.mutex.Unlock()
	if c != nil {
		c.joinRoom(room.Name, client.Name)
	}
}

type Server struct {
	clients    map[string]*Client
	rooms      map[string]*Room
	register   chan *Client
	unregister chan *Client
	repository *Repository
	broker     Broker
	mutex      sync.Mutex
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
	server.mutex.Lock()
	defer server.mutex.Unlock()
	server.clients[client.Name] = client
}

func (server *Server) unregisterClient(client *Client) {
	server.mutex.Lock()
	defer server.mutex.Unlock()
	delete(server.clients, client.Name)
}
