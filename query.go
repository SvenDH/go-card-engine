package main

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
		if card.card.Stats.Power.Number != c.Stats.Power.Number ||
			card.card.Stats.Health.Number != c.Stats.Health.Number {
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
	Ability []Keyword    `"with" ( (@@ (("," @@)* "and" @@)?)`
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
	Each     bool `@("each" "player")`
	Self     bool `| @("you")`
	Opponent bool `| @("your" "opponent")`
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
			if c.(*CardInstance).controller == p {
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