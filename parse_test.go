package main

import (
	"reflect"
	"testing"
)

var TargetCard = CardMatch{[]CardTypeMatch{{Target: true}}}

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
							},
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
						},
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
						},
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
						}},
					},
				},
				Stats: &Stats{One, One},
			},
		},
	}

	for _, test := range tests {
		card, err := parser.Parse(test.text)
		if err != nil {
			t.Errorf("Error parsing card: %v", err)
		}
		if !reflect.DeepEqual(card, test.card) {
			t.Errorf("Expected %v, got %v", test.card, card)
		}
	}
}