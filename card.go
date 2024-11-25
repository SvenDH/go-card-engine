package main

import "fmt"

type NumberOrX struct {
	Number int  `@Int`
	A 	   bool `| @("a"|"an")`
	X      bool `| @"X"`
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
}

type CardInstance struct {
	card       *Card
	activated  bool
	flipped    bool
	zone       Zone
	index      int
	owner      *Player
	controller *Player
	stats      *Stats
	modifier   []Mods
}

func NewCardInstance(card *Card, owner *Player, zone Zone) *CardInstance {
	c := &CardInstance{
		card:       card,
		owner:      owner,
		controller: owner,
		zone:       zone,
		modifier:   []Mods{},
	}
	if card.Stats != nil {
		c.stats = &Stats{card.Stats.Power, card.Stats.Health}
	}
	return c
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
	c.owner.game.Emit(EventOnDamage, c)
	if c.stats.Health.Number <= 0 {
		// TODO: check invurnability
		c.controller.Remove(c)
		c.owner.Place(c, ZonePile, -1)
		c.owner.game.Emit(EventOnDestroy, c)
	}
}

func (c *CardInstance) Activate() {
	c.activated = true
	c.owner.game.Emit(EventOnActivate, c)
}

func (c *CardInstance) Deactivate() {
	c.activated = false
	c.owner.game.Emit(EventOnDeactivate, c)
}

func (c *CardInstance) Do(a *Activated) *AbilityInstance {
	player := c.owner.game.turn.phase.priority
	player.Pay(c, a.Cost)
	return a.Do(player, c)
}

func (c *CardInstance) Play(index int) {
	c.activated = true
	player := c.owner.game.turn.phase.priority
	player.Place(c, ZoneBoard, index)
	player.game.Emit(EventOnPlay, c)
}

func (c *CardInstance) Cast(index int) *AbilityInstance {
	player := c.owner.game.turn.phase.priority
	player.Remove(c)
	player.Pay(c, c.GetCosts())
	return &AbilityInstance{Source: c, Controller: player, Field: index}
}

func (c *CardInstance) Trigger(event *Event) {
	for _, t := range c.GetTriggeredAbilities() {
		a := t.Do(c.controller, c)
		if a != nil {
			c.controller.game.Play(a)
		}
	}
}

func (c *CardInstance) CanDo() bool {
	for _, a := range c.GetActivatedAbilities() {
		if a.CanDo(c) && (a.IsCost() || !c.owner.game.IsReaction()) {
			return true
		}
	}
	return false
}

func (c *CardInstance) CanReact() bool {
	// TODO: check if card can react or instant
	return c.owner.game.turn.player == c.owner && !c.owner.game.IsReaction()
}

func (c *CardInstance) CanPlay() bool {
	// TODO: check castable from other locations
	if c.zone != ZoneHand || !c.CanReact() {
		return false
	}
	return c.owner.game.turn.phase.priority.CanPay(c, c.GetCosts())
}

func (c *CardInstance) CanSource() bool {
	if c.zone != ZoneHand || !c.CanReact() {
		return false
	}
	t := c.owner.game.turn
	return t.sourcesPlayed >= t.phase.priority.SourcesPerTurn()
}

func (c *CardInstance) GetCosts() []AbilityCost {
	costs := []AbilityCost{}
	for _, ct := range c.card.Costs {
		costs = append(costs, AbilityCost{Cost: &ct})
	}
	return costs
}

func (c *CardInstance) HasKeyword(k string) bool {
	for _, a := range c.card.Abilities {
		if keyword, ok := a.(*Keyword); ok && keyword.Value == k {
			return true
		}
	}
	return false
}

func (c *CardInstance) HasType(t string) bool {
	for _, ct := range c.card.Types {
		if ct.Value == t {
			return true
		}
	}
	return false
}

func (c *CardInstance) HasSubType(t string) bool {
	for _, ct := range c.card.Subtypes {
		if ct.Value == t {
			return true
		}
	}
	return false
}

func (c *CardInstance) HasColor(t string) bool {
	for _, ct := range c.card.Costs {
		if ct.Color == t {
			return true
		}
	}
	return false
}

func (c *CardInstance) GetActivatedAbilities() []*Activated {
	abilities := []*Activated{}
	for _, a := range c.card.Abilities {
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
	for _, a := range c.card.Abilities {
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
		for _, p := range card.controller.game.players {
			if p != card.controller {
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
		[]AbilityCost{{Cost:&CostType{Deactivate: true}}},
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
	fmt.Fprintf(f, "%s %+v\n%s - %s\n%s\n%s", b.Name, b.Costs, b.Types, b.Subtypes, b.Abilities, b.Stats)
}
func (b CardInstance) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "%v", b.card)
}
func (b Stats) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "%v/%v", b.Power, b.Health)
}