package main

import "fmt"

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
	Match   *CardMatch    `@@?`
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
	Match   *PlayerMatch   `(@@`
	Optional bool          `@("may")? )?`
	Effects []PlayerEffect `@@ ((",") ("then"|"and")? @@)*`
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
	return card.controller.CanPay(card, f.Cost)
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

func (a AbilityInstance) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "%v", a.Source)
}
