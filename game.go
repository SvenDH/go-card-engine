package main

import (
	"fmt"
	"iter"
	"math/rand"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/oklog/ulid/v2"
)

type PhaseType int8
type EventType int8
type Zone int8

const (
	PhaseStart PhaseType = iota
	PhaseDraw
	PhasePlay
	PhaseEnd

	NoEvent EventType = iota
	AllEvents
	EventAtStartPhase
	EventAtDrawPhase
	EventAtPlayPhase
	EventAtEndPhase
	EventOnDraw
	EventOnPlay
	EventOnEnterBoard
	EventOnLeaveBoard
	EventOnDestroy
	EventOnSacrifice
	EventOnTarget
	EventOnActivate
	EventOnDeactivate
	EventOnAttack
	EventOnBlock
	EventOnDamage
	EventOnPlayerDamage
	EventOnHeal
	EventOnCounter
	EventOnDiscard
	EventOnLoseLife
	EventOnGainLife
	EventOnWin
	EventOnLose

	ZoneAny Zone = iota
	ZoneDeck
	ZoneHand
	ZoneBoard
	ZonePile
	ZoneStack

	ErrorCode = -1
	SkipCode  = -2
)

const boardSize = 5
const startCards = 3
const startLife = 20

type CommandI interface {
	Card(*Player, []*CardInstance) (int, error)
	Field(*Player, []int) (int, error)
	Ability(*Player, []*Activated, *CardInstance) (int, error)
	Target(*Player, []*CardInstance, int) ([]int, error)
	Discard(*Player, []*CardInstance, int) ([]int, error)
}

type GameObject interface {
	GetId() ulid.ULID
}

func (phase PhaseType) String() string {
	switch phase {
	case PhaseStart:
		return "start"
	case PhaseDraw:
		return "draw"
	case PhasePlay:
		return "play"
	case PhaseEnd:
		return "end"
	}
	return "unknown"
}

func (e EventType) String() string {
	switch e {
	case NoEvent:
		return "none"
	case AllEvents:
		return "all"
	case EventAtStartPhase:
		return "start-phase"
	case EventAtDrawPhase:
		return "draw-phase"
	case EventAtPlayPhase:
		return "play-phase"
	case EventAtEndPhase:
		return "end-phase"
	case EventOnDraw:
		return "draw"
	case EventOnPlay:
		return "play"
	case EventOnEnterBoard:
		return "enter-board"
	case EventOnLeaveBoard:
		return "leave-board"
	case EventOnDestroy:
		return "destroy"
	case EventOnSacrifice:
		return "sacrifice"
	case EventOnTarget:
		return "target"
	case EventOnActivate:
		return "activate"
	case EventOnDeactivate:
		return "deactivate"
	case EventOnAttack:
		return "attack"
	case EventOnBlock:
		return "block"
	case EventOnDamage:
		return "damage"
	case EventOnPlayerDamage:
		return "player-damage"
	case EventOnHeal:
		return "heal"
	case EventOnCounter:
		return "counter"
	case EventOnDiscard:
		return "discard"
	case EventOnLoseLife:
		return "lose-life"
	case EventOnGainLife:
		return "gain-life"
	case EventOnWin:
		return "win"
	case EventOnLose:
		return "lose"
	}
	return "unknown"
}

type Event struct {
	Event   EventType
	Source  *AbilityInstance
	Subject GameObject
	Args   []any
}

type Board struct {
	slots []*CardInstance
}

func (b *Board) Insert(card *CardInstance, index int) {
	if index < 0 || index > len(b.slots) {
		panic("Invalid index")
	}
	b.slots[index] = card
}

func (b *Board) Remove(card *CardInstance) {
	for i, c := range b.slots {
		if c == card {
			b.slots[i] = nil
			return
		}
	}
}

type Pile struct {
	cards []*CardInstance
}

func (p *Pile) Add(cards ...*CardInstance) { p.cards = append(p.cards, cards...) }

func (p *Pile) Remove(card *CardInstance) {
	for i, c := range p.cards {
		if c == card {
			p.cards = append(p.cards[:i], p.cards[i+1:]...)
			return
		}
	}
}

func (p *Pile) Pop() *CardInstance {
	if len(p.cards) == 0 {
		return nil
	}
	card := p.cards[0]
	p.cards = p.cards[1:]
	return card
}

func (p *Pile) Shuffle() {
	for i := range p.cards {
		j := rand.Intn(i + 1)
		p.cards[i], p.cards[j] = p.cards[j], p.cards[i]
	}
}

func (p *Pile) Insert(card *CardInstance, i int) {
	if i < 0 {
		p.cards = append(p.cards, card)
	} else {
		p.cards = append(p.cards[:i], append([]*CardInstance{card}, p.cards[i:]...)...)
	}
}

type Player struct {
	Id         ulid.ULID
	game       *Game
	nr         int
	life       int
	deck       Pile
	hand       Pile
	pile       Pile
	board      Board
	essence    []string
	turnsAfter int
	cmdi       CommandI
}

func NewPlayer(cmdi CommandI, deck ...*Card) *Player {
	p := &Player{
		Id:      ulid.Make(),
		life:    0,
		deck:    Pile{cards: []*CardInstance{}},
		hand:    Pile{cards: []*CardInstance{}},
		pile:    Pile{cards: []*CardInstance{}},
		board:   Board{slots: make([]*CardInstance, boardSize)},
		essence: []string{},
		cmdi:    cmdi,
	}
	for _, card := range deck {
		p.deck.Add(NewCardInstance(card, p, ZoneDeck))
	}
	return p
}

func (p *Player) GetId() ulid.ULID { return p.Id }

func (p *Player) Run() bool {
	for {
		selected := []int{0}
		choices := p.GetPlayableCards()
		if !p.prompt("card", 1, choices, &selected, nil) || selected[0] < 0 || selected[0] >= len(choices) {
			return selected[0] == SkipCode
		}
		card := choices[selected[0]].(*CardInstance)
		if card.CanDo() {
			activatable := card.GetActivatable()
			if !p.prompt("ability", 1, activatable, &selected, card) {
				if selected[0] == SkipCode {
					continue
				}
				return false
			}
			if selected[0] >= 0 || selected[0] < len(activatable) {
				a := activatable[selected[0]].(*Activated)
				e := card.Do(a)
				if a.IsCost() {
					e.Resolve()
				} else {
					p.game.Play(e)
				}
				continue
			}
		} else {
			flipped := card.CanSource()
			if flipped && card.CanPlay() {
				if !p.prompt("source", 1, nil, &selected, nil) || selected[0] < 0 {
					return selected[0] == SkipCode
				}
				flipped = selected[0] == 1
			}
			fields := p.freeFields(card)
			if !p.prompt("field", 1, fields, &selected, nil) || selected[0] < 0 {
				if selected[0] == SkipCode {
					continue
				}
				return false
			}
			card.flipped = flipped
			if flipped {
				p.game.turn.sourcesPlayed += 1
				card.Play(fields[selected[0]].(int))
			} else {
				p.game.Play(card.Cast(fields[selected[0]].(int)))
			}
		}
	}
}

func convertArray[T any](arr []any) []T {
	res := make([]T, len(arr))
	for i, a := range arr {
		res[i] = a.(T)
	}
	return res
}

func (p *Player) prompt(
	cmd string,
	num int,
	choices []any,
	selected *[]int,
	context any,
) bool {
	var err error
	var arr []int
	if p.cmdi != nil {
		if cmd == "card" {
			(*selected)[0], err = p.cmdi.Card(p, convertArray[*CardInstance](choices))
		} else if cmd == "field" {
			(*selected)[0], err = p.cmdi.Field(p, convertArray[int](choices))
		} else if cmd == "ability" {
			(*selected)[0], err = p.cmdi.Ability(p, convertArray[*Activated](choices), context.(*CardInstance))
		} else if cmd == "target" {
			arr, err = p.cmdi.Target(p, convertArray[*CardInstance](choices), num)
			*selected = append(*selected, arr...)
		} else if cmd == "discard" {
			arr, err = p.cmdi.Discard(p, convertArray[*CardInstance](choices), num)
			*selected = append(*selected, arr...)
		}
	}
	if err != nil {
		(*selected)[0] = ErrorCode
	}
	return err == nil
}

func (p *Player) GainLife(n int) {
	p.life += n
	p.game.Emit(EventOnGainLife, p)
}

func (p *Player) LoseLife(n int) {
	p.life -= n
	p.game.Emit(EventOnLoseLife, p)
	if p.life <= 0 {
		p.game.Emit(EventOnLose, p)
	}
}

func (p *Player) Draw(n int) {
	for i := 0; i < n; i++ {
		card := p.deck.Pop()
		if card != nil {
			p.Place(card, ZoneHand, 0)
			p.game.Emit(EventOnDraw, card)
		} else {
			p.game.Emit(EventOnLose, p)
		}
		// TODO: handle empty deck
	}
}

func (p *Player) Shuffle(zone Zone) {
	switch zone {
	case ZoneDeck:
		p.deck.Shuffle()
	case ZonePile:
		p.pile.Shuffle()
	default:
		panic("Invalid zone")
	}
}

func (p *Player) Place(card *CardInstance, zone Zone, index int) {
	card.Controller.Remove(card)
	switch zone {
	case ZoneDeck:
		p.deck.Insert(card, index)
	case ZonePile:
		p.pile.Insert(card, index)
	case ZoneHand:
		p.hand.Add(card)
	case ZoneBoard:
		p.board.Insert(card, index)
		card.Controller = p
		p.game.Emit(EventOnEnterBoard, card, index)
	default:
		panic("Invalid zone")
	}
	card.zone = zone
	card.index = index
}

func (p *Player) Remove(card *CardInstance) {
	switch card.zone {
	case ZoneHand:
		p.hand.Remove(card)
	case ZonePile:
		p.pile.Remove(card)
	case ZoneBoard:
		p.board.Remove(card)
		p.game.Emit(EventOnLeaveBoard, card)
	case ZoneDeck:
		p.deck.Remove(card)
	default:
		panic("Invalid zone")
	}
}

func (p *Player) Match(a *AbilityInstance, m Match) bool {
	if m == nil {
		// No player match given so this is an imperative (Draw 2 cards). Subject is the caster.
		return a.Controller == p
	}
	return m.Match(a, p)
}

func (p *Player) Query(a *AbilityInstance, obj Match, zone *ZoneMatch) []any {
	found := []any{}
	if p.matchField(a, ZoneBoard, zone) {
		for _, card := range p.board.slots {
			if card != nil && (obj == nil || obj.Match(a, card)) {
				found = append(found, card)
			}
		}
	}
	if p.matchField(a, ZoneHand, zone) {
		for _, card := range p.hand.cards {
			if obj == nil || obj.Match(a, card) {
				found = append(found, card)
			}
		}
	}
	if p.matchField(a, ZonePile, zone) {
		for _, card := range p.pile.cards {
			if obj == nil || obj.Match(a, card) {
				found = append(found, card)
			}
		}
	}
	if p.matchField(a, ZoneDeck, zone) {
		for _, card := range p.deck.cards {
			if obj == nil || obj.Match(a, card) {
				found = append(found, card)
			}
		}
	}
	if p.matchField(a, ZoneStack, zone) {
		for _, card := range p.game.stack.cards {
			if card.Source.Owner == p && (obj == nil || obj.Match(a, card.Source)) {
				found = append(found, card.Source)
			}
		}
	}
	return found
}

func (p *Player) CanPay(card *CardInstance, costs []AbilityCost) bool {
	// TODO: add potential essence from essence sources
	// TODO: add extra costs
	pool := make([]string, len(p.essence))
	copy(pool, p.essence)
	for _, ct := range costs {
		if ct.Cost != nil {
			if ct.Cost.Activate {
				if card.activated {
					return false
				}
			} else if ct.Cost.Deactivate {
				if !card.activated {
					return false
				}
			} else if ct.Cost.Color != "" {
				found := false
				for i, c := range pool {
					if c == ct.Cost.Color {
						pool = append(pool[:i], pool[i+1:]...)
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
		}
	}
	for _, ct := range costs {
		if ct.Cost != nil &&
			!ct.Cost.Activate &&
			!ct.Cost.Deactivate &&
			ct.Cost.Color == "" {
			num := ct.Cost.Number.Number
			if len(pool) < num {
				return false
			}
			for i := 0; i < num; i++ {
				found := false
				for j, c := range pool {
					if c == "u" {
						pool = append(pool[:j], pool[j+1:]...)
						found = true
						break
					}
				}
				if !found {
					pool = append(pool[:0], pool[1:]...)
				}
			}
		}
	}
	return true
}

func (p *Player) Pay(card *CardInstance, costs []AbilityCost) {
	// TODO: pay action costs
	// TODO: add choice of essence sources
	// TODO: pay X
	for _, cost := range costs {
		if cost.Cost != nil {
			if cost.Cost.Activate {
				card.Activate()
			} else if cost.Cost.Deactivate {
				card.Deactivate()
			} else {
				p.RemoveEssence(cost.Cost.Color)
			}
		}
	}
	for _, ct := range costs {
		if ct.Cost != nil &&
			!ct.Cost.Activate &&
			!ct.Cost.Deactivate &&
			ct.Cost.Color == "" {
			for i := 0; i < ct.Cost.Number.Number; i++ {
				p.RemoveEssence("u")
			}
		}
	}
}

func (p *Player) AddEssence(t string) {
	p.essence = append(p.essence, t)
}

func (p *Player) RemoveEssence(t string) bool {
	for i, ct := range p.essence {
		if ct == t {
			p.essence = append(p.essence[:i], p.essence[i+1:]...)
			return true
		}
	}
	if t == "u" && len(p.essence) > 0 {
		p.essence = append(p.essence[:0], p.essence[1:]...)
		return true
	}
	return false
}

func (p *Player) HasEssence(t string) bool {
	for _, ct := range p.essence {
		if ct == t {
			return true
		}
	}
	if t == "u" && len(p.essence) > 0 {
		return true
	}
	return false
}

func (p *Player) SourcesPerTurn() int {
	// TODO: could be modified by other cards
	return 1
}

func (p *Player) GetPlayableCards() []any {
	// TODO: get playable cards from other zones
	playable := []any{}
	for _, card := range p.hand.cards {
		if card.CanPlay() || card.CanSource() {
			playable = append(playable, card)
		}
	}
	for _, card := range p.board.slots {
		if card != nil && card.CanDo() {
			playable = append(playable, card)
		}
	}
	return playable
}

func (p *Player) matchField(ability *AbilityInstance, place Zone, zone *ZoneMatch) bool {
	if zone == nil {
		return true
	}
	return zone.Match(ability, place, p)
}

func (p *Player) freeFields(card *CardInstance) []any {
	choices := []any{}
	for i, card := range p.board.slots {
		if card == nil {
			choices = append(choices, i)
		}
		// TODO: check for other free fields (on top of other cards?)
	}
	return choices
}

type Stack struct {
	cards []*AbilityInstance
}

func (p *Stack) Add(cards ...*AbilityInstance) {
	p.cards = append(p.cards, cards...)
}

func (p *Stack) Pop() *AbilityInstance {
	card := p.Top()
	if card != nil {
		p.cards = p.cards[:len(p.cards)-1]
	}
	return card
}

func (p *Stack) Top() *AbilityInstance {
	if len(p.cards) == 0 {
		return nil
	}
	return p.cards[len(p.cards)-1]
}

func (p *Stack) Remove(card *AbilityInstance) {
	for i, c := range p.cards {
		if c == card {
			p.cards = append(p.cards[:i], p.cards[i+1:]...)
			return
		}
	}
}

type EffectInstance struct {
	Ability  *AbilityInstance
	Subjects []any
	Effect   Effect
	Match    Match
	Zone     *ZoneMatch
	matches  []any
	zones    []Zone
}

type AbilityInstance struct {
	Source     *CardInstance
	Controller *Player
	Ability    Ability
	Effects    []EffectInstance
	This       []any
	Sacrificed []any
	Targeting  []any
	Field      int
	X          int
	Event      EventType
}

func NewAbilityInstance(p *Player, c *CardInstance, f Ability) *AbilityInstance {
	return &AbilityInstance{
		Source:     c,
		Controller: p,
		Ability:    f,
		Effects:    []EffectInstance{},
		This:       []any{},
		Sacrificed: []any{},
		Targeting:  []any{},
		Event:      p.game.currentEvent,
	}
}

func (a *AbilityInstance) Resolve() {
	a.Controller.game.resolving = a
	if len(a.Effects) == 0 {
		a.Controller.Place(a.Source, ZoneBoard, a.Field)
	}
	for _, e := range a.Effects {
		e.Effect.Resolve(&e)
	}
	a.Controller.game.resolving = nil
}

type Phase struct {
	turn     *Turn
	priority *Player
	phase    PhaseType
}

type Turn struct {
	game          *Game
	player        *Player
	phase         *Phase
	turn          int
	sourcesPlayed int
}

type EventHandler func(*Event)

type Game struct {
	Id            ulid.ULID
	players       []*Player
	stack         Stack
	turn          *Turn
	currentEvent  EventType
	resolving     *AbilityInstance
	eventHandlers map[EventType][]EventHandler
}

func NewGame(players ...*Player) *Game {
	g := &Game{
		Id:            ulid.Make(),
		players:       players,
		stack:         Stack{cards: []*AbilityInstance{}},
		eventHandlers: map[EventType][]EventHandler{},
	}
	for _, player := range g.players {
		player.game = g
	}
	return g
}

func (g *Game) Run() {
	if len(g.players) == 0 {
		panic("No players")
	}
	nrPlayers := len(g.players)
	beginningPlayer := rand.Intn(nrPlayers)
	for i := 0; i < nrPlayers; i++ {
		g.players[(beginningPlayer+i)%nrPlayers].nr = i + 1
	}
	turn := 1
	p := g.players[beginningPlayer]
	for _, player := range g.players {
		player.life = startLife
		player.Draw(startCards)
	}
	for {
		g.turn = &Turn{g, p, nil, turn, 0}
		for phase := range g.turn.Iter() {
			for player := range phase.Iter() {
				if !player.Run() {
					return
				}
			}
		}
		turn += 1
		if p.turnsAfter > 0 {
			p.turnsAfter -= 1
		} else {
			p = g.nextPlayer(p)
		}
	}
}

func (g *Game) AddPlayer(p *Player) {
	g.players = append(g.players, p)
	p.game = g
}

func (g *Game) On(event EventType, handler EventHandler) {
	if _, ok := g.eventHandlers[event]; !ok {
		g.eventHandlers[event] = []EventHandler{}
	}
	g.eventHandlers[event] = append(g.eventHandlers[event], handler)
}

func (g *Game) Emit(event EventType, subject GameObject, args ...any) {
	g.currentEvent = event
	e := &Event{event, g.resolving, subject, args}
	for _, player := range g.players {
		// TODO: Check cards in other zones
		for _, card := range player.board.slots {
			if card != nil {
				card.Trigger(e)
			}
		}
	}
	g.callHandlers(event, e)
	g.callHandlers(AllEvents, e)
	g.currentEvent = NoEvent
}

func (g *Game) callHandlers(e EventType, event *Event) {
	if handlers, ok := g.eventHandlers[e]; ok {
		for _, f := range handlers {
			f(event)
		}
	}
}

func (g *Game) IsReaction() bool {
	return g.turn.phase.phase != PhasePlay || len(g.stack.cards) > 0
}

func (g *Game) Play(a *AbilityInstance) {
	for i := 0; i < len(a.Effects); i++ {
		e := &a.Effects[i]
		if e.Match != nil {
			e.matches = g.Pick(a, e.Match, e.Zone)
		}
	}
	g.stack.Add(a)
}

func (g *Game) Pick(a *AbilityInstance, o Match, z *ZoneMatch) []any {
	if o == nil {
		found := g.Query(a, o, z, -1)
		if len(found) == 0 {
			return a.Targeting
		}
		targeted := []int{}
		if !a.Controller.prompt("target", 1, found, &targeted, nil) {
			return nil
		}
		for _, i := range targeted {
			a.Targeting = append(a.Targeting, found[i])
		}
		return found
	}
	n := o.NrTargets(a)
	if n > 0 {
		for i := 0; i < n; i++ {
			found := g.Query(a, o, z, -1)
			if len(found) == 0 {
				return a.Targeting
			}
			targeted := []int{}
			if !a.Controller.prompt("target", 1, found, &targeted, nil) {
				return nil
			}
			for _, i := range targeted {
				a.Targeting = append(a.Targeting, found[i])
			}
		}
		return a.Targeting
	}
	return g.Query(a, o, z, -1)
}

func (g *Game) Query(a *AbilityInstance, o Match, z *ZoneMatch, n int) []any {
	found := []any{}
	for _, player := range g.players {
		if player.Match(a, o) {
			found = append(found, player)
		}
	}
	if o != nil {
		for _, player := range g.players {
			found = append(found, player.Query(a, o, z)...)
		}
	}
	if n > 0 {
		return found[:n]
	}
	return found
}

func (t *Turn) Iter() iter.Seq[*Phase] {
	return func(yield func(*Phase) bool) {
		t.phase = &Phase{t, t.player, PhaseStart}
		t.game.Emit(EventAtStartPhase, t.player)
		t.player.essence = t.player.essence[:0]
		for _, card := range t.player.board.slots {
			if card != nil {
				card.Activate()
			}
		}
		if !yield(t.phase) {
			return
		}
		t.phase = &Phase{t, t.player, PhaseDraw}
		t.game.Emit(EventAtDrawPhase, t.player)
		t.player.Draw(1)
		if !yield(t.phase) {
			return
		}
		t.phase = &Phase{t, t.player, PhasePlay}
		t.game.Emit(EventAtPlayPhase, t.player)
		if !yield(t.phase) {
			return
		}
		t.phase = &Phase{t, t.player, PhaseEnd}
		t.game.Emit(EventAtEndPhase, t.player)
		if !yield(t.phase) {
			return
		}
	}
}

func (p *Phase) Iter() iter.Seq[*Player] {
	return func(yield func(*Player) bool) {
		for {
			for i := 0; i < len(p.turn.game.players); i++ {
				if !yield(p.priority) {
					return
				}
				p.priority = p.turn.game.nextPlayer(p.priority)
			}
			if len(p.turn.game.stack.cards) == 0 {
				return
			}
			p.turn.game.stack.Pop().Resolve()
		}
	}
}

func (g *Game) nextPlayer(p *Player) *Player {
	for i, player := range g.players {
		if player == p {
			return g.players[(i+1)%len(g.players)]
		}
	}
	return nil
}

type CardParser struct {
	parser *participle.Parser[Card]
}

func NewCardParser() *CardParser {
	parser := participle.MustBuild[Card](
		participle.Lexer(lexer.MustSimple([]lexer.SimpleRule{
			{"whitespace", `[\s]+`},
			{"Ident", `[a-zA-Z]\w*`},
			{"Punct", `[-+,{}/:.]`},
			{"Int", `\d+`},
		})),
		//participle.UseLookahead(2),
		participle.Union[Ability](Keyword{}, Composed{}, Activated{}, Triggered{}),
		participle.Union[Effect](
			PlayerSubjectAbility{},
			CardSubjectAbility{},
		),
		participle.Union[Match](
			AnyMatch{},
			PlayerMatch{},
			CardMatch{},
		),
		participle.Union[PlayerEffect](
			Draw{},
			Token{},
			Destroy{},
			Add{},
			GainLife{},
			LoseLife{},
			Discard{},
			Shuffle{},
			ExtraTurn{},
			Look{},
			Put{},
			Activate{},
			Deactivate{},
			Sacrifice{},
			PayEssence{},
			PayLife{},
		),
		participle.Union[CardEffect](
			Damage{},
			Gets{},
		),
		participle.UseLookahead(3),
	)
	fmt.Printf("parser: %s\n", parser.String())
	return &CardParser{parser}
}

func (p *CardParser) Parse(txt string) (*Card, error) {
	name := strings.TrimSpace(strings.SplitN(strings.SplitN(strings.TrimSpace(txt), "\n", 2)[0], "{", 2)[0])
	input := strings.ReplaceAll(strings.ToLower(txt), strings.ToLower(name), "NAME")
	card, err := p.parser.ParseString("", input)
	if err != nil {
		return nil, err
	}
	card.Name = name
	card.Text = txt
	return card, nil
}

type NumberOrX struct {
	Number int  `@Int`
	A      bool `| @("a"|"an")`
	X      bool `| @"X"`
}

func (n NumberOrX) String() string {
	if n.X {
		return "X"
	} else if n.A {
		return "a"
	}
	return fmt.Sprintf("%d", n.Number)
}

func (n NumberOrX) Value(a *AbilityInstance) int {
	if n.X {
		return a.X
	} else if n.A {
		return 1
	}
	return n.Number
}

type CostType struct {
	Color      string    `"{"( @("c"|"o"|"s"|"w")`
	Activate   bool      `| @"q"`
	Deactivate bool      `| @"t"`
	Number     NumberOrX `| @@ )"}"`
}

func (c CostType) String() string {
	if c.Color != "" {
		return fmt.Sprintf("{%s}", c.Color)
	} else if c.Activate {
		return "{q}"
	} else if c.Deactivate {
		return "{t}"
	}
	return c.Number.String()
}

func (c CostType) Pay(a *AbilityInstance, p *Player) bool {
	if c.Color != "" {
		if !p.RemoveEssence(c.Color) {
			return false
		}
	} else {
		n := c.Number.Value(a)
		for i := 0; i < n; i++ {
			if !p.RemoveEssence("u") {
				return false
			}
		}
	}
	return true
}

type Color struct {
	Value string `@("cup"|"coin"|"sword"|"wand"|"wild")`
}

type CardType struct {
	Value string `@(
"unit"|"units"|
"item"|"items"|
"source"|"sources"|
"spell"|"spells"|
"token"|"tokens"
)`
}

type SubType struct {
	Value string `@(
"human"|
"dwarf"|
"elf"|
"orc"|
"gnome"|
"undead"|
"demon"|
"dragon"|
"golem"|
"spirit"|
"soldier"|
"wizard"|
"warrior"|
"merchant"|
"cleric"|
"rogue"|
"beast"
)`
}

type Stats struct {
	Power  NumberOrX `@@`
	Health NumberOrX `"/" @@`
}

type Modifier interface {
	Apply(*CardInstance)
	Reverse(*CardInstance)
}

type Mods struct {
	Power  int
	Health int
}

func (m Mods) Apply(c *CardInstance) {
	c.stats.Power.Number += m.Power
	c.stats.Health.Number += m.Health
}

func (m Mods) Reverse(c *CardInstance) {
	c.stats.Power.Number -= m.Power
	c.stats.Health.Number -= m.Health
}

type Card struct {
	Name      string     `@Ident`
	Costs     []CostType `@@*`
	Types     []CardType `@@+`
	Subtypes  []SubType  `("-" @@*)?`
	Abilities []Ability  `(@@ (",")?)*`
	Stats     *Stats     `@@?`
	Text      string
}

type CardInstance struct {
	Id         ulid.ULID
	Card       *Card
	activated  bool
	flipped    bool
	zone       Zone
	index      int
	Owner      *Player
	Controller *Player
	stats      *Stats
	modifier   []Mods
}

func NewCardInstance(card *Card, owner *Player, zone Zone) *CardInstance {
	c := &CardInstance{
		Id:         ulid.Make(),
		Card:       card,
		Owner:      owner,
		Controller: owner,
		zone:       zone,
		modifier:   []Mods{},
	}
	if card.Stats != nil {
		c.stats = &Stats{card.Stats.Power, card.Stats.Health}
	}
	return c
}

func (c *CardInstance) GetId() ulid.ULID { return c.Id }

func (c *CardInstance) GetName() string { return c.Card.Name }

func (c *CardInstance) GetPower() NumberOrX {
	return c.stats.Power
}

func (c *CardInstance) GetHealth() NumberOrX {
	return c.stats.Health
}

func (c *CardInstance) GetActivatable() []any {
	abilities := []any{}
	for _, a := range c.GetActivatedAbilities() {
		if a.CanDo(c) {
			abilities = append(abilities, a)
		}
	}
	return abilities
}

func (c *CardInstance) TakeDamage(n int) {
	// TODO: check for protection of source
	c.stats.Health.Number -= n
	c.Owner.game.Emit(EventOnDamage, c)
	if c.stats.Health.Number <= 0 {
		// TODO: check invurnability
		c.Controller.Remove(c)
		c.Owner.Place(c, ZonePile, -1)
		c.Owner.game.Emit(EventOnDestroy, c)
	}
}

func (c *CardInstance) Activate() {
	c.activated = true
	c.Owner.game.Emit(EventOnActivate, c)
}

func (c *CardInstance) Deactivate() {
	c.activated = false
	c.Owner.game.Emit(EventOnDeactivate, c)
}

func (c *CardInstance) Do(a *Activated) *AbilityInstance {
	player := c.Owner.game.turn.phase.priority
	player.Pay(c, a.Cost)
	return a.Do(player, c)
}

func (c *CardInstance) Play(index int) {
	c.activated = true
	player := c.Owner.game.turn.phase.priority
	player.Place(c, ZoneBoard, index)
	player.game.Emit(EventOnPlay, c)
}

func (c *CardInstance) Cast(index int) *AbilityInstance {
	player := c.Owner.game.turn.phase.priority
	player.Remove(c)
	player.Pay(c, c.GetCosts())
	return &AbilityInstance{Source: c, Controller: player, Field: index}
}

func (c *CardInstance) Trigger(event *Event) {
	for _, t := range c.GetTriggeredAbilities() {
		a := t.Do(c.Controller, c)
		if a != nil {
			c.Controller.game.Play(a)
		}
	}
}

func (c *CardInstance) CanDo() bool {
	for _, a := range c.GetActivatedAbilities() {
		if a.CanDo(c) && (a.IsCost() || !c.Owner.game.IsReaction()) {
			return true
		}
	}
	return false
}

func (c *CardInstance) CanReact() bool {
	// TODO: check if card can react or instant
	return c.Owner.game.turn.player == c.Owner && !c.Owner.game.IsReaction()
}

func (c *CardInstance) CanPlay() bool {
	// TODO: check castable from other locations
	if c.zone != ZoneHand || !c.CanReact() {
		return false
	}
	return c.Owner.game.turn.phase.priority.CanPay(c, c.GetCosts())
}

func (c *CardInstance) CanSource() bool {
	if c.zone != ZoneHand || !c.CanReact() {
		return false
	}
	t := c.Owner.game.turn
	return t.sourcesPlayed >= t.phase.priority.SourcesPerTurn()
}

func (c *CardInstance) GetCosts() []AbilityCost {
	costs := []AbilityCost{}
	for _, ct := range c.Card.Costs {
		costs = append(costs, AbilityCost{Cost: &ct})
	}
	return costs
}

func (c *CardInstance) HasKeyword(k string) bool {
	for _, a := range c.Card.Abilities {
		if keyword, ok := a.(*Keyword); ok && keyword.Value == k {
			return true
		}
	}
	return false
}

func (c *CardInstance) HasType(t string) bool {
	for _, ct := range c.Card.Types {
		if ct.Value == t {
			return true
		}
	}
	return false
}

func (c *CardInstance) HasSubType(t string) bool {
	for _, ct := range c.Card.Subtypes {
		if ct.Value == t {
			return true
		}
	}
	return false
}

func (c *CardInstance) HasColor(t string) bool {
	for _, ct := range c.Card.Costs {
		if ct.Color == t {
			return true
		}
	}
	return false
}

func (c *CardInstance) GetActivatedAbilities() []*Activated {
	abilities := []*Activated{}
	for _, a := range c.Card.Abilities {
		if ab, ok := a.(Activated); ok {
			abilities = append(abilities, &ab)
		}
	}
	if c.HasType("unit") {
		abilities = append(abilities, AttackAbility)
	}
	if c.HasType("source") {
		if c.HasColor("s") {
			abilities = append(abilities, SEssenseAbility)
		}
		if c.HasColor("o") {
			abilities = append(abilities, OEssenseAbility)
		}
		if c.HasColor("c") {
			abilities = append(abilities, CEssenseAbility)
		}
		if c.HasColor("w") {
			abilities = append(abilities, WEssenseAbility)
		}
	}
	return abilities
}

func (c *CardInstance) GetTriggeredAbilities() []*Triggered {
	abilities := []*Triggered{}
	for _, a := range c.Card.Abilities {
		if ab, ok := a.(Triggered); ok {
			abilities = append(abilities, &ab)
		}
	}
	return abilities
}

type Attack struct{}

func (f Attack) HasTarget() bool      { return false }
func (f Attack) IsCost() bool         { return false }
func (f Attack) Do(a *EffectInstance) {}
func (f Attack) Resolve(e *EffectInstance) {
	for _, c := range e.Subjects {
		card := c.(*CardInstance)
		var o *Player
		for _, p := range card.Controller.game.players {
			if p != card.Controller {
				o = p
			}
		}
		if o != nil {
			if other := o.board.slots[card.index]; other != nil {
				other.TakeDamage(card.stats.Power.Number)
			} else {
				o.LoseLife(card.stats.Power.Number)
			}
		}
	}
}

func createEssenceAbility(color string) *Activated {
	return &Activated{
		[]AbilityCost{{Cost: &CostType{Deactivate: true}}},
		Composed{[]Effect{PlayerSubjectAbility{
			nil, false, []PlayerEffect{Add{[]CostType{{Color: color}}}},
		}}},
	}
}

var SEssenseAbility = createEssenceAbility("s")
var OEssenseAbility = createEssenceAbility("o")
var CEssenseAbility = createEssenceAbility("c")
var WEssenseAbility = createEssenceAbility("w")

var AttackAbility = &Activated{
	[]AbilityCost{{Cost: &CostType{Deactivate: true}}},
	Composed{[]Effect{CardSubjectAbility{nil, []CardEffect{Attack{}}}}},
}

type Ability interface{ value() }

type Effect interface {
	HasTarget() bool
	IsCost() bool
	Do(*EffectInstance)
	Resolve(*EffectInstance)
}
type CardEffect interface {
	HasTarget() bool
	IsCost() bool
	Do(*EffectInstance)
	Resolve(*EffectInstance)
}
type PlayerEffect interface {
	HasTarget() bool
	IsCost() bool
	Do(*EffectInstance)
	Resolve(*EffectInstance)
}

type Keyword struct {
	Value string `@("fly"|"siege"|"poison"|"ambush")`
}

func (f Keyword) value() {}

type AbilityCost struct {
	Cost   *CostType `@@`
	Action *Effect   `| @@`
}

type Composed struct {
	Effects []Effect `@@ ("," ("then"|"and")? @@)* "."`
}

func (f Composed) value() {}

func (f Composed) Do(p *Player, a *AbilityInstance) {
	for _, e := range f.Effects {
		effect := EffectInstance{Ability: a, Effect: e}
		e.Do(&effect)
	}
}

func (f Composed) HasTarget() bool {
	for _, e := range f.Effects {
		if e.HasTarget() {
			return true
		}
	}
	return false
}

func (f Composed) IsCost() bool {
	if f.HasTarget() {
		return false
	}
	for _, e := range f.Effects {
		if !e.IsCost() {
			return false
		}
	}
	return true
}

type CardSubjectAbility struct {
	Match   *CardMatch   `@@?`
	Effects []CardEffect `@@ ((",") ("then"|"and")? @@)*`
}

func (f CardSubjectAbility) HasTarget() bool { return f.Match != nil && f.Match.HasTarget() }

func (f CardSubjectAbility) IsCost() bool {
	if f.Match != nil && f.Match.HasTarget() {
		return false
	}
	for _, e := range f.Effects {
		if e.HasTarget() {
			return false
		} else if e.IsCost() {
			return true
		}
	}
	return false
}

func (f CardSubjectAbility) Do(e *EffectInstance) {
	a := e.Ability
	m := CardSelf
	if f.Match != nil {
		m = *f.Match
	}
	cardSubject := a.Controller.game.Query(a, m, nil, -1)
	for _, ef := range f.Effects {
		effect := EffectInstance{
			Ability:  a,
			Effect:   ef,
			Subjects: cardSubject,
		}
		ef.Do(&effect)
		a.Effects = append(a.Effects, effect)
	}
}

func (f CardSubjectAbility) Resolve(e *EffectInstance) {}

type PlayerSubjectAbility struct {
	Match    *PlayerMatch   `(@@`
	Optional bool           `@("may")? )?`
	Effects  []PlayerEffect `@@ ((",") ("then"|"and")? @@)*`
}

func (f PlayerSubjectAbility) HasTarget() bool {
	if f.Match == nil {
		return false
	}
	return f.Match.HasTarget()
}

func (f PlayerSubjectAbility) IsCost() bool {
	if f.Match != nil && f.Match.HasTarget() {
		return false
	}
	for _, e := range f.Effects {
		if e.HasTarget() {
			return false
		} else if e.IsCost() {
			return true
		}
	}
	return false
}

func (f PlayerSubjectAbility) Do(e *EffectInstance) {
	a := e.Ability
	m := PlayerSelf
	if f.Match != nil {
		m = *f.Match
	}
	playerSubject := a.Controller.game.Query(a, m, nil, -1)
	for _, ef := range f.Effects {
		effect := EffectInstance{
			Ability:  a,
			Effect:   ef,
			Subjects: playerSubject,
		}
		ef.Do(&effect)
		a.Effects = append(a.Effects, effect)
	}
}

func (f PlayerSubjectAbility) Resolve(e *EffectInstance) {}

type Activated struct {
	Cost   []AbilityCost `@@+ ":"`
	Effect Composed      `@@`
}

func (f Activated) value() {}

func (f Activated) CanDo(card *CardInstance) bool {
	if card.zone != ZoneBoard {
		// TOOO: check if card can activate from other locations
		return false
	}
	return card.Controller.CanPay(card, f.Cost)
}

func (f *Activated) Do(p *Player, c *CardInstance) *AbilityInstance {
	a := NewAbilityInstance(p, c, f)
	p.Pay(c, f.Cost)
	f.Effect.Do(p, a)
	return a
}

func (f Activated) IsCost() bool {
	if f.Effect.HasTarget() {
		return false
	}
	return f.Effect.IsCost()
}

func (f Activated) HasTarget() bool { return f.Effect.HasTarget() }

type PlayerCondition struct {
	Player    PlayerMatch `@@`
	Sacrifice *CardMatch  `("sacrifice"|"sacrifices") @@`
	Draw      bool        `| ("draw"|"draws") "a" "card"`
}

func (c PlayerCondition) Match(a *AbilityInstance, o *CardInstance) bool {
	if !c.Player.Match(a, o) {
		return false
	}
	if c.Sacrifice != nil {
		if a.Event != EventOnSacrifice || !c.Sacrifice.Match(a, o) {
			return false
		}
	} else if c.Draw {
		if a.Event != EventOnDraw {
			return false
		}
	}
	return true
}

func (c PlayerCondition) Do(p *Player, a *AbilityInstance) {
	a.This = p.game.Query(a, c.Player, nil, -1)
}

type CardCondition struct {
	Cards  CardMatch `@@`
	Enters bool      `@(("is"|"are") "put" "on" "the" "board")`
	Leaves bool      `| @(("leave"|"leave") "the" "board")`
}

func (c CardCondition) Match(a *AbilityInstance, o *CardInstance) bool {
	if !c.Cards.Match(a, o) {
		return false
	}
	if c.Enters {
		if a.Event != EventOnEnterBoard {
			return false
		}
	} else if c.Leaves {
		if a.Event != EventOnLeaveBoard {
			return false
		}
	}
	return true
}

func (c CardCondition) Do(p *Player, a *AbilityInstance) {
	a.This = p.game.Query(a, c.Cards, nil, -1)
}

type Condition struct {
	YourTurn        bool             `@("it's" "your" "turn")`
	NotYourTurn     bool             `| @("it's" "not" "your" "turn")`
	PlayerCondition *PlayerCondition `| @@`
	CardCondition   *CardCondition   `| @@`
	Number          *Numberical      `| ( @@ "is"`
	Compare         *Compare         `@@ )`
}

func (c Condition) Match(a *AbilityInstance, o *CardInstance) bool {
	if c.YourTurn {
		if a.Controller != a.Source.Owner.game.turn.player {
			return false
		}
	} else if c.NotYourTurn {
		if a.Controller == a.Source.Owner.game.turn.player {
			return false
		}
	} else if c.PlayerCondition != nil {
		if !c.PlayerCondition.Match(a, o) {
			return false
		}
	} else if c.CardCondition != nil {
		if !c.CardCondition.Match(a, o) {
			return false
		}
	} else if c.Number != nil {
		if !c.Compare.Compare(a, c.Number.Value(a, o)) {
			return false
		}
	}
	return true
}

func (c Condition) Do(p *Player, a *AbilityInstance) {
	if c.PlayerCondition != nil {
		c.PlayerCondition.Do(p, a)
	} else if c.CardCondition != nil {
		c.CardCondition.Do(p, a)
	}
}

type Trigger struct {
	Play        *CardMatch   `("when"|"whenever") ("you" "play" @@`
	GainLife    *PlayerMatch `| @@ ("gain"|"gains") "life"`
	LosesLife   *PlayerMatch `| @@ ("lost"|"loses") "life"`
	DealtDamage *CardMatch   `| @@ "is" "dealt" "damage"`
	Condition   *Condition   `| @@ )`
}

func (t Trigger) Match(a *AbilityInstance, o *CardInstance) bool {
	if t.Play != nil {
		if a.Event != EventOnPlay || !t.Play.Match(a, o) {
			return false
		}
	} else if t.GainLife != nil {
		if a.Event != EventOnGainLife || !t.GainLife.Match(a, o) {
			return false
		}
	} else if t.LosesLife != nil {
		if a.Event != EventOnLoseLife || !t.LosesLife.Match(a, o) {
			return false
		}
	} else if t.DealtDamage != nil {
		if a.Event != EventOnDamage || !t.DealtDamage.Match(a, o) {
			return false
		}
	} else if t.Condition != nil {
		if !t.Condition.Match(a, o) {
			return false
		}
	}
	return true
}

func (t Trigger) Do(p *Player, a *AbilityInstance) {
	if t.Play != nil {
		if a.Event == EventOnPlay {
			a.This = p.game.Query(a, t.Play, nil, -1)
		}
	} else if t.GainLife != nil {
		if a.Event == EventOnGainLife {
			a.This = p.game.Query(a, t.GainLife, nil, -1)
		}
	} else if t.LosesLife != nil {
		if a.Event == EventOnLoseLife {
			a.This = p.game.Query(a, t.LosesLife, nil, -1)
		}
	} else if t.DealtDamage != nil {
		if a.Event == EventOnDamage {
			a.This = p.game.Query(a, t.DealtDamage, nil, -1)
		}
	} else if t.Condition != nil {
		t.Condition.Do(p, a)
	}
}

type Triggered struct {
	Trigger Trigger  `@@ ","`
	Effect  Composed `@@`
}

func (f Triggered) value() {}

func (f Triggered) Do(p *Player, c *CardInstance) *AbilityInstance {
	a := NewAbilityInstance(p, c, f)
	if f.Trigger.Match(a, c) {
		f.Trigger.Do(p, a)
		f.Effect.Do(p, a)
		return a
	}
	return nil
}

var PlayerSelf = PlayerMatch{[]PlayerTypeMatch{{Self: true}}}
var CardSelf = CardMatch{[]CardTypeMatch{{Self: true}}}

type Match interface {
	Match(*AbilityInstance, any) bool
	HasTarget() bool
	NrTargets(*AbilityInstance) int
}

type Prefix struct {
	Color       Color    `@@`
	NonColor    Color    `| "non" "-" @@`
	Type        CardType `| @@`
	NonType     CardType `| "non" "-" @@`
	Activated   bool     `| "activated"`
	Deactivated bool     `| "deactivated"`
	Stats       *Stats   `| @@`
}

func (c Prefix) Match(a *AbilityInstance, card *CardInstance) bool {
	if c.Color.Value != "" {
		if !card.HasType(c.Color.Value) {
			return false
		}
	} else if c.NonColor.Value != "" {
		if card.HasType(c.NonColor.Value) {
			return false
		}
	} else if c.NonType.Value != "" {
		if card.HasType(c.NonType.Value) {
			return false
		}
	} else if c.Activated {
		if !card.activated {
			return false
		}
	} else if c.Deactivated {
		if card.activated {
			return false
		}
	} else if c.Stats != nil {
		if card.Card.Stats.Power.Number != c.Stats.Power.Number ||
			card.Card.Stats.Health.Number != c.Stats.Health.Number {
			return false
		}
	}
	return true
}

type Suffix struct {
	Targets *CardMatch `"that" "targets" @@`
}

func (c Suffix) Match(a *AbilityInstance, card *CardInstance) bool {
	if c.Targets != nil {
		for _, target := range a.Targeting {
			if c.Targets.Match(a, target) {
				return true
			}
		}
		return false
	}
	return true
}

type Numberical struct {
	Attribute string `@("damage"|"health")`
}

func (n Numberical) Value(a *AbilityInstance, c *CardInstance) int {
	if n.Attribute == "damage" {
		return c.stats.Power.Number
	} else if n.Attribute == "health" {
		return c.stats.Health.Number
	}
	panic("Invalid numberical")
}

type Count struct {
	Objects    *CardMatch  `"the" "number" "of" @@`
	Owner      *CardMatch  `| ( @@ "'s" `
	Numberical *Numberical `@@ )`
	Number     NumberOrX   `| @@`
}

func (c Count) Value(a *AbilityInstance) int {
	if c.Objects != nil {
		return len(a.Controller.game.Query(a, c.Objects, nil, -1))
	}
	if c.Owner != nil {
		o := a.Controller.game.Query(a, c.Objects, nil, 1)
		if len(o) == 0 {
			return -1
		}
		return c.Numberical.Value(a, o[0].(*CardInstance))
	}
	return c.Number.Value(a)
}

type Compare struct {
	GreaterThen bool  `(@("greater" "then")`
	LessThen    bool  `| @("less" "then"))?`
	Value       Count `@@`
	OrGreater   bool  `(@("or" "greater")`
	OrLess      bool  `| @("or" "less"))?`
}

func (c Compare) Compare(a *AbilityInstance, val int) bool {
	if c.GreaterThen {
		return val > c.Value.Value(a)
	} else if c.LessThen {
		return val < c.Value.Value(a)
	} else if c.OrGreater {
		return val >= c.Value.Value(a)
	} else if c.OrLess {
		return val <= c.Value.Value(a)
	}
	return val == c.Value.Value(a)
}

type With struct {
	Ability []Keyword   `"with" ( (@@ (("," @@)* "and" @@)?)`
	Number  *Numberical `| (@@`
	Compare *Compare    `@@))`
}

func (w With) Match(a *AbilityInstance, o any) bool {
	card := o.(*CardInstance)
	if w.Ability != nil {
		for _, k := range w.Ability {
			if !card.HasKeyword(k.Value) {
				return false
			}
		}
	} else if w.Number != nil {
		value := w.Number.Value(a, card)
		return w.Compare.Compare(a, value)
	}
	panic("Invalid with")
}

type CardTypeMatch struct {
	Self      bool     `@("NAME")`
	This      bool     `| @("this"|"thas"|"it")`
	Sacrifice bool     `| ( ( @("the" "sacrificed")`
	Target    bool     `| @("target") )?`
	Prefix    []Prefix `@@*`
	Type      CardType `@@? ("card"|"cards")?`
	Without   *Keyword `("without" @@)?`
	With      *With    `@@?`
	Suffix    []Suffix `@@*)!`
}

func (c CardTypeMatch) Match(a *AbilityInstance, o any) bool {
	card, ok := o.(*CardInstance)
	if !ok {
		return false
	}
	if c.Type.Value != "" && !card.HasType(c.Type.Value) {
		return false
	}
	if c.Self && a.Source != card {
		return false
	} else if c.This {
		if !IsIn(card, a.This) {
			return false
		}
	} else if c.Sacrifice {
		if !IsIn(card, a.Sacrificed) {
			return false
		}
	} else if c.Target {
		if !IsIn(card, a.Targeting) {
			return false
		}
	}
	for _, prefix := range c.Prefix {
		if !prefix.Match(a, card) {
			return false
		}
	}
	if c.Without != nil && card.HasKeyword(c.Without.Value) {
		return false
	}
	if c.With != nil && !c.With.Match(a, o) {
		return false
	}
	for _, suffix := range c.Suffix {
		if !suffix.Match(a, card) {
			return false
		}
	}
	return true
}

func (c CardTypeMatch) NrTargets(a *AbilityInstance) int {
	if c.Target {
		return 1
	}
	return -1
}

type CardMatch struct {
	M []CardTypeMatch `@@ (("," @@)* ("and"|"or") @@)?`
}

func (c CardMatch) NrTargets(a *AbilityInstance) int {
	for _, match := range c.M {
		n := match.NrTargets(a)
		if n > 0 {
			return n
		}
	}
	return -1
}

func (c CardMatch) HasTarget() bool {
	for _, match := range c.M {
		if match.Target {
			return true
		}
	}
	return false
}

func (c CardMatch) Match(a *AbilityInstance, o any) bool {
	for _, match := range c.M {
		if match.Match(a, o) {
			return true
		}
	}
	return false
}

type PlayerTypeMatch struct {
	Each       bool `@("each" "player")`
	Self       bool `| @("you")`
	Opponent   bool `| @("your" "opponent")`
	Controller bool `| @("its" "controller")`
}

func (c PlayerTypeMatch) NrTargets(a *AbilityInstance) int {
	return -1
}

func (c PlayerTypeMatch) HasTarget() bool {
	return false
}

func (c PlayerTypeMatch) Match(a *AbilityInstance, o any) bool {
	p, ok := o.(*Player)
	if !ok {
		return false
	}
	if c.Each {
		return true
	} else if c.Self {
		if a.Controller != p {
			return false
		}
	} else if c.Opponent {
		if a.Controller == p {
			return false
		}
	} else if c.Controller {
		for _, c := range a.This {
			if c.(*CardInstance).Controller == p {
				return true
			}
		}
		return false
	}
	return true
}

type PlayerMatch struct {
	M []PlayerTypeMatch `@@ (("," @@)* ("and"|"or") @@)?`
}

func (c PlayerMatch) NrTargets(a *AbilityInstance) int {
	for _, match := range c.M {
		n := match.NrTargets(a)
		if n > 0 {
			return n
		}
	}
	return -1
}

func (c PlayerMatch) HasTarget() bool {
	for _, match := range c.M {
		if match.HasTarget() {
			return true
		}
	}
	return false
}

func (c PlayerMatch) Match(a *AbilityInstance, o any) bool {
	for _, match := range c.M {
		if match.Match(a, o) {
			return true
		}
	}
	return false
}

type AnyMatch struct {
	P         *PlayerMatch `@@`
	C         *CardMatch   `| @@`
	AnyTarget bool         `| @("any" "target")`
}

func (c AnyMatch) NrTargets(a *AbilityInstance) int {
	if c.AnyTarget {
		return 1
	} else if c.P != nil {
		return c.P.NrTargets(a)
	}
	return c.C.NrTargets(a)
}

func (c AnyMatch) HasTarget() bool {
	if c.AnyTarget {
		return true
	} else if c.P != nil {
		return c.P.HasTarget()
	}
	return c.C.HasTarget()
}

func (c AnyMatch) Match(a *AbilityInstance, o any) bool {
	if c.AnyTarget {
		for _, i := range a.Targeting {
			if i == o {
				return true
			}
		}
	} else if c.P != nil {
		return c.P.Match(a, o)
	}
	return c.C.Match(a, o)
}

type ZoneMatch struct {
	Z []Zone `@("deck"|"hand"|"board"|"pile"|"stack")+`
}

func (c *ZoneMatch) Match(ability *AbilityInstance, place Zone, player *Player) bool {
	for _, zone := range c.Z {
		if zone == place {
			return true
		}
	}
	return false
}

func IsIn(o any, l []any) bool {
	for _, i := range l {
		if i == o {
			return true
		}
	}
	return false
}

type Draw struct {
	Number NumberOrX `("draw"|"draws") (@@ | "a") ("card"|"cards")`
}

func (f Draw) HasTarget() bool      { return false }
func (f Draw) IsCost() bool         { return false }
func (f Draw) Do(a *EffectInstance) {}
func (f Draw) Resolve(e *EffectInstance) {
	n := 1
	if f.Number.X || f.Number.Number > 0 {
		n = f.Number.Value(e.Ability)
	}
	for _, p := range e.Subjects {
		p.(*Player).Draw(n)
	}
}

type Token struct {
	Number   NumberOrX  `("create"|"creates") @@`
	Stats    *Stats     `@@?`
	Types    []CardType `@@*`
	Subtypes []SubType  `@@* ("token"|"tokens")`
}

func (f Token) HasTarget() bool      { return false }
func (f Token) IsCost() bool         { return false }
func (f Token) Do(a *EffectInstance) {}
func (f Token) Resolve(e *EffectInstance) {
	types := append(f.Types, CardType{"token"})
	c := &Card{Types: types, Subtypes: f.Subtypes, Stats: f.Stats}
	n := f.Number.Value(e.Ability)
	for _, p := range e.Subjects {
		for i := 0; i < n; i++ {
			p.(*Player).Place(
				NewCardInstance(c, p.(*Player), ZoneBoard),
				ZoneBoard,
				-1,
			)
		}
	}
}

type Destroy struct {
	Value *CardMatch `("destroy"|"destroys") @@`
}

func (f Destroy) HasTarget() bool { return f.Value.HasTarget() }
func (f Destroy) IsCost() bool    { return false }
func (f Destroy) Do(a *EffectInstance) {
	a.Match = f.Value
	a.Zone = &ZoneMatch{[]Zone{ZoneBoard}}
}
func (f Destroy) Resolve(e *EffectInstance) {
	for _, c := range e.matches {
		card := c.(*CardInstance)
		card.Owner.Place(card, ZonePile, -1)
		card.Owner.game.Emit(EventOnDestroy, card)
	}
}

type Add struct {
	Value []CostType `("add"|"adds") @@+`
}

func (f Add) HasTarget() bool      { return false }
func (f Add) IsCost() bool         { return true }
func (f Add) Do(a *EffectInstance) {}
func (f Add) Resolve(e *EffectInstance) {
	for _, p := range e.Subjects {
		for _, c := range f.Value {
			if c.Color != "" {
				p.(*Player).AddEssence(c.Color)
			} else {
				n := c.Number.Value(e.Ability)
				for i := 0; i < n; i++ {
					p.(*Player).AddEssence("u")
				}
			}
		}
	}
}

type GainLife struct {
	Value NumberOrX `("gain"|"gains") @@ "life"`
}

func (f GainLife) HasTarget() bool      { return false }
func (f GainLife) IsCost() bool         { return false }
func (f GainLife) Do(a *EffectInstance) {}
func (f GainLife) Resolve(e *EffectInstance) {
	for _, p := range e.Subjects {
		p.(*Player).GainLife(f.Value.Value(e.Ability))
	}
}

type LoseLife struct {
	Value NumberOrX `("lose"|"loses") @@ "life"`
}

func (f LoseLife) HasTarget() bool      { return false }
func (f LoseLife) IsCost() bool         { return false }
func (f LoseLife) Do(a *EffectInstance) {}
func (f LoseLife) Resolve(e *EffectInstance) {
	for _, p := range e.Subjects {
		p.(*Player).LoseLife(f.Value.Value(e.Ability))
	}
}

type Discard struct {
	Number NumberOrX  `("discard"|"discards") @@`
	Value  *CardMatch `@@?`
}

func (f Discard) HasTarget() bool      { return f.Value.HasTarget() }
func (f Discard) IsCost() bool         { return false }
func (f Discard) Do(a *EffectInstance) { a.Match = f.Value }
func (f Discard) Resolve(e *EffectInstance) {
	n := f.Number.Value(e.Ability)
	for _, p := range e.Subjects {
		choices := []int{}
		if !p.(*Player).prompt("discard", n, e.matches, &choices, nil) {
			return
		}
		for _, i := range choices {
			card := e.matches[i].(*CardInstance)
			card.Owner.Place(card, ZonePile, -1)
			card.Owner.game.Emit(EventOnDiscard, card)
		}
	}
}

type Shuffle struct {
	Objects *CardMatch `("shuffle"|"shuffles") (@@ "into")?`
	Value   *ZoneMatch `@@`
}

func (f Shuffle) HasTarget() bool      { return false }
func (f Shuffle) IsCost() bool         { return false }
func (f Shuffle) Do(a *EffectInstance) { a.Match = f.Objects; a.Zone = f.Value }
func (f Shuffle) Resolve(e *EffectInstance) {
	// TODO: handle multiple zones?
	zone := e.zones[0]
	for _, c := range e.matches {
		card := c.(*CardInstance)
		card.Owner.Place(card, zone, -1)
	}
	for _, p := range e.Subjects {
		p.(*Player).Shuffle(zone)
	}
}

type ExtraTurn struct {
	Number NumberOrX `("take"|"takes") @@ "extra" ("turn"|"turns")`
}

func (f ExtraTurn) HasTarget() bool      { return false }
func (f ExtraTurn) IsCost() bool         { return false }
func (f ExtraTurn) Do(a *EffectInstance) {}
func (f ExtraTurn) Resolve(e *EffectInstance) {
	for _, p := range e.Subjects {
		p.(*Player).turnsAfter += f.Number.Value(e.Ability)
	}
}

type Look struct {
	Number NumberOrX  `("look"|"looks") "at" "the" "top" @@`
	Zone   *ZoneMatch `"cards" "of" @@`
}

func (f Look) HasTarget() bool      { return false }
func (f Look) IsCost() bool         { return false }
func (f Look) Do(a *EffectInstance) { a.Zone = f.Zone }
func (f Look) Resolve(e *EffectInstance) {
	n := f.Number.Value(e.Ability)
	for _, p := range e.Subjects {
		cards := p.(*Player).Query(e.Ability, nil, f.Zone)
		p.(*Player).prompt("look", n, cards[:n], nil, nil)
	}
}

type Put struct {
	Objects     *CardMatch `("put"|"puts") @@`
	Top         bool       `( @"on top of"`
	Bottom      bool       `| @"on the bottom of"`
	Zone        *ZoneMatch `| "into") @@`
	Ordered     bool       `"in any order"?`
	Random      bool       `"in random order"?`
	Deactivated bool       `@("deactivated")?`
}

func (f Put) HasTarget() bool      { return f.Objects.HasTarget() }
func (f Put) IsCost() bool         { return false }
func (f Put) Do(a *EffectInstance) { a.Match = f.Objects; a.Zone = f.Zone }
func (f Put) Resolve(e *EffectInstance) {
	// TODO: handle ordering and top/bottom
	zone := e.zones[0]
	for _, c := range e.matches {
		card := c.(*CardInstance)
		card.Owner.Place(card, zone, -1)
		if f.Deactivated {
			card.Deactivate()
		}
	}
}

type Activate struct {
	Objects *CardMatch `("activate"|"activates") @@`
}

func (f Activate) HasTarget() bool      { return f.Objects.HasTarget() }
func (f Activate) IsCost() bool         { return false }
func (f Activate) Do(a *EffectInstance) { a.Match = f.Objects }
func (f Activate) Resolve(e *EffectInstance) {
	for _, c := range e.matches {
		c.(*CardInstance).Activate()
	}
}

type Deactivate struct {
	Objects *CardMatch `("deactivate"|"deactivates") @@`
}

func (f Deactivate) HasTarget() bool      { return f.Objects.HasTarget() }
func (f Deactivate) IsCost() bool         { return false }
func (f Deactivate) Do(a *EffectInstance) { a.Match = f.Objects }
func (f Deactivate) Resolve(e *EffectInstance) {
	for _, c := range e.matches {
		c.(*CardInstance).Deactivate()
	}
}

type Sacrifice struct {
	Objects *CardMatch `("sacrifice"|"sacrifices") @@`
}

func (f Sacrifice) HasTarget() bool      { return f.Objects.HasTarget() }
func (f Sacrifice) IsCost() bool         { return false }
func (f Sacrifice) Do(a *EffectInstance) { a.Match = f.Objects }
func (f Sacrifice) Resolve(e *EffectInstance) {
	for _, c := range e.matches {
		card := c.(*CardInstance)
		card.Owner.Place(card, ZonePile, -1)
		card.Owner.game.Emit(EventOnSacrifice, card)
	}
}

type PayEssence struct {
	Value []CostType `("spend"|"spends") @@+`
}

func (f PayEssence) HasTarget() bool      { return false }
func (f PayEssence) IsCost() bool         { return false }
func (f PayEssence) Do(a *EffectInstance) {}
func (f PayEssence) Resolve(e *EffectInstance) {
	for _, p := range e.Subjects {
		for _, c := range f.Value {
			c.Pay(e.Ability, p.(*Player))
		}
	}
}

type PayLife struct {
	Value NumberOrX `("spend"|"spends") @@ "life"`
}

func (f PayLife) HasTarget() bool      { return false }
func (f PayLife) IsCost() bool         { return false }
func (f PayLife) Do(a *EffectInstance) {}
func (f PayLife) Resolve(e *EffectInstance) {
	for _, p := range e.Subjects {
		p.(*Player).LoseLife(f.Value.Value(e.Ability))
	}
}

type Damage struct {
	Number  NumberOrX `("deal"|"deals") @@ "damage" "to"`
	Objects Match     `@@`
}

func (f Damage) HasTarget() bool      { return f.Objects.HasTarget() }
func (f Damage) IsCost() bool         { return false }
func (f Damage) Do(a *EffectInstance) { a.Match = f.Objects }
func (f Damage) Resolve(e *EffectInstance) {
	n := f.Number.Value(e.Ability)
	for _, c := range e.Subjects {
		if card, ok := c.(*CardInstance); ok {
			card.TakeDamage(n)
		} else {
			c.(*Player).LoseLife(n)
			c.(*Player).game.Emit(EventOnPlayerDamage, c.(*Player))
		}
	}
}

type Gets struct {
	Pplus  bool      `("get"|"gets") @("+"|"-")`
	Power  NumberOrX `@@ "/"`
	Hplus  bool      `@("+"|"-")`
	Health NumberOrX `@@`
}

func (f Gets) HasTarget() bool      { return false }
func (f Gets) IsCost() bool         { return false }
func (f Gets) Do(a *EffectInstance) {}
func (f Gets) Resolve(e *EffectInstance) {
	// TODO: add duration
	m := Mods{f.Power.Value(e.Ability), f.Health.Value(e.Ability)}
	if !f.Pplus {
		m.Power *= -1
	}
	if !f.Hplus {
		m.Health *= -1
	}
	for _, c := range e.Subjects {
		card := c.(*CardInstance)
		card.modifier = append(card.modifier, m)
	}
}

func (b NumberOrX) Format(f fmt.State, c rune) {
	if b.X {
		fmt.Fprintf(f, "{X}")
	} else {
		fmt.Fprintf(f, "%d", b.Number)
	}
}
func (b CostType) Format(f fmt.State, c rune) {
	if b.Color != "" {
		fmt.Fprintf(f, "{%s}", b.Color)
	} else if b.Activate {
		fmt.Fprintf(f, "{Q}")
	} else if b.Deactivate {
		fmt.Fprintf(f, "{T}")
	} else {
		fmt.Fprintf(f, "{%v}", b.Number)
	}
}
func (b Card) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "%s ", b.Name)
	for _, c := range b.Costs {
		fmt.Fprintf(f, "%+v", c)
	}
	//fmt.Fprintf(f, "%s %+v", b.Name, b.Costs) // \n%s - %s\n%s\n%s", b.Name, b.Costs, b.Types, b.Subtypes, b.Abilities, b.Stats)
}
func (b CardInstance) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "%v", b.Card)
}
func (b Stats) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "%v/%v", b.Power, b.Health)
}
func (a AbilityInstance) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "%v", a.Source)
}
