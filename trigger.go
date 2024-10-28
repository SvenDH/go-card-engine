package main

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
		if a.Controller != a.Source.owner.game.turn.player {
			return false
		}
	} else if c.NotYourTurn {
		if a.Controller == a.Source.owner.game.turn.player {
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
		t.Do(p, a)
	}
}
