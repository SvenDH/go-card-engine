package main

import (
	"reflect"
	"testing"
)

var X = NumberOrX{X: true}
var Zero = NumberOrX{}
var One = NumberOrX{Number: 1}
var Two = NumberOrX{Number: 2}
var Three = NumberOrX{Number: 3}

func newGame(players []*Player) *Game {
	g := &Game{
		players: players,
		stack:   Stack{cards: []*AbilityInstance{}},
	}
	for _, player := range g.players {
		player.game = g
	}
	return g
}

func newPlayer(
	board []*Card,
	deck []*Card,
	hand []*Card,
	pile []*Card,
) *Player {
	p := &Player{
		life:    10,
		deck:    Pile{cards: []*CardInstance{}},
		hand:    Pile{cards: []*CardInstance{}},
		pile:    Pile{cards: []*CardInstance{}},
		board:   Board{slots: make([]*CardInstance, boardSize)},
		essence: []string{},
	}
	for _, card := range deck {
		p.deck.Add(NewCardInstance(card, p, ZoneDeck))
	}
	for _, card := range hand {
		p.hand.Add(NewCardInstance(card, p, ZoneHand))
	}
	for _, card := range pile {
		p.hand.Add(NewCardInstance(card, p, ZoneHand))
	}
	for i, card := range board {
		if card != nil {
			p.board.Insert(NewCardInstance(card, p, ZoneBoard), i)
		}
	}
	return p
}

func newSimpleUnit(name string) *Card {
	return &Card{
		Name:  name,
		Types: []CardType{{"unit"}},
		Stats: &Stats{One, One},
	}
}

func TestGamePhases(t *testing.T) {
	p1 := newPlayer(
		[]*Card{newSimpleUnit("card1")},
		[]*Card{newSimpleUnit("card2"), newSimpleUnit("card3")},
		[]*Card{},
		[]*Card{},
	)
	i := 1
	game := newGame([]*Player{p1})
	game.turn = &Turn{game, p1, nil, 0, 0}
	game.turn.Iter()(func(phase *Phase) bool {
		switch phase.phase {
		case PhaseStart:
			if i != 1 {
				t.Fatalf("Phase draw is not first phase: %v", phase)
			}
		case PhaseDraw:
			if i != 2 {
				t.Fatalf("Phase draw is not second phase: %v", phase)
			}
			if len(p1.deck.cards) != 1 {
				t.Fatalf("Deck size not 1")
			} else if len(p1.hand.cards) != 1 {
				t.Fatalf("Hand does not contain card")
			} else if p1.hand.cards[0].card.Name != "card2" {
				t.Fatalf("Hand does not contain top card from deck")
			}
		case PhasePlay:
			if i != 3 {
				t.Fatalf("Phase play is not third phase: %v", phase)
			}
			a1 := p1.hand.cards[0].Cast(1)
			game.Play(a1)
			if len(game.stack.cards) != 1 {
				t.Fatalf("Stack size not 1")
			}
			if game.stack.cards[0].Source.card.Name != "card2" {
				t.Fatalf("Stack does not contain card from hand")
			}
			a2 := game.stack.Pop()
			if !reflect.DeepEqual(a1, a2) {
				t.Fatalf("Ability not the cast card")
			}
			a2.Resolve()
			if p1.board.slots[1].card.Name != "card2" {
				t.Fatalf("Board does not contain card from hand")
			}
		case PhaseEnd:
			if i != 4 {
				t.Fatalf("Phase end is not fourth phase: %v", phase)
			}
			return false
		}
		i++
		return true
	})
}

func TestTrigger(t *testing.T) {
	card := &Card{
		Name: "Flashcaster",
		Types: []CardType{{"unit"}},
		Costs: []CostType{
			{Number: NumberOrX{Number: 1}},
			{Color: "r"},
		},
		Abilities: []Ability{
			Triggered{
				Trigger: Trigger{
					Condition: &Condition{
						CardCondition: &CardCondition{
							Cards: CardMatch{M: []CardTypeMatch{{Self: true}}},
							Enters: true,
						},
					},
				},
				Effect: Composed{
					[]Effect{
						PlayerSubjectAbility{
							nil, false,
							[]PlayerEffect{
								Draw{NumberOrX{}},
							},
						},
					},
				},
			},
		},
		Stats: &Stats{One, One},
	}
	p1 := newPlayer(
		[]*Card{},
		[]*Card{newSimpleUnit("card1")},
		[]*Card{card},
		[]*Card{},
	)
	game := newGame([]*Player{p1})
	game.turn = &Turn{game, p1, nil, 0, 0}
	game.turn.phase = &Phase{game.turn, p1, PhasePlay}

	a1 := p1.hand.cards[0].Cast(1)
	game.Play(a1)

	if len(game.stack.cards) != 1 {
		t.Fatalf("Stack size not 1")
	}
	a2 := game.stack.Pop()
	if !reflect.DeepEqual(a1, a2) {
		t.Fatalf("Ability not the cast card")
	}
	a2.Resolve()
	if len(game.stack.cards) != 1 {
		t.Fatalf("Stack size not 1")
	}
	game.stack.Pop().Resolve()
	if len(p1.hand.cards) != 1 {
		t.Fatalf("Hand size not 1")
	}
}