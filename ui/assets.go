package ui

import (
	"fmt"
	"bytes"
	_ "embed"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"strings"
	"strconv"
	"unicode"
    "encoding/csv"
	"runtime"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	TileSize = 8
)

//go:embed assets/palette.csv
var paletteCSV []byte

//go:embed assets/tileset.png
var tilesetImage []byte

//go:embed assets/tileset.char
var tilesetChar []byte

//go:embed assets/borders.csv
var borderCSV []byte

//go:embed assets/icons.csv
var tilesCSV []byte

var (
	TilesImage  *ebiten.Image

	Palette     []ebiten.ColorScale
	// paletteRGBA stores the concrete RGBA values corresponding to palette ColorScales
	PaletteRGBA []color.NRGBA
	// tilesSource retains the original decoded tileset image for CPU-side pixel analysis
	TilesSource image.Image
	// faintMap stores the index of the faint color for each color
	FaintMap []int
	// brightMap stores the index of the bright color for each color
	BrightMap []int

	desatMap []int
	// borderMap stores the index of the border for each border
	Borders map[string]Border
	Colors map[string]ColorIndex
	TileIndices map[string]int

	runeCharMap map[rune]int
	reverseRunesMap map[int]rune

	tileCount int
	tilesPix [][]color.NRGBA
	tileEmbeddings [][]float32 // Pre-computed embeddings for all tiles
)

type workerData struct {
	s *Source
	cell int
	currTile int
}

// AssetEmbedder is the interface for computing embeddings during asset initialization
type AssetEmbedder interface {
	ComputeEmbedding(img image.Image) ([]float32, error)
}

// SetTileEmbedder computes embeddings for all tiles using the provided embedder
func SetTileEmbedder(embedder AssetEmbedder) error {
	if embedder == nil {
		return nil
	}
	
	log.Printf("DEBUG: SetTileEmbedder called, computing embeddings for %d tiles\n", len(tilesPix))
	tileEmbeddings = make([][]float32, len(tilesPix))
	successCount := 0
	for i := range tilesPix {
		// Create image from tile pixels
		tileImg := image.NewRGBA(image.Rect(0, 0, TileSize, TileSize))
		for pi, px := range tilesPix[i] {
			x := pi % TileSize
			y := pi / TileSize
			tileImg.Set(x, y, px)
		}
		
		embedding, err := embedder.ComputeEmbedding(tileImg)
		if err == nil {
			tileEmbeddings[i] = embedding
			successCount++
		} else {
			log.Printf("DEBUG: Failed to compute embedding for tile %d: %v\n", i, err)
		}
	}
	
	log.Printf("DEBUG: Computed %d tile embeddings out of %d tiles\n", successCount, len(tilesPix))
	if successCount > 0 && tileEmbeddings[0] != nil {
		log.Printf("DEBUG: First tile embedding length: %d, first 5 values: %v\n", len(tileEmbeddings[0]), tileEmbeddings[0][:5])
	}
	
	return nil
}

var (
	jobs chan workerData
	wg sync.WaitGroup
)

func init() {
	workerCount := runtime.GOMAXPROCS(0)
	if workerCount < 1 {
		workerCount = 1
	}
	jobs = make(chan workerData, workerCount*2)
	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go worker()
	}

    records, err := csv.NewReader(bytes.NewReader(paletteCSV)).ReadAll()
    if err != nil {
        log.Fatal("Unable to parse palette file as CSV", err)
    }
	Colors = make(map[string]ColorIndex, len(records)-1)
	for i, record := range records {
		if i < 1 {
			continue
		}
		r, g, b, err := hexToRGB(record[0])
		if err != nil {
			log.Fatal("Unable to parse color on line ", i, err)
		}
		cs := ebiten.ColorScale{}
		cs.SetR(float32(r) / 255.)
		cs.SetG(float32(g) / 255.)
		cs.SetB(float32(b) / 255.)
		Palette = append(Palette, cs)
		PaletteRGBA = append(PaletteRGBA, color.NRGBA{R: r, G: g, B: b, A: 255})

		fc, err := strconv.Atoi(record[2])
		if err != nil {
			log.Fatal("Unable to parse faint color index on line ", i, err)
		}
		FaintMap = append(FaintMap, fc)
		bc, err := strconv.Atoi(record[3])
		if err != nil {
			log.Fatal("Unable to parse bright color index on line ", i, err)
		}
		BrightMap = append(BrightMap, bc)
		dc, err := strconv.Atoi(record[4])
		if err != nil {
			log.Fatal("Unable to parse desaturated color index on line ", i, err)
		}
		desatMap = append(desatMap, dc)
		Colors[record[5]] = ColorIndex(i-1)
	}

    records, err = csv.NewReader(bytes.NewReader(borderCSV)).ReadAll()
    if err != nil {
        log.Fatal("Unable to parse border file as CSV", err)
    }
	Borders = make(map[string]Border, len(records)-1)
	for i, record := range records {
		if i < 1 {
			continue
		}
		Borders[record[0]] = parseBorder(record[1:])
	}

	img, img2, err := ebitenutil.NewImageFromReader(bytes.NewReader(tilesetImage))
	if err != nil {
		log.Fatal(err)
	}
	TilesImage = ebiten.NewImageFromImage(img)
	TilesSource = img2

	runeCharMap = map[rune]int{' ': 0}
	reverseRunesMap = make(map[int]rune)
	i := 0
	for j, r := range strings.Split(string(tilesetChar), "\n") {
		if r == "// end" {
			break
		}
		if j >= 3 {
			for _, c := range r {
				if c != ' ' {
					runeCharMap[c] = i
					runeCharMap[unicode.ToLower(c)] = i
					reverseRunesMap[i] = c
				}
				i++
			}
		}
	}

	records, err = csv.NewReader(bytes.NewReader(tilesCSV)).ReadAll()
	if err != nil {
		log.Fatal("Unable to parse tiles file as CSV", err)
	}
	TileIndices = make(map[string]int, len(records)-1)
	for i, record := range records {
		if i < 1 {
			continue
		}
		// CSV format: id,name,x,y
		x, err := strconv.Atoi(record[2])
		if err != nil {
			log.Fatal("Unable to parse tile x on line ", i, err)
		}
		y, err := strconv.Atoi(record[3])
		if err != nil {
			log.Fatal("Unable to parse tile y on line ", i, err)
		}
		tileName := record[1]
		TileIndices[tileName] = x + y * (TilesImage.Bounds().Dx() / TileSize)
	}

	tw := TilesSource.Bounds().Dx() / TileSize
	th := TilesSource.Bounds().Dy() / TileSize
	tileCount = tw * th
	// Pre-extract all tiles' pixels from tilesSource into memory
	tilesPix = make([][]color.NRGBA, tileCount)
	for ti := range tileCount {
		sx := (ti % tw) * TileSize
		sy := (ti / tw) * TileSize
		tilesPix[ti] = make([]color.NRGBA, TileSize * TileSize)
		idx := 0
		for y := 0; y < TileSize; y++ {
			for x := 0; x < TileSize; x++ {
				c := color.NRGBAModel.Convert(TilesSource.At(sx+x, sy+y)).(color.NRGBA)
				tilesPix[ti][idx] = c
				idx++
			}
		}
	}
}

func hexToRGB(hex string) (r, g, b uint8, err error) {
	// Remove "#" if present
	hex = strings.TrimPrefix(hex, "#")

	// Must be exactly 6 characters
	if len(hex) != 6 {
		return 0, 0, 0, fmt.Errorf("invalid hex color: %s", hex)
	}

	// Parse each two-digit pair as base 16
	r64, err := strconv.ParseUint(hex[0:2], 16, 8)
	if err != nil {
		return
	}
	g64, err := strconv.ParseUint(hex[2:4], 16, 8)
	if err != nil {
		return
	}
	b64, err := strconv.ParseUint(hex[4:6], 16, 8)
	if err != nil {
		return
	}

	return uint8(r64), uint8(g64), uint8(b64), nil
}

func parseBorder(records []string) Border {		
	border := Border{}
	topLeft, _ := strconv.Atoi(records[0])
	if topLeft != 0 {
		border.TopLeft = []int{topLeft}
	}
	top, _ := strconv.Atoi(records[1])
	if top != 0 {
		border.Top = []int{top}
	}
	topRight, _ := strconv.Atoi(records[2])
	if topRight != 0 {
		border.TopRight = []int{topRight}
	}
	left, _ := strconv.Atoi(records[3])
	if left != 0 {
		border.Left = []int{left}
	}
	right, _ := strconv.Atoi(records[5])
	if right != 0 {
		border.Right = []int{right}
	}
	bottomLeft, _ := strconv.Atoi(records[6])
	if bottomLeft != 0 {
		border.BottomLeft = []int{bottomLeft}
	}
	bottom, _ := strconv.Atoi(records[7])
	if bottom != 0 {
		border.Bottom = []int{bottom}
	}
	bottomRight, _ := strconv.Atoi(records[8])
	if bottomRight != 0 {
		border.BottomRight = []int{bottomRight}
	}
	return border
}