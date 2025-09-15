package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"os"
	"strconv"
	"strings"

	cards "github.com/SvenDH/go-card-engine/game"
)

////go:embed ../cards.txt
var cardText []byte

var parser = cards.NewCardParser()

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
}=+

func (c *CliCommand) Card(player *cards.Player, options []any) (int, error) {
	if len(options) == 0 { return -1, nil }
	fmt.Println("Choose card", options)
	return c.read(len(options))
}

func (c *CliCommand) Mode(player *cards.Player, options []any) (int, error) {
	if len(options) == 0 { return -1, nil }
	fmt.Println("Choose mode", options)
	return c.read(len(options))
}

func (c *CliCommand) Field(player *cards.Player, options []any) (int, error) {
	if len(options) == 0 { return -1, nil }
	fmt.Println("Choose field", options)
	return c.read(len(options))
}

func (c *CliCommand) Ability(player *cards.Player, options []any) (int, error) {
	if len(options) == 0 { return -1, nil }
	fmt.Println("Choose ability", options)
	return c.read(len(options))
}

func (c *CliCommand) Target(player *cards.Player, options []any, num int) ([]int, error) {
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

func (c *CliCommand) Discard(player *cards.Player, options []any, num int) ([]int, error) {
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

func TestCli() {

	cards := []*cards.Card{}
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
	p1 := cards.NewPlayer(cli, cards...)
	game := cards.NewGame(p1)
	game.Run()
}
