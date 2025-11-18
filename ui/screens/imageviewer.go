package screens

import (
	"fmt"
	"image"
	"log"

	"github.com/SvenDH/go-card-engine/ui"
	"github.com/anthonynsimon/bild/transform"
	"github.com/hajimehoshi/ebiten/v2"
)

// ImageViewer is a screen for viewing and manipulating images as tilemaps
type ImageViewer struct {
	source      *ui.Source
	originalImg image.Image
	panX, panY  int     // Pan offset in pixels
	zoom        float64 // Zoom level
	tmWidth     int     // Tilemap width in tiles
	tmHeight    int     // Tilemap height in tiles
	numColors       int     // Number of palette colors to search
	blurRadius      float64 // Gaussian blur radius
	edgeWeight      float64 // Edge detection weight
	embeddingWeight float64 // Embedding distance weight
	embedder        ui.Embedder // Embedder for computing cell embeddings
	saveOutput      string  // Path to save the tilemap
	statusMsg   string  // Status message to display
	statusTicks int     // Ticks to display status message
}

// NewImageViewer creates a new image viewer screen
func NewImageViewer(img image.Image, width, height, numColors int, blurRadius, edgeWeight, embeddingWeight float64, saveOutput string) *ImageViewer {
	return &ImageViewer{
		originalImg:     img,
		tmWidth:         width,
		tmHeight:        height,
		numColors:       numColors,
		blurRadius:      blurRadius,
		edgeWeight:      edgeWeight,
		embeddingWeight: embeddingWeight,
		saveOutput:      saveOutput,
		zoom:            1.0,
		panX:            0,
		panY:            0,
	}
}

// SetEmbedder sets the embedder for computing cell embeddings
func (iv *ImageViewer) SetEmbedder(embedder ui.Embedder) {
	iv.embedder = embedder
}

// Init initializes the image viewer
func (iv *ImageViewer) Init() ui.Cmd {
	// Create the tilemap image
	tm := ui.NewImage(iv.tmWidth, iv.tmHeight, nil)

	// Initialize source with the transformed image
	iv.source = ui.NewSource(tm, iv.getTransformedImage(), iv.numColors, iv.blurRadius, iv.edgeWeight, iv.embeddingWeight)
	
	// Set embedder on source if available
	if iv.embedder != nil {
		iv.source.SetEmbedder(iv.embedder)
	}
	
	return iv.source.Init()
}

// getTransformedImage applies pan and zoom to the original image
func (iv *ImageViewer) getTransformedImage() image.Image {
	bounds := iv.originalImg.Bounds()

	// Calculate the target size based on zoom
	targetWidth := int(float64(bounds.Dx()) * iv.zoom)
	targetHeight := int(float64(bounds.Dy()) * iv.zoom)

	// Resize the image
	resized := transform.Resize(iv.originalImg, targetWidth, targetHeight, transform.Lanczos)

	// Calculate crop bounds based on pan offset
	cropWidth := iv.tmWidth * ui.TileSize
	cropHeight := iv.tmHeight * ui.TileSize

	// Create a canvas of the exact desired size
	canvas := image.NewRGBA(image.Rect(0, 0, cropWidth, cropHeight))

	// Calculate the source rectangle (intersection with resized image bounds)
	resizedBounds := resized.Bounds()
	srcRect := image.Rect(
		iv.panX,
		iv.panY,
		iv.panX+cropWidth,
		iv.panY+cropHeight,
	).Intersect(resizedBounds)

	// Copy the valid portion of the image to the canvas
	// Canvas coordinates are offset by how much the source rect differs from desired crop
	for y := srcRect.Min.Y; y < srcRect.Max.Y; y++ {
		for x := srcRect.Min.X; x < srcRect.Max.X; x++ {
			canvasX := x - iv.panX
			canvasY := y - iv.panY
			canvas.Set(canvasX, canvasY, resized.At(x, y))
		}
	}

	return canvas
}

// Update handles input and updates the viewer
func (iv *ImageViewer) Update(msg ui.Msg) (ui.Model, ui.Cmd) {
	switch m := msg.(type) {
	case ui.KeyEvent:
		if !m.Pressed {
			return iv, nil
		}

		panSpeed := 1 // pixels per keypress
		zoomSpeed := 0.1
		needsUpdate := false

		switch m.Key {
		// Pan controls
		case ebiten.KeyArrowLeft, ebiten.KeyA:
			iv.panX += panSpeed
			needsUpdate = true
		case ebiten.KeyArrowRight, ebiten.KeyD:
			iv.panX -= panSpeed
			needsUpdate = true
		case ebiten.KeyArrowUp, ebiten.KeyW:
			iv.panY += panSpeed
			needsUpdate = true
		case ebiten.KeyArrowDown, ebiten.KeyS:
			iv.panY -= panSpeed
			needsUpdate = true

		// Zoom controls
		case ebiten.KeyEqual, ebiten.KeyKPAdd: // + key
			iv.zoom += zoomSpeed
			if iv.zoom > 5.0 {
				iv.zoom = 5.0
			}
			needsUpdate = true
			iv.setStatus(fmt.Sprintf("Zoom: %.1fx", iv.zoom))
		case ebiten.KeyMinus, ebiten.KeyKPSubtract: // - key
			iv.zoom -= zoomSpeed
			if iv.zoom < 0.1 {
				iv.zoom = 0.1
			}
			needsUpdate = true
			iv.setStatus(fmt.Sprintf("Zoom: %.1fx", iv.zoom))

		// Reset view
		case ebiten.KeyR:
			iv.panX = 0
			iv.panY = 0
			iv.zoom = 1.0
			needsUpdate = true
			iv.setStatus("View reset")

		// Save
		case ebiten.KeySpace:
			if iv.saveOutput != "" {
				err := iv.source.View().Save(iv.saveOutput)
				if err != nil {
					log.Printf("Failed to save: %v", err)
					iv.setStatus(fmt.Sprintf("Save failed: %v", err))
				} else {
					log.Printf("Saved to %s", iv.saveOutput)
					iv.setStatus(fmt.Sprintf("Saved to %s", iv.saveOutput))
				}
			} else {
				iv.setStatus("No output path specified (use --out)")
			}
		}

		// Update the source with transformed image if needed
		if needsUpdate {
			iv.source.Image = iv.getTransformedImage()
			// Keep embedder when reinitializing
			if iv.embedder != nil {
				iv.source.SetEmbedder(iv.embedder)
			}
			iv.source.Init()
		}

	case ui.Tick:
		// Decrease status message timer
		if iv.statusTicks > 0 {
			iv.statusTicks--
		}
	}

	// Update the source
	_, cmd := iv.source.Update(msg)
	return iv, cmd
}

// setStatus sets a temporary status message
func (iv *ImageViewer) setStatus(msg string) {
	iv.statusMsg = msg
	iv.statusTicks = 180 // Show for 3 seconds at 60fps
}

// View renders the viewer
func (iv *ImageViewer) View() *ui.Image {
	// Get the source view
	return iv.source.View()
}
