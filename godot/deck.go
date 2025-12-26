package godot

import (
	"fmt"
	"os"
	"strings"

	"github.com/SvenDH/go-card-engine/engine"
)

func loadDeck() ([]*engine.Card, error) {
	parser := engine.NewCardParser()
	data, err := loadCardBytes()
	if err != nil {
		return nil, err
	}
	blocks := strings.Split(strings.TrimSpace(string(data)), "\n\n")
	deck := []*engine.Card{}
	for _, block := range blocks {
		if strings.TrimSpace(block) == "" {
			continue
		}
		card, err := parser.Parse(block, true)
		if err != nil {
			continue
		}
		deck = append(deck, card)
	}
	if len(deck) == 0 {
		return nil, fmt.Errorf("no cards parsed from assets/cards.txt")
	}
	return deck, nil
}

// loadCardBytes tries common locations so the game works when run from the graphics project dir.
func loadCardBytes() ([]byte, error) {
	paths := []string{
		"assets/cards.txt",       // running from repo root
		"../assets/cards.txt",    // running from graphics/ subproject
		"../../assets/cards.txt", // running from graphics/.godot or similar
	}
	for _, p := range paths {
		if data, err := os.ReadFile(p); err == nil {
			return data, nil
		}
	}
	return nil, fmt.Errorf("unable to locate assets/cards.txt (tried %v)", paths)
}
