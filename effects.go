package main

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
		card.owner.Place(card, ZonePile, -1)
		card.owner.game.Emit(EventOnDestroy, card)
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
		if !p.(*Player).Prompt("discard", n, e.matches, &choices) {
			return
		}
		for _, i := range choices {
			card := e.matches[i].(*CardInstance)
			card.owner.Place(card, ZonePile, -1)
			card.owner.game.Emit(EventOnDiscard, card)
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
		card.owner.Place(card, zone, -1)
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
		p.(*Player).Prompt("look", n, cards[:n], nil)
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
		card.owner.Place(card, zone, -1)
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
		card.owner.Place(card, ZonePile, -1)
		card.owner.game.Emit(EventOnSacrifice, card)
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
