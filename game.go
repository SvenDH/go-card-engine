package main

import (
	"iter"
	"math/rand"
)

type PhaseType int8
type EventType int8
type Zone int8

const (
	PhaseStart PhaseType = iota
	PhaseDraw
	PhasePlay
	PhaseEnd
)
const (
	NoEvent EventType = iota
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
)
const (
	ZoneAny Zone = iota
	ZoneDeck
	ZoneHand
	ZoneBoard
	ZonePile
	ZoneStack
)

const boardSize = 5
const startCards = 3
const startLife = 20

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

type Game struct {
	players      []*Player
	stack        Stack
	turn         *Turn
	currentEvent EventType
	resolving    *AbilityInstance
}

type Turn struct {
	game          *Game
	player        *Player
	phase         *Phase
	turn          int
	sourcesPlayed int
}

type Phase struct {
	turn     *Turn
	priority *Player
	phase    PhaseType
}

type Event struct {
	Event   EventType
	Source  any
	Subject any
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

func NewGame(players ...*Player) *Game {
	if len(players) == 0 {
		panic("No players")
	}
	g := &Game{
		players: players,
		stack:   Stack{cards: []*AbilityInstance{}},
	}
	for _, player := range g.players {
		player.game = g
	}
	return g
}

func (g *Game) Emit(event EventType, subject any) {
	g.currentEvent = event
	e := &Event{event, g.resolving, subject}
	for _, player := range g.players {
		// TODO: Check cards in other zones
		for _, card := range player.board.slots {
			if card != nil {
				card.Trigger(e)
			}
		}
	}
	g.currentEvent = NoEvent
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
		if !a.Controller.Prompt("target", 1, found, &targeted) {
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
			if !a.Controller.Prompt("target", 1, found, &targeted) {
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

func (g *Game) Iter() iter.Seq[*Turn] {
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
	return func(yield func(*Turn) bool) {
		// TODO: Check for game over
		for {
			g.turn = &Turn{g, p, nil, turn, 0}
			if !yield(g.turn) {
				return
			}
			turn += 1
			if p.turnsAfter > 0 {
				p.turnsAfter -= 1
			} else {
				p = g.nextPlayer(p)
			}
		}
	}
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
