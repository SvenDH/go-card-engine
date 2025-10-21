package screens

import (
	"fmt"
	"time"

	"github.com/SvenDH/go-card-engine/ui"
)

// renderEssencePools renders the essence pool for player and enemy
func (e *CardGame) renderEssencePools(screen *ui.Image) *ui.Image {
	// Color type to icon name and color mapping
	essenceTypes := []struct {
		code     string
		iconName string
		color    string
	}{
		{"c", "hearts", "blue"},
		{"s", "spades", "yellow"},
		{"o", "diamonds", "green"},
		{"w", "clubs", "red"},
		{"u", "heart", "white"},
	}

	// Player essence pool (bottom-left)
	startY := e.H - 12
	displayIndex := 0
	for _, et := range essenceTypes {
		count := e.playerEssence[et.code]
		if count > 0 { // Only show essence types with count > 0
			// Render icon
			icon := ui.Icon(et.iconName)
			if colorIndex, ok := ui.Colors[et.color]; ok {
				icon.Tiles[0].Color = ui.ColorIndex(colorIndex)
			}
			icon.Tiles[0].Background = ui.ColorIndex(ui.Colors["dark"])
			screen = screen.Overlay(icon, 2+e.shakeX, startY+displayIndex*2+e.shakeY)

			// Render count
			countText := fmt.Sprintf("%d", count)
			countStyle := ui.NewStyle().
				Foreground(ui.Colors["light-beige"]).
				Background(ui.Colors["dark"])
			screen = screen.Overlay(countStyle.Render(ui.Text(countText).View()), 4+e.shakeX, startY+displayIndex*2+e.shakeY)
			displayIndex++
		}
	}

	// Enemy essence pool (top-right)
	startY = 5
	displayIndex = 0
	for _, et := range essenceTypes {
		count := e.enemyEssence[et.code]
		if count > 0 { // Only show essence types with count > 0
			// Render icon
			icon := ui.Icon(et.iconName)
			if colorIndex, ok := ui.Colors[et.color]; ok {
				icon.Tiles[0].Color = ui.ColorIndex(colorIndex)
			}
			icon.Tiles[0].Background = ui.ColorIndex(ui.Colors["dark"])
			screen = screen.Overlay(icon, e.W-6+e.shakeX, startY+displayIndex*2+e.shakeY)

			// Render count
			countText := fmt.Sprintf("%d", count)
			countStyle := ui.NewStyle().
				Foreground(ui.Colors["light-beige"]).
				Background(ui.Colors["dark"])
			screen = screen.Overlay(countStyle.Render(ui.Text(countText).View()), e.W-4+e.shakeX, startY+displayIndex*2+e.shakeY)
			displayIndex++
		}
	}

	return screen
}

// renderLifeTotals renders player and enemy life totals
func (e *CardGame) renderLifeTotals(screen *ui.Image) *ui.Image {
	playerLifeStr := fmt.Sprintf("Life: %d", e.playerLife)

	// Flash red if damage was taken recently (within 500ms)
	playerColor := ui.Colors["light-beige"]
	if !e.playerHealthFlash.IsZero() && time.Since(e.playerHealthFlash) < 500*time.Millisecond {
		playerColor = ui.Colors["red"]
	}

	// Player heart icon
	playerHeart := ui.Icon("heart")
	playerHeart.Tiles[0].Color = ui.ColorIndex(playerColor)
	playerHeart.Tiles[0].Background = ui.ColorIndex(ui.Colors["dark"])
	screen = screen.Overlay(playerHeart, 2+e.shakeX, e.H-2+e.shakeY)

	playerLifeStyle := ui.NewStyle().
		Foreground(playerColor).
		Background(ui.Colors["dark"])
	playerLifeText := ui.Text(playerLifeStr)
	screen = screen.Overlay(playerLifeStyle.Render(playerLifeText.View()), 4+e.shakeX, e.H-2+e.shakeY)

	enemyLifeStr := fmt.Sprintf("Enemy: %d", e.enemyLife)

	// Flash red if damage was taken recently (within 500ms)
	enemyColor := ui.Colors["light-beige"]
	if !e.enemyHealthFlash.IsZero() && time.Since(e.enemyHealthFlash) < 500*time.Millisecond {
		enemyColor = ui.Colors["red"]
	}

	// Enemy heart icon
	enemyHeart := ui.Icon("heart")
	enemyHeart.Tiles[0].Color = ui.ColorIndex(enemyColor)
	enemyHeart.Tiles[0].Background = ui.ColorIndex(ui.Colors["dark"])
	screen = screen.Overlay(enemyHeart, e.W-13+e.shakeX, 2+e.shakeY)

	enemyLifeStyle := ui.NewStyle().
		Foreground(enemyColor).
		Background(ui.Colors["dark"])
	enemyLifeText := ui.Text(enemyLifeStr)
	return screen.Overlay(enemyLifeStyle.Render(enemyLifeText.View()), e.W-11+e.shakeX, 2+e.shakeY)
}

// renderCards renders all cards in the correct z-order
func (e *CardGame) renderCards(screen *ui.Image) *ui.Image {
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
					screen = screen.Overlay(c.View(), c.X+e.shakeX, c.Y+e.shakeY)
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
			screen = screen.Overlay(card.View(), card.X+e.shakeX, card.GetDrawY()+e.shakeY)
		}
	}

	// Draw all dragged cards on top (excluding stack card)
	for _, cards := range allCardCollections {
		for _, card := range cards {
			if card != nil {
				c := card.(*Card)
				if c.dragging && c != stackCard {
					screen = screen.Overlay(c.View(), c.X+e.shakeX, c.Y+e.shakeY)
				}
			}
		}
	}

	// Draw hovered/dragged hand card on top of everything
	if hoveredCard != nil {
		screen = screen.Overlay(hoveredCard.View(), hoveredCard.X+e.shakeX, hoveredCard.GetDrawY()+e.shakeY)
	}

	return screen
}

// renderStack renders the stack zone with the card or ability on it
func (e *CardGame) renderStack(screen *ui.Image) *ui.Image {
	abilities := e.stack.GetAbilities()

	// Check if there's anything to show
	hasCard := e.stack.HasCard()
	hasAbilities := len(abilities) > 0

	if !hasCard && !hasAbilities {
		return screen
	}

	stackX := e.W/2 - 6
	stackY := e.H/2 - 8

	// Draw card or ability
	if hasCard {
		// Draw the card being cast
		stackCard := e.stack.GetCard()
		if stackCard != nil {
			cardX := stackX + 1
			cardY := stackY + 1
			screen = screen.Overlay(stackCard.View(), cardX+e.shakeX, cardY+e.shakeY)
		}
	} else if hasAbilities {
		// Show the top ability on the stack with its source card
		topAbility := abilities[len(abilities)-1]

		// Draw the source card without border highlight and stats
		if topAbility.Source != nil {
			sourceCard := e.cardMap[topAbility.Source.GetId()]
			if sourceCard != nil {
				cardX := stackX + 1
				cardY := stackY + 1

				// Render card without border highlight and stats
				cardView := renderCardForStack(sourceCard)
				screen = screen.Overlay(cardView, cardX+e.shakeX, cardY+e.shakeY)

				// Overlay ability text on the card
				if textGetter, ok := topAbility.Ability.(interface{ Text() string }); ok {
					abilityText := textGetter.Text()

					// Position overlay in middle of card
					overlayX := cardX + 1
					overlayY := cardY + 2

					// Add ability text (word wrapped)
					textStyle := ui.NewStyle().
						Foreground(ui.Colors["white"]).
						Background(ui.Colors["dark"]).
						Width(10)

					// Simple word wrap
					lines := []string{}
					words := splitWords(abilityText)
					currentLine := ""
					for _, word := range words {
						if len(currentLine)+len(word)+1 <= 10 {
							if currentLine == "" {
								currentLine = word
							} else {
								currentLine += " " + word
							}
						} else {
							if currentLine != "" {
								lines = append(lines, currentLine)
							}
							currentLine = word
						}
					}
					if currentLine != "" {
						lines = append(lines, currentLine)
					}

					// Draw text lines
					textY := overlayY + 1
					for _, line := range lines {
						lineText := ui.Text(line)
						screen = screen.Overlay(textStyle.Render(lineText.View()), overlayX+e.shakeX, textY+e.shakeY)
						textY++
					}
				}
			}
		}
	}

	return screen
}

// renderPhaseLabel renders the current phase and player turn indicator
func (e *CardGame) renderPhaseLabel(screen *ui.Image) *ui.Image {
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
	return screen.Overlay(e.phaseLabel.View(), 2+e.shakeX, 2+e.shakeY)
}

// renderAbilityMenu renders the ability selection menu
func (e *CardGame) renderAbilityMenu(screen *ui.Image) *ui.Image {
	if !e.promptingAbility || len(e.abilityMenu) == 0 {
		return screen
	}

	// Calculate menu position - default to center if no card is set
	menuX := e.W/2 - 15
	menuY := e.H/2 - (len(e.abilityMenu)*4)/2

	// Position menu next to the ability card if available
	if e.abilityCard != nil {
		// Position to the right of the card
		menuX = e.abilityCard.X + e.abilityCard.Picture.W + 4
		menuY = e.abilityCard.Y

		// If menu would go off-screen right, position to the left instead
		if menuX+30 > e.W {
			menuX = e.abilityCard.X - 34
		}

		// Ensure menu doesn't go off-screen top/bottom
		maxMenuHeight := len(e.abilityMenu) * 4
		if menuY+maxMenuHeight > e.H {
			menuY = e.H - maxMenuHeight - 1
		}
		if menuY < 0 {
			menuY = 0
		}
	}

	for i, zone := range e.abilityMenu {
		borderColor := ui.Colors["dark-brown"]
		// Keyboard focus takes priority over mouse hover
		if e.focusMode == "ability-menu" && e.focusAbilityIndex == i {
			borderColor = ui.Colors["white"]
		} else if e.abilityMenuHovered[i] {
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
		screen = screen.Overlay(zone.View(), menuX+e.shakeX, menuY+i*4+e.shakeY)
	}
	return screen
}

// renderSkipButton renders the skip button during prompts
func (e *CardGame) renderSkipButton(screen *ui.Image) *ui.Image {
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
	return screen.Overlay(e.skipButton.View(), e.W-12+e.shakeX, e.H/2+e.shakeY)
}

// renderZoneWithHover is a helper function to render a zone with hover highlighting
func renderZoneWithHover(zone *ui.Zone, hovered bool, border ui.Border, normalColor, hoverColor ui.ColorIndex) *ui.Image {
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

func splitWords(text string) []string {
	words := []string{}
	current := ""
	for _, r := range text {
		if r == ' ' {
			if current != "" {
				words = append(words, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		words = append(words, current)
	}
	return words
}

// renderCardForStack renders a card for the stack without border highlight and stats
func renderCardForStack(c *Card) *ui.Image {
	// Use neutral border (no highlight)
	borderStyle := ui.NewStyle().
		Border(ui.Borders["round"]).
		BorderForeground(ui.Colors["dark-brown"]).
		BorderBackground(ui.Colors["dark"])

	topStyle := ui.NewStyle().
		Foreground(ui.Colors["dark"]).
		Background(c.BorderColor).
		Width(c.Picture.W)

	pic := c.Picture.View()
	label := ui.Text(c.Name)
	if len(c.Name) > c.Picture.W-2 {
		label = ui.Text(c.Name[:c.Picture.W-2])
	}

	// Build cost icons
	var costIcons *ui.Image
	if c.cardInstance != nil && len(c.cardInstance.Card.Costs) > 0 {
		// Map shorthand color codes to full icon names
		colorToIcon := map[string]string{
			"c": "hearts",
			"s": "spades",
			"o": "diamonds",
			"w": "clubs",
		}
		
		costParts := []*ui.Image{}
		for _, cost := range c.cardInstance.Card.Costs {
			if cost.Color != "" {
				// Convert shorthand color code to full icon name
				iconName := colorToIcon[cost.Color]
				if iconName == "" {
					iconName = cost.Color // Use as-is if not a shorthand
				}
				icon := ui.Icon(iconName)
				icon.Tiles[0].Color = ui.ColorIndex(ui.Colors["dark"])
				icon.Tiles[0].Background = ui.ColorIndex(c.BorderColor)
				costParts = append(costParts, icon)
			} else if cost.Number.Number > 0 || cost.Number.X {
				// Add number text
				numText := ui.Text(cost.Number.String())
				numText.Tiles[0].Color = ui.ColorIndex(ui.Colors["dark"])
				numText.Tiles[0].Background = ui.ColorIndex(c.BorderColor)
				costParts = append(costParts, numText)
			}
		}
		if len(costParts) > 0 {
			costIcons = ui.JoinHorizontal(ui.Top, costParts...)
		}
	}

	top := ui.JoinHorizontal(ui.Top, label.View())
	top = topStyle.Render(top)

	// Join top and picture (no stats)
	content := ui.JoinVertical(ui.Left, top, pic)

	// Overlay cost icons in the top right corner
	if costIcons != nil {
		costX := c.Picture.W - costIcons.W
		content = content.Overlay(costIcons, costX, 0)
	}

	// Don't add stats - skip the stats overlay section

	tm := borderStyle.Render(content)
	if c.Disabled {
		tm.Tiles.Darken(2)
		tm.Tiles.Desaturate()
	}
	return tm
}
