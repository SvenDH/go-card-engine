package ui

import (
	"image"
	"image/color"
	"runtime"
	"sync"

	"github.com/anthonynsimon/bild/transform"

	"github.com/SvenDH/go-card-engine/ui"
)

// ImageToTileMap converts a source image into a TileMap that approximates it as closely as possible
// using the available tiles in tilesSource and the palette colors. The provided rect defines where
// the tilemap will be placed and its size; only full tiles inside the rect are considered.
func ImageToTileMap(src image.Image, w, h int) *ui.Image {
	if src == nil || ui.TilesSource == nil || len(ui.PaletteRGBA) == 0 {
		return ui.NewImage(w, h, nil)
	}

	// Determine grid size (only full tiles)
	n := w * h
	if n <= 0 {
		return ui.NewImage(w, h, nil)
	}

	tm := ui.NewImage(w, h, nil)

	// Tileset layout
	tw := ui.TilesSource.Bounds().Dx() / ui.TileSize
	th := ui.TilesSource.Bounds().Dy() / ui.TileSize
	tileCount := tw * th
	if tileCount == 0 {
		return tm
	}
	bgTileIndex := tileCount - 1 // convention: last tile is solid background tile

	src = transform.Resize(src, w*ui.TileSize, h*ui.TileSize, transform.Lanczos)

	// Pre-extract all tiles' pixels from tilesSource into memory
	tilesPix := make([][]color.NRGBA, tileCount)
	tilesA := make([][]uint8, tileCount)
	for ti := 0; ti < tileCount; ti++ {
		sx := (ti % tw) * ui.TileSize
		sy := (ti / tw) * ui.TileSize
		tilesPix[ti] = make([]color.NRGBA, ui.TileSize*ui.TileSize)
		tilesA[ti] = make([]uint8, ui.TileSize*ui.TileSize)
		idx := 0
		for y := 0; y < ui.TileSize; y++ {
			for x := 0; x < ui.TileSize; x++ {
				c := color.NRGBAModel.Convert(ui.TilesSource.At(sx+x, sy+y)).(color.NRGBA)
				tilesPix[ti][idx] = c
				tilesA[ti][idx] = c.A
				idx++
			}
		}
	}

	// Pre-extract the target source pixels for the destination rect
	srcPix := make([]color.NRGBA, w*h*ui.TileSize*ui.TileSize)
	for gy := 0; gy < h; gy++ {
		for gx := 0; gx < w; gx++ {
			base := (gy*w + gx) * ui.TileSize * ui.TileSize
			idx := 0
			for py := 0; py < ui.TileSize; py++ {
				for px := 0; px < ui.TileSize; px++ {
					x := gx*ui.TileSize + px
					y := gy*ui.TileSize + py
					c := color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA)
					srcPix[base+idx] = c
					idx++
				}
			}
		}
	}

	// Exhaustive search for each cell: choose tile, fg, bg that minimize squared RGB error
	palLen := len(ui.PaletteRGBA)
	workerCount := runtime.GOMAXPROCS(0)
	if workerCount < 1 {
		workerCount = 1
	}
	jobs := make(chan int, workerCount*2)
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for cell := range jobs {
			bestErr := int(^uint(0) >> 1) // max int
			bestT, bestFg, bestBg := 0, 0, 0
			base := cell * ui.TileSize * ui.TileSize
			// Loop tiles
			for ti := 0; ti < tileCount; ti++ {
				// Loop foreground colors
				for fg := 0; fg < palLen; fg++ {
					fgc := ui.PaletteRGBA[fg]
					// Loop background colors
					for bg := 0; bg < palLen; bg++ {
						bgc := ui.PaletteRGBA[bg]
						err := 0
						// Compare pixels
						for i := 0; i < ui.TileSize*ui.TileSize; i++ {
							// FG pixel after color scale multiplication
							fp := tilesPix[ti][i]
							fa := float64(tilesA[ti][i]) / 255.0
							fr := float64(fp.R) * float64(fgc.R) / 255.0
							fg_ := float64(fp.G) * float64(fgc.G) / 255.0
							fb_ := float64(fp.B) * float64(fgc.B) / 255.0

							// BG tile pixel: take background tile pixel modulated by bg color
							bp := tilesPix[bgTileIndex][i]
							br := float64(bp.R) * float64(bgc.R) / 255.0
							bgG := float64(bp.G) * float64(bgc.G) / 255.0
							bb := float64(bp.B) * float64(bgc.B) / 255.0

							// Alpha composite fg over bg: out = fg + (1-fa)*bg
							or := fr + (1.0-fa)*br
							og := fg_ + (1.0-fa)*bgG
							ob := fb_ + (1.0-fa)*bb

							s := srcPix[base+i]
							dr := int(or - float64(s.R))
							dg := int(og - float64(s.G))
							db := int(ob - float64(s.B))
							err += dr*dr + dg*dg + db*db
						}
						if err < bestErr {
							bestErr = err
							bestT, bestFg, bestBg = ti, fg, bg
						}
					}
				}
			}
			tm.Tiles[cell] = ui.Tile{Index: int16(bestT), Color: ui.ColorIndex(bestFg), Background: ui.ColorIndex(bestBg)}
		}
	}

	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go worker()
	}
	for cell := 0; cell < n; cell++ {
		jobs <- cell
	}
	close(jobs)
	wg.Wait()

	return tm
}
