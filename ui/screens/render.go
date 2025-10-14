package screens

import (
	"fmt"

	"github.com/SvenDH/go-card-engine/ui"
)

// renderDeckCount renders the deck count (currently unused as cards are created on-demand)
func (e *CardGame) renderDeckCount(screen *ui.TileMap) *ui.TileMap {
	// No longer showing deck count since cards are created on-demand
	return screen
}

// renderLifeTotals renders player and enemy life totals
func (e *CardGame) renderLifeTotals(screen *ui.TileMap) *ui.TileMap {
	playerLifeStr := fmt.Sprintf("Life: %d", e.playerLife)
	playerLifeStyle := ui.NewStyle().
		Foreground(ui.Colors["light-beige"]).
		Background(ui.Colors["dark"])
	playerLifeText := ui.Text(playerLifeStr)
	screen = screen.Overlay(playerLifeStyle.Render(playerLifeText.View()), 2, e.H-2)

	enemyLifeStr := fmt.Sprintf("Enemy: %d", e.enemyLife)
	enemyLifeStyle := ui.NewStyle().
		Foreground(ui.Colors["light-beige"]).
		Background(ui.Colors["dark"])
	enemyLifeText := ui.Text(enemyLifeStr)
	return screen.Overlay(enemyLifeStyle.Render(enemyLifeText.View()), e.W-12, 2)
}

// renderCards renders all cards in the correct z-order
func (e *CardGame) renderCards(screen *ui.TileMap) *ui.TileMap {
	// Collect all card collections for rendering
	allCardCollections := [][]ui.Model{
		e.enemyResources.Cards(),
		e.enemyLanes.(*Lanes).cards,
		e.playerLanes.(*Lanes).cards,
		e.playerResources.Cards(),
	}

	stackCard := e.stack.GetCard()
	// Render all non-dragged cards (excluding stack card)
	for _, cards := range allCardCollections {
		for _, card := range cards {
			if card != nil {
				c := card.(*Card)
				if !c.dragging && c != stackCard {
					screen = screen.Overlay(c.View(), c.X, c.Y)
				}
			}
		}
	}

	// Draw cards in hand - non-hovered first, save hovered for last (excluding stack card)
	var hoveredCard *Card
	for _, c := range e.hand.Cards() {
		card := c.(*Card)
		if card == stackCard {
			continue
		}
		if card.hovered || card.dragging {
			hoveredCard = card
		} else {
			screen = screen.Overlay(card.View(), card.X, card.GetDrawY())
		}
	}

	// Draw all dragged cards on top (excluding stack card)
	for _, cards := range allCardCollections {
		for _, card := range cards {
			if card != nil {
				c := card.(*Card)
				if c.dragging && c != stackCard {
					screen = screen.Overlay(c.View(), c.X, c.Y)
				}
			}
		}
	}

	// Draw hovered/dragged hand card on top of everything
	if hoveredCard != nil {
		screen = screen.Overlay(hoveredCard.View(), hoveredCard.X, hoveredCard.GetDrawY())
	}

	return screen
}

// renderStack renders the stack zone with the card on it
func (e *CardGame) renderStack(screen *ui.TileMap) *ui.TileMap {
	if !e.stack.HasCard() {
		return screen
	}
	stackCard := e.stack.GetCard()

	stackX := e.W/2 - 6
	stackY := e.H/2 - 8

	// Render stack zone background (create a fresh tilemap to avoid recursion)
	stackBg := ui.NewTileMap(12, 16, nil)
	stackStyle := ui.NewStyle().
		Border(ui.Borders["roundheavy"]).
		BorderForeground(ui.Colors["brown"]).
		BorderBackground(ui.Colors["dark"]).
		Background(ui.Colors["dark"])
	screen = screen.Overlay(stackStyle.Render(stackBg), stackX, stackY)

	// Draw the card centered in the stack (account for border)
	// Double-check card isn't nil (could have been cleared by concurrent event)
	if stackCard == nil {
		return screen
	}
	cardX := stackX + 1
	cardY := stackY + 1
	screen = screen.Overlay(stackCard.View(), cardX, cardY)

	// Draw "Stack" label above
	stackLabel := ui.Text("Stack")
	labelStyle := ui.NewStyle().
		Foreground(ui.Colors["light-beige"]).
		Background(ui.Colors["dark"])
	return screen.Overlay(labelStyle.Render(stackLabel.View()), stackX+2, stackY-2)
}

// renderPhaseLabel renders the current phase and player turn indicator
func (e *CardGame) renderPhaseLabel(screen *ui.TileMap) *ui.TileMap {
	phaseText := fmt.Sprintf("%s Phase %s", e.currentPhase, e.currentPlayer)
	labelStyle := ui.NewStyle().
		Border(ui.Borders["round"]).
		BorderForeground(ui.Colors["dark-brown"]).
		BorderBackground(ui.Colors["dark"]).
		Foreground(ui.Colors["light-beige"]).
		Background(ui.Colors["dark"]).
		Padding(0, 1).
		AlignHorizontal(ui.Center)
	e.phaseLabel.M = labelStyle.Render(ui.Text(phaseText).View())
	return screen.Overlay(e.phaseLabel.View(), 2, 2)
}

// renderAbilityMenu renders the ability selection menu
func (e *CardGame) renderAbilityMenu(screen *ui.TileMap) *ui.TileMap {
	if !e.promptingAbility || len(e.abilityMenu) == 0 {
		return screen
	}

	menuY := e.H/2 - (len(e.abilityMenu)*4)/2
	for i, zone := range e.abilityMenu {
		borderColor := ui.Colors["dark-brown"]
		if e.abilityMenuHovered[i] {
			borderColor = ui.Colors["brown"]
		}
		abilityText := fmt.Sprintf("%d. %s", i+1, e.abilityChoices[i])
		// Extract ability text if it's from a card
		if abilityStr, ok := e.abilityChoices[i].(string); ok {
			// Parse the ability identifier (e.g., "123.0")
			var cardID, abilityIdx int
			if n, _ := fmt.Sscanf(abilityStr, "%d.%d", &cardID, &abilityIdx); n == 2 {
				// Find the card and get the ability text
				if card, ok := e.cardMap[cardID]; ok && card.cardInstance != nil {
					abilities := card.cardInstance.GetActivatedAbilities()
					if abilityIdx >= 0 && abilityIdx < len(abilities) {
						abilityText = abilities[abilityIdx].Text()
					}
				}
			}
		}
		menuStyle := ui.NewStyle().
			Border(ui.Borders["round"]).
			BorderForeground(borderColor).
			BorderBackground(ui.Colors["dark"]).
			Foreground(ui.Colors["light-beige"]).
			Background(ui.Colors["dark"]).
			Padding(0, 1)
		zone.M = menuStyle.Render(ui.Text(abilityText).View())
		screen = screen.Overlay(zone.View(), e.W/2-15, menuY+i*4)
	}
	return screen
}

// renderSkipButton renders the skip button during prompts
func (e *CardGame) renderSkipButton(screen *ui.TileMap) *ui.TileMap {
	if !e.prompting {
		return screen
	}

	borderColor := ui.Colors["dark-brown"]
	if e.skipButtonHovered {
		borderColor = ui.Colors["brown"]
	}
	buttonStyle := ui.NewStyle().
		Border(ui.Borders["round"]).
		BorderForeground(borderColor).
		BorderBackground(ui.Colors["dark"]).
		Foreground(ui.Colors["light-beige"]).
		Background(ui.Colors["dark"]).
		AlignHorizontal(ui.Center).
		AlignVertical(ui.Center)
	e.skipButton.M = buttonStyle.Render(ui.Text("Skip").View())
	return screen.Overlay(e.skipButton.View(), e.W-12, e.H/2)
}

// renderZoneWithHover is a helper function to render a zone with hover highlighting
func renderZoneWithHover(zone *ui.Zone, hovered bool, border ui.Border, normalColor, hoverColor ui.ColorIndex) *ui.TileMap {
	borderColor := normalColor
	if hovered {
		borderColor = hoverColor
	}
	style := ui.NewStyle().
		Border(border).
		BorderForeground(borderColor).
		BorderBackground(ui.Colors["dark"]).
		Background(ui.Colors["dark"])
	return style.Render(zone.View())
}
