package ui

import (
	"image"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

type ColorIndex int8

const (
	// ColorLockBit prevents the color from being changed by subsequent Color() calls
	ColorLockBit ColorIndex = 0x40
	// ColorMask extracts the actual color value without the lock bit
	ColorMask ColorIndex = 0x3F
)

type Tile struct {
	Index      int16
	Color      ColorIndex
	Background ColorIndex
}

type Tiles []Tile

func (t Tiles) Background(bg ColorIndex) {
	for i := range len(t) {
		// Only change background if the lock bit is not set
		if t[i].Background&ColorLockBit == 0 {
			t[i].Background = bg
		}
	}
}

func (t Tiles) Clear() {
	for i := range len(t) {
		t[i].Index = 0
	}
}

func (t Tiles) Color(color ColorIndex) {
	for i := range len(t) {
		// Only change color if the lock bit is not set
		if t[i].Color&ColorLockBit == 0 {
			t[i].Color = color
		}
	}
}

func (t Tiles) Reverse() {
	for i := range t {
		// Only reverse if neither color nor background has lock bit
		if t[i].Color&ColorLockBit == 0 && t[i].Background&ColorLockBit == 0 {
			t[i].Color, t[i].Background = t[i].Background, t[i].Color
		}
	}
}

func (t Tiles) Faint() {
	for i := range t {
		if t[i].Color&ColorLockBit == 0 {
			t[i].Color = ColorIndex(FaintMap[t[i].Color&ColorMask])
		}
	}
}

func (t Tiles) Bright() {
	for i := range t {
		if t[i].Color&ColorLockBit == 0 {
			t[i].Color = ColorIndex(BrightMap[t[i].Color&ColorMask])
		}
	}
}

func (t Tiles) Darken(amount int) {
	for i := range t {
		if t[i].Color&ColorLockBit == 0 {
			for range amount {
				t[i].Color = ColorIndex(FaintMap[t[i].Color&ColorMask])
			}
		}
		if t[i].Background&ColorLockBit == 0 {
			for range amount {
				t[i].Background = ColorIndex(FaintMap[t[i].Background&ColorMask])
			}
		}
	}
}

func (t Tiles) Desaturate() {
	for i := range t {
		if t[i].Color&ColorLockBit == 0 {
			t[i].Color = ColorIndex(desatMap[t[i].Color&ColorMask])
		}
		if t[i].Background&ColorLockBit == 0 {
			t[i].Background = ColorIndex(desatMap[t[i].Background&ColorMask])
		}
	}
}

func (t Tiles) StringLength() int {
	len := len(t)
	for i := range len {
		if !(t[i].Index == 0 && t[i].Color == 0 && t[i].Background == 0) {
			break
		}
		len--
	}
	return len
}

func (t Tiles) Style(s Style) Tiles {
	var (
		bold          = s.getAsBool(boldKey, false)
		italic        = s.getAsBool(italicKey, false)
		underline     = s.getAsBool(underlineKey, false)
		strikethrough = s.getAsBool(strikethroughKey, false)
		reverse       = s.getAsBool(reverseKey, false)
		faint         = s.getAsBool(faintKey, false)

		fg = s.getAsColor(foregroundKey)
		bg = s.getAsColor(backgroundKey)
	)

	if fg != -1 {
		t.Color(ColorIndex(fg))
	}
	if bg != -1 {
		t.Background(ColorIndex(bg))
	}
	if reverse {
		t.Reverse()
	}
	if faint {
		t.Faint()
	}
	if bold {
		t.Bright()
	}
	if italic {

	}
	if underline {

	}
	if strikethrough {

	}
	return t
}

type Image struct {
	W, H  int
	Tiles Tiles
	Zones []*Zone
}

func NewImage(w, h int, zones []*Zone) *Image {
	return &Image{
		W:     w,
		H:     h,
		Tiles: make(Tiles, w*h),
		Zones: zones,
	}
}

func Text(text string) *Image {
	// Check if text contains cost symbols - if so, use RichText
	if strings.Contains(text, "{") {
		return RichText(strings.ToLower(text))
	}
	tm := NewImage(len(text), 1, nil)
	tm.SetText(text)
	tm.Color("white")
	return tm
}

// RichText creates a TileMap that supports icon and color syntax:
// - {iconname} - looks up icon in TileIndices (e.g., {clubs}, {hearts})
// - {color:iconname} - looks up both color and icon (e.g., {blue:hearts})
// - Backward compatible with single-char shortcuts: {c}, {s}, {o}, {w}, {t}, {q}
func RichText(text string) *Image {
	// Shorthand mappings for backward compatibility
	shorthandIcons := map[byte]string{
		'c': "hearts",
		's': "spades",
		'o': "diamonds",
		'w': "clubs",
		't': "arrow_down",
		'q': "arrow_up",
	}
	shorthandColors := map[byte]string{
		'c': "blue",
		's': "yellow",
		'o': "green",
		'w': "red",
		't': "sky-blue",
		'q': "sky-blue",
	}

	// First pass: calculate width by parsing the text
	width := 0
	i := 0
	for i < len(text) {
		if text[i] == '{' {
			// Find the closing brace
			j := i + 1
			for j < len(text) && text[j] != '}' {
				j++
			}
			if j < len(text) {
				// Found closing brace - icon takes 1 tile
				width++
				i = j + 1
			} else {
				// No closing brace, treat as regular character
				width++
				i++
			}
		} else {
			// Regular character
			width++
			i++
		}
	}

	tm := NewImage(width, 1, nil)

	// Second pass: build the tilemap
	x := 0
	i = 0
	for i < len(text) {
		if text[i] == '{' {
			// Find the closing brace
			j := i + 1
			for j < len(text) && text[j] != '}' {
				j++
			}
			if j < len(text) {
				// Found closing brace
				content := text[i+1 : j]
				
				// Parse content - could be "iconname" or "color:iconname"
				var iconName, colorName string
				if colonIdx := strings.Index(content, ":"); colonIdx != -1 {
					// Format: {color:iconname}
					colorName = content[:colonIdx]
					iconName = content[colonIdx+1:]
				} else if len(content) == 1 {
					// Single character - check for shorthand
					char := content[0]
					if icon, ok := shorthandIcons[char]; ok {
						iconName = icon
						colorName = shorthandColors[char]
					} else if char >= '0' && char <= '9' || char == 'X' {
						// Number or X - render as text character
						if r, ok := runeCharMap[rune(char)]; ok {
							tm.Tiles[x].Index = int16(r)
							tm.Tiles[x].Color = Colors["sky-blue"] | ColorLockBit
						}
						x++
						i = j + 1
						continue
					} else {
						// Unknown single char, try as icon name
						iconName = content
					}
				} else {
					// Multi-character - treat as icon name
					iconName = content
				}

				// Look up icon in TileIndices
				if iconIndex, ok := TileIndices[iconName]; ok {
					tm.Tiles[x].Index = int16(iconIndex)
				}

				// Apply color if specified
				if colorName != "" {
					if color, ok := Colors[colorName]; ok {
						tm.Tiles[x].Color = color | ColorLockBit
					}
				} else {
					tm.Tiles[x].Color = Colors["white"] | ColorLockBit
				}

				x++
				i = j + 1
			} else {
				// No closing brace, treat as regular character
				if r, ok := runeCharMap[rune(text[i])]; ok {
					tm.Tiles[x].Index = int16(r)
					tm.Tiles[x].Color = Colors["white"] | ColorLockBit
				}
				x++
				i++
			}
		} else {
			// Regular character
			if r, ok := runeCharMap[rune(text[i])]; ok {
				tm.Tiles[x].Index = int16(r)
				tm.Tiles[x].Color = Colors["white"] | ColorLockBit
			}
			x++
			i++
		}
	}
	tm.Color("white")
	return tm
}

func Icon(key string) *Image {
	tm := NewImage(1, 1, nil)
	tm.Color("white")
	if index, ok := TileIndices[key]; ok {
		tm.Tiles[0].Index = int16(index)
	}
	return tm
}

func (tm *Image) Init() Cmd {
	return nil
}

func (tm *Image) Update(msg Msg) (Model, Cmd) {
	return tm, nil
}

func (tm *Image) View() *Image {
	return tm.Copy()
}

func (tm *Image) Copy() *Image {
	return &Image{
		W:     tm.W,
		H:     tm.H,
		Tiles: append(Tiles{}, tm.Tiles...),
		Zones: append([]*Zone{}, tm.Zones...),
	}
}

func (tm *Image) Set(x, y, tile, c, bg int) *Image {
	tm.Tiles[x+y*tm.W] = Tile{Index: int16(tile), Color: ColorIndex(c), Background: ColorIndex(bg)}
	return tm
}

func (tm *Image) AddZone(zone *Zone) *Image {
	tm.Zones = append(tm.Zones, zone)
	return tm
}

func (tm *Image) Draw(screen *ebiten.Image, ox, oy int) {
	if tm.W == 0 || tm.H == 0 {
		return
	}
	tw := TilesImage.Bounds().Dx() / TileSize
	bgx, bgy := TilesImage.Bounds().Dx(), TilesImage.Bounds().Dy()
	for i, t := range tm.Tiles {
		idx := int(t.Index)
		// Mask out lock bit to get actual color index
		fg := Palette[t.Color&ColorMask]
		bg := Palette[t.Background&ColorMask]
		if idx < 0 {
			idx = -idx
			fg, bg = bg, fg
		}
		sx, sy := (idx%tw)*TileSize, (idx/tw)*TileSize
		tx := float64(ox + (i%tm.W)*TileSize)
		ty := float64(oy + (i/tm.W)*TileSize)
		if ty >= float64(oy+tm.H*TileSize) {
			break
		}
		op := &ebiten.DrawImageOptions{ColorScale: bg}
		op.GeoM.Translate(tx, ty)
		screen.DrawImage(TilesImage.SubImage(image.Rect(bgx-TileSize, bgy-TileSize, bgx, bgy)).(*ebiten.Image), op)

		op = &ebiten.DrawImageOptions{ColorScale: fg}
		op.GeoM.Translate(tx, ty)
		screen.DrawImage(TilesImage.SubImage(image.Rect(sx, sy, sx+TileSize, sy+TileSize)).(*ebiten.Image), op)
	}
}

func (tm *Image) Clear() *Image {
	tm.Tiles.Clear()
	return tm
}

func (tm *Image) Color(color string) *Image {
	tm.Tiles.Color(Colors[color])
	return tm
}

func (tm *Image) Background(color string) *Image {
	tm.Tiles.Background(Colors[color])
	return tm
}

func (tm *Image) SetText(text string) {
	x, y := 0, 0
	width := tm.W
	for _, c := range text {
		if r, ok := runeCharMap[c]; ok {
			if x < tm.W && y < tm.H {
				tm.Tiles[x+(y*width)].Index = int16(r)
			}
			x++
			if x >= tm.W || c == '\n' {
				x = 0
				y++
				if y >= tm.H {
					break
				}
			}
		}
	}
}

func (tm *Image) Overlay(tm2 *Image, x, y int) *Image {
	for i := range tm2.Tiles {
		t := tm2.Tiles[i]
		tx := x + i%tm2.W
		ty := y + i/tm2.W
		// Check bounds including negative coordinates
		if tx >= 0 && tx < tm.W && ty >= 0 && ty < tm.H {
			idx := tx + ty*tm.W
			// If the background is 0, use the background of the tile below it
			if t.Background == 0 {
				t.Background = tm.Tiles[idx].Background
			}
			if t.Color == 0 {
				t.Color = tm.Tiles[idx].Background
			}
			tm.Tiles[idx] = t
		}
	}
	for _, zone := range tm2.Zones {
		tm.Zones = append(tm.Zones, zone.Add(x, y))
	}
	return tm
}

func (tm *Image) In(pos image.Point) bool {
	return pos.In(image.Rectangle{Max: image.Point{tm.W * TileSize, tm.H * TileSize}})
}

func (tm *Image) Wrap(width int) *Image {
	tm.W = width
	return tm
}

func (tm *Image) ExtendBottom(n int, s Style) *Image {
	tm.H += n
	tm.Tiles = append(tm.Tiles, make(Tiles, n*tm.W).Style(s)...)
	return tm
}

func (tm *Image) ExtendTop(n int, s Style) *Image {
	tm.Tiles = append(make(Tiles, n*tm.W).Style(s), tm.Tiles...)
	tm.H += n
	for i := range tm.Zones {
		tm.Zones[i] = tm.Zones[i].Add(0, n)
	}
	return tm
}

func (tm *Image) ExtendRight(n int, s Style) *Image {
	var newTiles Tiles
	for y := range tm.H {
		pad := append(tm.Tiles[y*tm.W:(y+1)*tm.W], make(Tiles, n).Style(s)...)
		newTiles = append(newTiles, pad...)
	}
	tm.Tiles = newTiles
	tm.W += n
	return tm
}

func (tm *Image) ExtendLeft(n int, s Style) *Image {
	var newTiles Tiles
	for y := range tm.H {
		pad := append(make(Tiles, n).Style(s), tm.Tiles[y*tm.W:(y+1)*tm.W]...)
		newTiles = append(newTiles, pad...)
	}
	tm.Tiles = newTiles
	tm.W += n
	for i := range tm.Zones {
		tm.Zones[i] = tm.Zones[i].Add(n, 0)
	}
	return tm
}

func (tm *Image) Crop(bounds ...int) *Image {
	top, right, bottom, left, ok := whichSidesInt(bounds...)
	if !ok {
		return tm
	}
	var newTiles Tiles
	for y := top; y < tm.H-bottom; y++ {
		newTiles = append(newTiles, tm.Tiles[y*tm.W+left:(y+1)*tm.W-left]...)
	}
	tm.Tiles = newTiles
	tm.W -= left + right
	tm.H -= top + bottom
	return tm
}

func (tm *Image) GetContentBounds() (int, int, int, int) {
	startX, startY, endX, endY := 1<<31, 1<<31, -1, -1
	for y := range tm.H {
		for x := range tm.W {
			tile := tm.Tiles[x+(y*tm.W)]
			if tile.Index != 0 {
				if x < startX {
					startX = x
				}
				if y < startY {
					startY = y
				}
				if x > endX {
					endX = x
				}
				if y > endY {
					endY = y
				}
			}
		}
	}
	return startX, startY, endX, endY
}

func (tm Image) getRows() []Tiles {
	var rows []Tiles
	for y := 0; y < tm.H; y++ {
		rows = append(rows, tm.Tiles[y*tm.W:(y+1)*tm.W])
	}
	return rows
}
