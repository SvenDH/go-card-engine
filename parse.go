package main

import (
	"strings"
	"fmt"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type CardParser struct {
	parser *participle.Parser[Card]
}

func NewCardParser() *CardParser {
	parser := participle.MustBuild[Card](
		participle.Lexer(lexer.MustSimple([]lexer.SimpleRule{
			{"whitespace", `[\s]+`},
			{"Ident", `[a-zA-Z]\w*`},
			{"Punct", `[-+,{}/:.]`},
			{"Int", `\d+`},
		})),
		//participle.UseLookahead(2),
		participle.Union[Ability](Keyword{}, Composed{}, Activated{}, Triggered{}),
		participle.Union[Effect](
			PlayerSubjectAbility{},
			CardSubjectAbility{},
		),
		participle.Union[Match](
			AnyMatch{},
			PlayerMatch{},
			CardMatch{},
		),
		participle.Union[PlayerEffect](
			Draw{},
			Token{},
			Destroy{},
			Add{},
			GainLife{},
			LoseLife{},
			Discard{},
			Shuffle{},
			ExtraTurn{},
			Look{},
			Put{},
			Activate{},
			Deactivate{},
			Sacrifice{},
			PayEssence{},
			PayLife{},
		),
		participle.Union[CardEffect](
			Damage{},
			Gets{},
		),
		participle.UseLookahead(3),
	)
	fmt.Printf("parser: %s\n", parser.String())
	return &CardParser{parser}
}

func (p *CardParser) Parse(txt string) (*Card, error) {
	name := strings.TrimSpace(strings.SplitN(strings.SplitN(strings.TrimSpace(txt), "\n", 2)[0], "{", 2)[0])
	txt = strings.ReplaceAll(strings.ToLower(txt), strings.ToLower(name), "NAME")
	card, err := p.parser.ParseString("", txt)
	if err != nil {
		return nil, err
	}
	card.Name = name
	return card, nil
}
