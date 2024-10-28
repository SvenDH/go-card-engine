package main

import (
	"math/rand"
)

type CommandI interface {
	Card(*Player, []any) (int, error)
	Mode(*Player, []any) (int, error)
	Field(*Player, []any) (int, error)
	Ability(*Player, []any) (int, error)
	Target(*Player, []any, int) ([]int, error)
	Discard(*Player, []any, int) ([]int, error)
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

func (p *Player) Prompt(
	cmd string,
	num int,
	choices []any,
	selected *[]int,
) bool {
	if p.cmdi != nil {
		if cmd == "card" {
			(*selected)[0], _ = p.cmdi.Card(p, choices)
		} else if cmd == "mode" {
			(*selected)[0], _ = p.cmdi.Mode(p, choices)
		} else if cmd == "field" {
			(*selected)[0], _ = p.cmdi.Field(p, choices)
		} else if cmd == "ability" {
			(*selected)[0], _ = p.cmdi.Ability(p, choices)
		} else if cmd == "target" {
			arr, _ := p.cmdi.Target(p, choices, num)
			*selected = append(*selected, arr...)
		} else if cmd == "discard" {
			arr, _ := p.cmdi.Discard(p, choices, num)
			*selected = append(*selected, arr...)
		}
	}
	return true
}

func (p *Player) Do() bool {
	selected := []int{0}
	choices := p.GetPlayableCards()
	if !p.Prompt("card", 1, choices, &selected) ||
		selected[0] < 0 || selected[0] >= len(choices) {
		return false
	}
	card := choices[selected[0]].(*CardInstance)
	if card.CanDo() {
		activatable := card.GetActivatable()
		if !p.Prompt("ability", 1, activatable, &selected) {
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
			return true
		}
	}
	flipped := card.CanSource()
	if flipped && card.CanPlay() {
		if !p.Prompt("source", 1, nil, &selected) || selected[0] < 0 {
			return false
		}
		flipped = selected[0] == 1
	}
	fields := p.freeFields(card)
	if !p.Prompt("field", 1, fields, &selected) || selected[0] < 0 {
		return false
	}
	card.flipped = flipped
	if flipped {
		p.game.turn.sourcesPlayed += 1
		card.Play(fields[selected[0]].(int))
	} else {
		p.game.Play(card.Cast(fields[selected[0]].(int)))
	}
	return true
}

func (p *Player) Iter() {
	for p.Do() {
	}
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
	card.controller.Remove(card)
	switch zone {
	case ZoneDeck:
		p.deck.Insert(card, index)
	case ZonePile:
		p.pile.Insert(card, index)
	case ZoneHand:
		p.hand.Add(card)
	case ZoneBoard:
		p.board.Insert(card, index)
		card.controller = p
		p.game.Emit(EventOnEnterBoard, card)
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
			if card.Source.owner == p && (obj == nil || obj.Match(a, card.Source)) {
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
