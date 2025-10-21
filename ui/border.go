package ui

type Border struct {
	Top          []int
	Bottom       []int
	Left         []int
	Right        []int
	TopLeft      []int
	TopRight     []int
	BottomLeft   []int
	BottomRight  []int
	MiddleLeft   []int
	MiddleRight  []int
	Middle       []int
	MiddleTop    []int
	MiddleBottom []int
}

// isEmpty checks if the border is empty (all slices are nil)
func (b Border) isEmpty() bool {
	return b.Top == nil && b.Bottom == nil && b.Left == nil && b.Right == nil &&
		b.TopLeft == nil && b.TopRight == nil && b.BottomLeft == nil && b.BottomRight == nil &&
		b.MiddleLeft == nil && b.MiddleRight == nil && b.Middle == nil &&
		b.MiddleTop == nil && b.MiddleBottom == nil
}

func (s Style) applyBorder(tm *Image) *Image {
	var (
		border    = s.getBorderStyle()
		hasTop    = s.getAsBool(borderTopKey, false)
		hasRight  = s.getAsBool(borderRightKey, false)
		hasBottom = s.getAsBool(borderBottomKey, false)
		hasLeft   = s.getAsBool(borderLeftKey, false)

		topFG    = s.getAsColor(borderTopForegroundKey)
		rightFG  = s.getAsColor(borderRightForegroundKey)
		bottomFG = s.getAsColor(borderBottomForegroundKey)
		leftFG   = s.getAsColor(borderLeftForegroundKey)

		topBG    = s.getAsColor(borderTopBackgroundKey)
		rightBG  = s.getAsColor(borderRightBackgroundKey)
		bottomBG = s.getAsColor(borderBottomBackgroundKey)
		leftBG   = s.getAsColor(borderLeftBackgroundKey)
	)

	// If a border is set and no sides have been specifically turned on or off
	// render borders on all sides.
	if s.implicitBorders() {
		hasTop = true
		hasRight = true
		hasBottom = true
		hasLeft = true
	}

	// If no border is set or all borders are been disabled, abort.
	if border.isEmpty() || (!hasTop && !hasRight && !hasBottom && !hasLeft) {
		return tm
	}

	lines := tm.getRows()
	width := tm.W
	var xOffset, yOffset int
	if hasLeft {
		if border.Left == nil {
			border.Left = []int{0}
		}
		width += 1
		xOffset = 1
	}

	if hasRight && border.Right == nil {
		border.Right = []int{0}
	}

	if hasTop && hasLeft && border.TopLeft == nil {
		border.TopLeft = []int{0}
	}
	if hasTop && hasRight && border.TopRight == nil {
		border.TopRight = []int{0}
	}
	if hasBottom && hasLeft && border.BottomLeft == nil {
		border.BottomLeft = []int{0}
	}
	if hasBottom && hasRight && border.BottomRight == nil {
		border.BottomRight = []int{0}
	}

	// Figure out which corners we should actually be using based on which
	// sides are set to show.
	if hasTop {
		switch {
		case !hasLeft && !hasRight:
			border.TopLeft = []int{}
			border.TopRight = []int{}
		case !hasLeft:
			border.TopLeft = []int{}
		case !hasRight:
			border.TopRight = []int{}
		}
	}
	if hasBottom {
		switch {
		case !hasLeft && !hasRight:
			border.BottomLeft = []int{}
			border.BottomRight = []int{}
		case !hasLeft:
			border.BottomLeft = []int{}
		case !hasRight:
			border.BottomRight = []int{}
		}
	}

	var out Tiles
	height := 0
	// Render top
	if hasTop {
		yOffset = 1
		top := renderHorizontalEdge(border.TopLeft, border.Top, border.TopRight, width)
		out = append(out, s.styleBorder(top, topFG, topBG)...)
		height++
	}

	leftIndex := 0
	rightIndex := 0

	// Render sides
	for _, l := range lines {
		if hasLeft {
			r := []int{border.Left[leftIndex]}
			leftIndex++
			if leftIndex >= len(border.Left) {
				leftIndex = 0
			}
			out = append(out, s.styleBorder(r, leftFG, leftBG)...)
		}
		out = append(out, l...)
		if hasRight {
			r := []int{border.Right[rightIndex]}
			rightIndex++
			if rightIndex >= len(border.Right) {
				rightIndex = 0
			}
			out = append(out, s.styleBorder(r, rightFG, rightBG)...)
		}
		height++
	}

	// Render bottom
	if hasBottom {
		bottom := renderHorizontalEdge(border.BottomLeft, border.Bottom, border.BottomRight, width)
		out = append(out, s.styleBorder(bottom, bottomFG, bottomBG)...)
		height++
	}
	newZones := make([]*Zone, len(tm.Zones))
	for i, z := range tm.Zones {
		newZones[i] = z.Add(xOffset, yOffset)
	}

	return &Image{
		W:     len(out) / height,
		H:     height,
		Tiles: out,
		Zones: newZones,
	}
}

func renderHorizontalEdge(left, middle, right []int, width int) []int {
	if middle == nil {
		middle = []int{0}
	}
	j := 0
	var out []int
	out = append(out, left...)
	for i := len(left) + len(right); i < width+len(right); i++ {
		out = append(out, middle[j])
		j++
		if j >= len(middle) {
			j = 0
		}
	}
	return append(out, right...)
}

func (s Style) styleBorder(border []int, fg, bg ColorIndex) Tiles {
	tiles := make(Tiles, len(border))
	for i := range border {
		tiles[i].Index = int16(border[i])
	}
	if fg == noColor && bg == noColor {
		return tiles
	}
	if fg != noColor {
		tiles.Color(fg)
	}
	if bg != noColor {
		tiles.Background(bg)
	}
	return tiles
}

func (s Style) getBorderStyle() Border {
	if !s.isSet(borderStyleKey) {
		return Border{}
	}
	return s.borderStyle
}

func (s Style) implicitBorders() bool {
	var (
		borderStyle = s.getBorderStyle()
		topSet      = s.isSet(borderTopKey)
		rightSet    = s.isSet(borderRightKey)
		bottomSet   = s.isSet(borderBottomKey)
		leftSet     = s.isSet(borderLeftKey)
	)
	return !borderStyle.isEmpty() && !(topSet || rightSet || bottomSet || leftSet)
}
