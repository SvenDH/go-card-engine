package main

import (
	"reflect"
	"testing"
)

var TargetCard = CardMatch{[]CardTypeMatch{{Target: true}}}

var X = NumberOrX{X: true}
var Zero = NumberOrX{}
var One = NumberOrX{Number: 1}
var Two = NumberOrX{Number: 2}
var Three = NumberOrX{Number: 3}

func TestCardParser(t *testing.T) {
	parser := NewCardParser()

	tests := []struct{
		text string
		card *Card
	}{
		{
			text: `Test {s}{1}
			Unit`,
			card: &Card{
				Name: "Test",
				Types: []CardType{{"unit"}},
				Costs: []CostType{
					{Color: "s"},
					{Number: NumberOrX{Number: 1}},
				},
			},
		},
		{
			text: `Soldier {s}
			Unit - Human
			1/1`,
			card: &Card{
				Name: "Soldier",
				Types: []CardType{{"unit"}},
				Subtypes: []SubType{{"human"}},
				Costs: []CostType{{Color: "s"}},
				Stats: &Stats{One, One},
			},
		},
		{
			text: `Knight {s}
			Unit
			Siege
			2/2`,
			card: &Card{
				Name: "Knight",
				Types: []CardType{{"unit"}},
				Costs: []CostType{{Color: "s"}},
				Abilities: []Ability{
					Keyword{"siege"},
				},
				Stats: &Stats{Two, Two},
			},
		},
		{
			text: `Land
			Source
			{t}: Add {s}.
			`,
			card: &Card{
				Name: "Land",
				Types: []CardType{{"source"}},
				Abilities: []Ability{
					Activated{
						Cost: []AbilityCost{
							{Cost:&CostType{Deactivate: true}},
						},
						Effect: Composed{
							[]Effect{
								PlayerSubjectAbility{
									nil, false,
									[]PlayerEffect{
										Add{
											[]CostType{
												{Color: "s"},
											},
										},
									},
								},
							},
							"",
						},
					},
				},
			},
		},
		{
			text: `Super Soldier {s}
			Unit
			Fly, Siege
			{q}: Destroy target card.
			1/1`,
			card: &Card{
				Name: "Super Soldier",
				Types: []CardType{{"unit"}},
				Costs: []CostType{{Color: "s"}},
				Abilities: []Ability{
					Keyword{"fly"},
					Keyword{"siege"},
					Activated{
						Cost: []AbilityCost{
							{Cost: &CostType{Activate: true}},
						},
						Effect: Composed{
							[]Effect{
								PlayerSubjectAbility{
									nil, false,
									[]PlayerEffect{
										Destroy{&TargetCard},
									},
								},
							}, "",
						},
					},
				},
				Stats: &Stats{One, One},
			},
		},
		{
			text: `Wizard {w}{2}
			Unit
			{t}: Wizard deals 1 damage to any target.
			1/2`,
			card: &Card{
				Name: "Wizard",
				Types: []CardType{{"unit"}},
				Costs: []CostType{
					{Color: "w"},
					{Number: NumberOrX{Number: 2}},
				},
				Abilities: []Ability{
					Activated{
						Cost: []AbilityCost{
							{Cost: &CostType{Deactivate: true}},
						},
						Effect: Composed{
							[]Effect{
								CardSubjectAbility{
									&CardMatch{[]CardTypeMatch{{Self: true}}},
									[]CardEffect{
										Damage{
											NumberOrX{Number: 1},
											AnyMatch{AnyTarget: true},
										},
									},
								},
							},
							"",
						},
					},
				},
				Stats: &Stats{One, Two},
			},
		},
		{
			text: `Test {s}{o}{2}
			Unit - Beast Human
			Fly, Siege, Poison
			Draw 4 cards.
			Destroy non-cup units.
			3/2`,
			card: &Card{
				Name: "Test",
				Types: []CardType{{"unit"}},
				Subtypes: []SubType{{"beast"}, {"human"}},
				Costs: []CostType{
					{Color: "s"},
					{Color: "o"},
					{Number: Two},
				},
				Abilities: []Ability{
					Keyword{"fly"},
					Keyword{"siege"},
					Keyword{"poison"},
					Composed{
						[]Effect{
							PlayerSubjectAbility{
								nil, false,
								[]PlayerEffect{
									Draw{NumberOrX{Number: 4}},
								},
							},
						}, "",
					},
					Composed{
						[]Effect{
							PlayerSubjectAbility{
								nil, false,
								[]PlayerEffect{
									Destroy{
										&CardMatch{
											[]CardTypeMatch{
												{
													Prefix:[]Prefix{
														{NonColor:Color{"cup"}},
														{Type:CardType{"units"}},
													},
													
												},
											},
										},
									},
								},
							},
						}, "",
					},

				},
				Stats: &Stats{Three, Two},
			},
		},
		{
			text: `Flashcaster {1}{c}
			Unit
			When Flashcaster is put on the board, draw a card.
			1/1`,
			card: &Card{
				Name: "Flashcaster",
				Types: []CardType{{"unit"}},
				Costs: []CostType{
					{Number: NumberOrX{Number: 1}},
					{Color: "c"},
				},
				Abilities: []Ability{
					Triggered{
						Trigger{
							Condition: &Condition{
								CardCondition: &CardCondition{
									Cards: CardMatch{[]CardTypeMatch{{Self: true}}},
									Enters: true,
								},
							},
						},
						Composed{[]Effect{
							PlayerSubjectAbility{
								nil, false,
								[]PlayerEffect{
									Draw{NumberOrX{A: true}},
								},
							},
						}, ""},
						"",
					},
				},
				Stats: &Stats{One, One},
			},
		},
	}

	for _, test := range tests {
		card, err := parser.Parse(test.text, false)
		if err != nil {
			t.Errorf("Error parsing card: %v", err)
		}
		if !reflect.DeepEqual(card, test.card) {
			t.Errorf("Expected %v, got %v", test.card, card)
		}
	}
}

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
			} else if p1.hand.cards[0].Card.Name != "card2" {
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
			if game.stack.cards[0].Source.Card.Name != "card2" {
				t.Fatalf("Stack does not contain card from hand")
			}
			a2 := game.stack.Pop()
			if !reflect.DeepEqual(a1, a2) {
				t.Fatalf("Ability not the cast card")
			}
			a2.Resolve()
			if p1.board.slots[1].Card.Name != "card2" {
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
					"",
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