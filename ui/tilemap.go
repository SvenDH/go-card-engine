package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type ColorIndex int8

type Tile struct {
	Index      int16
	Color      ColorIndex
	Background ColorIndex
}

type Tiles []Tile

func (t Tiles) Background(bg ColorIndex) {
	for i := range len(t) {
		t[i].Background = bg
	}
}

func (t Tiles) Clear(bg ColorIndex) {
	for i := range len(t) {
		t[i].Index = 0
		t[i].Background = bg
	}
}

func (t Tiles) Color(color ColorIndex) {
	for i := range len(t) {
		t[i].Color = color
	}
}

func (t Tiles) Reverse() {
	for i := range t {
		t[i].Color, t[i].Background = t[i].Background, t[i].Color
	}
}

func (t Tiles) Faint() {
	for i := range t {
		t[i].Color = ColorIndex(faintMap[t[i].Color])
	}
}

func (t Tiles) Bright() {
	for i := range t {
		t[i].Color = ColorIndex(brightMap[t[i].Color])
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

type TileMap struct {
	W, H  int
	Tiles Tiles
	Zones []*Zone
}

func NewTileMap(w, h int, zones []*Zone) *TileMap {
	return &TileMap{
		W:     w,
		H:     h,
		Tiles: make(Tiles, w*h),
		Zones: zones,
	}
}

func Text(text string) *TileMap {
	tm := NewTileMap(len(text), 1, nil)
	tm.SetText(text)
	tm.Color(ColorIndex(Colors["white"]))
	return tm
}

func (tm *TileMap) Init() Cmd {
	return nil
}

func (tm *TileMap) Update(msg Msg) (Model, Cmd) {
    return tm, nil
}

func (tm *TileMap) View() *TileMap {
	return tm.Copy()
}

func (tm *TileMap) Copy() *TileMap {
	return &TileMap{
		W:     tm.W,
		H:     tm.H,
		Tiles: append(Tiles{}, tm.Tiles...),
		Zones: append([]*Zone{}, tm.Zones...),
	}
}

func (tm *TileMap) Set(x, y, tile, c, bg int) *TileMap {
	tm.Tiles[x+y*tm.W] = Tile{Index: int16(tile), Color: ColorIndex(c), Background: ColorIndex(bg)}
	return tm
}

func (tm *TileMap) AddZone(zone *Zone) *TileMap {
	tm.Zones = append(tm.Zones, zone)
	return tm
}

func (tm *TileMap) Draw(screen *ebiten.Image, ox, oy int) {
	if tm.W == 0 || tm.H == 0 {
		return
	}
	tw := TilesImage.Bounds().Dx() / TileSize
	bgx, bgy := TilesImage.Bounds().Dx(), TilesImage.Bounds().Dy()
	for i, t := range tm.Tiles {
		idx := int(t.Index)
		fg := Palette[t.Color]
		bg := Palette[t.Background]
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

func (tm *TileMap) Clear(bg ColorIndex) *TileMap {
	tm.Tiles.Clear(bg)
	return tm
}

func (tm *TileMap) Color(color ColorIndex) *TileMap {
	tm.Tiles.Color(color)
	return tm
}

func (tm *TileMap) Background(color ColorIndex) *TileMap {
	tm.Tiles.Background(color)
	return tm
}

func (tm *TileMap) SetText(text string) {
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

func (tm *TileMap) Overlay(tm2 *TileMap, x, y int) *TileMap {
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

func (tm *TileMap) In(pos image.Point) bool {
	return pos.In(image.Rectangle{Max: image.Point{tm.W * TileSize, tm.H * TileSize}})
}

func (tm *TileMap) Wrap(width int) *TileMap {
	tm.W = width
	return tm
}

func (tm *TileMap) ExtendBottom(n int, s Style) *TileMap {
	tm.H += n
	tm.Tiles = append(tm.Tiles, make(Tiles, n*tm.W).Style(s)...)
	return tm
}

func (tm *TileMap) ExtendTop(n int, s Style) *TileMap {
	tm.Tiles = append(make(Tiles, n*tm.W).Style(s), tm.Tiles...)
	tm.H += n
	for i := range tm.Zones {
		tm.Zones[i] = tm.Zones[i].Add(0, n)
	}
	return tm
}

func (tm *TileMap) ExtendRight(n int, s Style) *TileMap {
	var newTiles Tiles
	for y := range tm.H {
		pad := append(tm.Tiles[y*tm.W:(y+1)*tm.W], make(Tiles, n).Style(s)...)
		newTiles = append(newTiles, pad...)
	}
	tm.Tiles = newTiles
	tm.W += n
	return tm
}

func (tm *TileMap) ExtendLeft(n int, s Style) *TileMap {
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

func (tm *TileMap) Crop(bounds ...int) *TileMap {
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

func (tm *TileMap) GetContentBounds() (int, int, int, int) {
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

func (tm TileMap) getRows() []Tiles {
	var rows []Tiles
	for y := 0; y < tm.H; y++ {
		rows = append(rows, tm.Tiles[y*tm.W:(y+1)*tm.W])
	}
	return rows
}
