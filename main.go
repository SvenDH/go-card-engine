package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"os"
	"strconv"
	"strings"
)

//go:embed cards.txt
var cardText []byte

type CliCommand struct {
	*bufio.Reader
}

func (c *CliCommand) read(max int) (int, error) {
	var choice int
	for {
		char, _, err := c.Reader.ReadRune()
		if err != nil {
			return 0, err
		}
		if char == '\n' || char == '\r' {
			continue
		} else if char == 's' {
			return -1, nil
		}
		choice, err = strconv.Atoi(string(char))
		if err != nil || choice < 1 && choice > max {
			fmt.Printf("invalid choice: %v, choose number between 1 and %d or \"s\" (skip)\n", choice, max)
			continue
		}
		return choice-1, nil
	}
}

func (c *CliCommand) Card(player *Player, options []any) (int, error) {
	if len(options) == 0 { return -1, nil }
	fmt.Println("Choose card", options)
	return c.read(len(options))
}

func (c *CliCommand) Mode(player *Player, options []any) (int, error) {
	if len(options) == 0 { return -1, nil }
	fmt.Println("Choose mode", options)
	return c.read(len(options))
}

func (c *CliCommand) Field(player *Player, options []any) (int, error) {
	if len(options) == 0 { return -1, nil }
	fmt.Println("Choose field", options)
	return c.read(len(options))
}

func (c *CliCommand) Ability(player *Player, options []any) (int, error) {
	if len(options) == 0 { return -1, nil }
	fmt.Println("Choose ability", options)
	return c.read(len(options))
}

func (c *CliCommand) Target(player *Player, options []any, num int) ([]int, error) {
	if len(options) == 0 { return nil, nil }
	fmt.Println("Choose target", options)
	choices := []int{}
	for i := 0; i < num; i++ {
		c, err := c.read(len(options))
		if err != nil {
			return nil, err
		}
		choices = append(choices, c)
		options = append(options[:c], options[c+1:]...)
	}
	return choices, nil
}

func (c *CliCommand) Discard(player *Player, options []any, num int) ([]int, error) {
	if len(options) == 0 || num == 0 { return nil, nil }
	fmt.Println("Choose", num, "cards to discard", options)
	choices := []int{}
	for i := 0; i < num; i++ {
		c, err := c.read(len(options))
		if err != nil {
			return nil, err
		}
		choices = append(choices, c)
		options = append(options[:c], options[c+1:]...)
	}
	return choices, nil
}

func main() {
	parser := NewCardParser()

	cards := []*Card{}
	for _, txt := range strings.Split(string(cardText), "\n\n") {
		card, err := parser.Parse(txt)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println(card)
		cards = append(cards, card)
	}
	
	cli := &CliCommand{bufio.NewReader(os.Stdin)}
	p1 := NewPlayer(cli, cards...)
	game := NewGame(p1)
	for turn := range game.Iter() {
		fmt.Println("Turn", turn.turn)
		for phase := range turn.Iter() {
			fmt.Println("Phase", phaseToString(phase.phase))
			for player := range phase.Iter() {
				fmt.Printf("Hand: %v\n", player.hand.cards)
				fmt.Printf("Player %d has priority\n", player.nr)
				player.Iter()
				fmt.Println("Essence", player.essence)
				fmt.Print("Board: |")
				for _, card := range player.board.slots {
					if card != nil {
						fmt.Print(card.card.Name)
					} else {
						fmt.Print("Empty")
					}
					fmt.Print("|")
				}
				fmt.Printf("\nStack: %v\n", game.stack)
			}
		}
	}
}

func phaseToString(phase PhaseType) string {
	switch phase {
	case PhaseStart:
		return "start"
	case PhaseDraw:
		return "draw"
	case PhasePlay:
		return "play"
	case PhaseEnd:
		return "end"
	}
	return "unknown"
}
