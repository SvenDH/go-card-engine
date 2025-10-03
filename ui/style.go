package ui

type propKey int64

const (
	noColor ColorIndex = -1

	// Boolean props come first.
	boldKey propKey = 1 << iota
	italicKey
	underlineKey
	strikethroughKey
	reverseKey
	faintKey

	// Non-boolean props.
	foregroundKey
	backgroundKey
	widthKey
	heightKey
	alignHorizontalKey
	alignVerticalKey

	// Padding.
	paddingTopKey
	paddingRightKey
	paddingBottomKey
	paddingLeftKey

	// Margins.
	marginTopKey
	marginRightKey
	marginBottomKey
	marginLeftKey
	marginBackgroundKey

	// Border runes.
	borderStyleKey

	// Border edges.
	borderTopKey
	borderRightKey
	borderBottomKey
	borderLeftKey

	// Border foreground colors.
	borderTopForegroundKey
	borderRightForegroundKey
	borderBottomForegroundKey
	borderLeftForegroundKey

	// Border background colors.
	borderTopBackgroundKey
	borderRightBackgroundKey
	borderBottomBackgroundKey
	borderLeftBackgroundKey

	inlineKey
	maxWidthKey
	maxHeightKey
	tabWidthKey

	transformKey
)

// Style contains a set of rules that comprise a style as a whole.
type Style struct {
	props props

	// we store bool props values here
	attrs int

	// props that have values
	fgColor ColorIndex
	bgColor ColorIndex

	width  int
	height int

	alignHorizontal Position
	alignVertical   Position

	paddingTop    int
	paddingRight  int
	paddingBottom int
	paddingLeft   int

	marginTop     int
	marginRight   int
	marginBottom  int
	marginLeft    int
	marginBgColor ColorIndex

	borderStyle         Border
	borderTopFgColor    ColorIndex
	borderRightFgColor  ColorIndex
	borderBottomFgColor ColorIndex
	borderLeftFgColor   ColorIndex
	borderTopBgColor    ColorIndex
	borderRightBgColor  ColorIndex
	borderBottomBgColor ColorIndex
	borderLeftBgColor   ColorIndex

	maxWidth  int
	maxHeight int
	tabWidth  int

	transform func(*TileMap) *TileMap
}

func NewStyle() Style {
	return Style{}
}

// Bold sets a bold formatting rule.
func (s Style) Bold(v bool) Style {
	s.set(boldKey, v)
	return s
}

// Italic sets an italic formatting rule. In some terminal emulators this will
// render with "reverse" coloring if not italic font variant is available.
func (s Style) Italic(v bool) Style {
	s.set(italicKey, v)
	return s
}

// Underline sets an underline rule. By default, underlines will not be drawn on
// whitespace like margins and padding. To change this behavior set
// UnderlineSpaces.
func (s Style) Underline(v bool) Style {
	s.set(underlineKey, v)
	return s
}

// Strikethrough sets a strikethrough rule. By default, strikes will not be
// drawn on whitespace like margins and padding. To change this behavior set
// StrikethroughSpaces.
func (s Style) Strikethrough(v bool) Style {
	s.set(strikethroughKey, v)
	return s
}

// Reverse sets a rule for inverting foreground and background colors.
func (s Style) Reverse(v bool) Style {
	s.set(reverseKey, v)
	return s
}

// Faint sets a rule for rendering the foreground color in a dimmer shade.
func (s Style) Faint(v bool) Style {
	s.set(faintKey, v)
	return s
}

// Foreground sets a foreground color.
//
//	// Sets the foreground to blue
//	s := lipgloss.NewStyle().Foreground(lipgloss.Color("#0000ff"))
//
//	// Removes the foreground color
//	s.Foreground(lipgloss.NoColor)
func (s Style) Foreground(c ColorIndex) Style {
	s.set(foregroundKey, c)
	return s
}

// Background sets a background color.
func (s Style) Background(c ColorIndex) Style {
	s.set(backgroundKey, c)
	return s
}

// Width sets the width of the block before applying margins. The width, if
// set, also determines where text will wrap.
func (s Style) Width(i int) Style {
	s.set(widthKey, i)
	return s
}

// Height sets the height of the block before applying margins. If the height of
// the text block is less than this value after applying padding (or not), the
// block will be set to this height.
func (s Style) Height(i int) Style {
	s.set(heightKey, i)
	return s
}

// Align is a shorthand method for setting horizontal and vertical alignment.
//
// With one argument, the position value is applied to the horizontal alignment.
//
// With two arguments, the value is applied to the horizontal and vertical
// alignments, in that order.
func (s Style) Align(p ...Position) Style {
	if len(p) > 0 {
		s.set(alignHorizontalKey, p[0])
	}
	if len(p) > 1 {
		s.set(alignVerticalKey, p[1])
	}
	return s
}

// AlignHorizontal sets a horizontal text alignment rule.
func (s Style) AlignHorizontal(p Position) Style {
	s.set(alignHorizontalKey, p)
	return s
}

// AlignVertical sets a vertical text alignment rule.
func (s Style) AlignVertical(p Position) Style {
	s.set(alignVerticalKey, p)
	return s
}

// Padding is a shorthand method for setting padding on all sides at once.
//
// With one argument, the value is applied to all sides.
//
// With two arguments, the value is applied to the vertical and horizontal
// sides, in that order.
//
// With three arguments, the value is applied to the top side, the horizontal
// sides, and the bottom side, in that order.
//
// With four arguments, the value is applied clockwise starting from the top
// side, followed by the right side, then the bottom, and finally the left.
//
// With more than four arguments no padding will be added.
func (s Style) Padding(i ...int) Style {
	top, right, bottom, left, ok := whichSidesInt(i...)
	if !ok {
		return s
	}

	s.set(paddingTopKey, top)
	s.set(paddingRightKey, right)
	s.set(paddingBottomKey, bottom)
	s.set(paddingLeftKey, left)
	return s
}

// PaddingLeft adds padding on the left.
func (s Style) PaddingLeft(i int) Style {
	s.set(paddingLeftKey, i)
	return s
}

// PaddingRight adds padding on the right.
func (s Style) PaddingRight(i int) Style {
	s.set(paddingRightKey, i)
	return s
}

// PaddingTop adds padding to the top of the block.
func (s Style) PaddingTop(i int) Style {
	s.set(paddingTopKey, i)
	return s
}

// PaddingBottom adds padding to the bottom of the block.
func (s Style) PaddingBottom(i int) Style {
	s.set(paddingBottomKey, i)
	return s
}

// Margin is a shorthand method for setting margins on all sides at once.
//
// With one argument, the value is applied to all sides.
//
// With two arguments, the value is applied to the vertical and horizontal
// sides, in that order.
//
// With three arguments, the value is applied to the top side, the horizontal
// sides, and the bottom side, in that order.
//
// With four arguments, the value is applied clockwise starting from the top
// side, followed by the right side, then the bottom, and finally the left.
//
// With more than four arguments no margin will be added.
func (s Style) Margin(i ...int) Style {
	top, right, bottom, left, ok := whichSidesInt(i...)
	if !ok {
		return s
	}

	s.set(marginTopKey, top)
	s.set(marginRightKey, right)
	s.set(marginBottomKey, bottom)
	s.set(marginLeftKey, left)
	return s
}

// MarginLeft sets the value of the left margin.
func (s Style) MarginLeft(i int) Style {
	s.set(marginLeftKey, i)
	return s
}

// MarginRight sets the value of the right margin.
func (s Style) MarginRight(i int) Style {
	s.set(marginRightKey, i)
	return s
}

// MarginTop sets the value of the top margin.
func (s Style) MarginTop(i int) Style {
	s.set(marginTopKey, i)
	return s
}

// MarginBottom sets the value of the bottom margin.
func (s Style) MarginBottom(i int) Style {
	s.set(marginBottomKey, i)
	return s
}

// MarginBackground sets the background color of the margin. Note that this is
// also set when inheriting from a style with a background color. In that case
// the background color on that style will set the margin color on this style.
func (s Style) MarginBackground(c ColorIndex) Style {
	s.set(marginBackgroundKey, c)
	return s
}

// Border is shorthand for setting the border style and which sides should
// have a border at once. The variadic argument sides works as follows:
//
// With one value, the value is applied to all sides.
//
// With two values, the values are applied to the vertical and horizontal
// sides, in that order.
//
// With three values, the values are applied to the top side, the horizontal
// sides, and the bottom side, in that order.
//
// With four values, the values are applied clockwise starting from the top
// side, followed by the right side, then the bottom, and finally the left.
//
// With more than four arguments the border will be applied to all sides.
//
// Examples:
//
//	// Applies borders to the top and bottom only
//	lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true, false)
//
//	// Applies rounded borders to the right and bottom only
//	lipgloss.NewStyle().Border(lipgloss.RoundedBorder(), false, true, true, false)
func (s Style) Border(b Border, sides ...bool) Style {
	s.set(borderStyleKey, b)

	top, right, bottom, left, ok := whichSidesBool(sides...)
	if !ok {
		top = true
		right = true
		bottom = true
		left = true
	}

	s.set(borderTopKey, top)
	s.set(borderRightKey, right)
	s.set(borderBottomKey, bottom)
	s.set(borderLeftKey, left)

	return s
}

// BorderStyle defines the Border on a style. A Border contains a series of
// definitions for the sides and corners of a border.
//
// Note that if border visibility has not been set for any sides when setting
// the border style, the border will be enabled for all sides during rendering.
//
// You can define border characters as you'd like, though several default
// styles are included: NormalBorder(), RoundedBorder(), BlockBorder(),
// OuterHalfBlockBorder(), InnerHalfBlockBorder(), ThickBorder(),
// and DoubleBorder().
//
// Example:
//
//	lipgloss.NewStyle().BorderStyle(lipgloss.ThickBorder())
func (s Style) BorderStyle(b Border) Style {
	s.set(borderStyleKey, b)
	return s
}

// BorderTop determines whether or not to draw a top border.
func (s Style) BorderTop(v bool) Style {
	s.set(borderTopKey, v)
	return s
}

// BorderRight determines whether or not to draw a right border.
func (s Style) BorderRight(v bool) Style {
	s.set(borderRightKey, v)
	return s
}

// BorderBottom determines whether or not to draw a bottom border.
func (s Style) BorderBottom(v bool) Style {
	s.set(borderBottomKey, v)
	return s
}

// BorderLeft determines whether or not to draw a left border.
func (s Style) BorderLeft(v bool) Style {
	s.set(borderLeftKey, v)
	return s
}

// BorderForeground is a shorthand function for setting all of the
// foreground colors of the borders at once. The arguments work as follows:
//
// With one argument, the argument is applied to all sides.
//
// With two arguments, the arguments are applied to the vertical and horizontal
// sides, in that order.
//
// With three arguments, the arguments are applied to the top side, the
// horizontal sides, and the bottom side, in that order.
//
// With four arguments, the arguments are applied clockwise starting from the
// top side, followed by the right side, then the bottom, and finally the left.
//
// With more than four arguments nothing will be set.
func (s Style) BorderForeground(c ...ColorIndex) Style {
	if len(c) == 0 {
		return s
	}

	top, right, bottom, left, ok := whichSidesColor(c...)
	if !ok {
		return s
	}

	s.set(borderTopForegroundKey, top)
	s.set(borderRightForegroundKey, right)
	s.set(borderBottomForegroundKey, bottom)
	s.set(borderLeftForegroundKey, left)

	return s
}

// BorderTopForeground set the foreground color for the top of the border.
func (s Style) BorderTopForeground(c ColorIndex) Style {
	s.set(borderTopForegroundKey, c)
	return s
}

// BorderRightForeground sets the foreground color for the right side of the
// border.
func (s Style) BorderRightForeground(c ColorIndex) Style {
	s.set(borderRightForegroundKey, c)
	return s
}

// BorderBottomForeground sets the foreground color for the bottom of the
// border.
func (s Style) BorderBottomForeground(c ColorIndex) Style {
	s.set(borderBottomForegroundKey, c)
	return s
}

// BorderLeftForeground sets the foreground color for the left side of the
// border.
func (s Style) BorderLeftForeground(c ColorIndex) Style {
	s.set(borderLeftForegroundKey, c)
	return s
}

// BorderBackground is a shorthand function for setting all of the
// background colors of the borders at once. The arguments work as follows:
//
// With one argument, the argument is applied to all sides.
//
// With two arguments, the arguments are applied to the vertical and horizontal
// sides, in that order.
//
// With three arguments, the arguments are applied to the top side, the
// horizontal sides, and the bottom side, in that order.
//
// With four arguments, the arguments are applied clockwise starting from the
// top side, followed by the right side, then the bottom, and finally the left.
//
// With more than four arguments nothing will be set.
func (s Style) BorderBackground(c ...ColorIndex) Style {
	if len(c) == 0 {
		return s
	}

	top, right, bottom, left, ok := whichSidesColor(c...)
	if !ok {
		return s
	}

	s.set(borderTopBackgroundKey, top)
	s.set(borderRightBackgroundKey, right)
	s.set(borderBottomBackgroundKey, bottom)
	s.set(borderLeftBackgroundKey, left)

	return s
}

// BorderTopBackground sets the background color of the top of the border.
func (s Style) BorderTopBackground(c ColorIndex) Style {
	s.set(borderTopBackgroundKey, c)
	return s
}

// BorderRightBackground sets the background color of right side the border.
func (s Style) BorderRightBackground(c ColorIndex) Style {
	s.set(borderRightBackgroundKey, c)
	return s
}

// BorderBottomBackground sets the background color of the bottom of the
// border.
func (s Style) BorderBottomBackground(c ColorIndex) Style {
	s.set(borderBottomBackgroundKey, c)
	return s
}

// BorderLeftBackground set the background color of the left side of the
// border.
func (s Style) BorderLeftBackground(c ColorIndex) Style {
	s.set(borderLeftBackgroundKey, c)
	return s
}

// Inline makes rendering output one line and disables the rendering of
// margins, padding and borders. This is useful when you need a style to apply
// only to font rendering and don't want it to change any physical dimensions.
// It works well with Style.MaxWidth.
//
// Because this in intended to be used at the time of render, this method will
// not mutate the style and instead return a copy.
//
// Example:
//
//	var userInput string = "..."
//	var userStyle = text.Style{ /* ... */ }
//	fmt.Println(userStyle.Inline(true).Render(userInput))
func (s Style) Inline(v bool) Style {
	o := s // copy
	o.set(inlineKey, v)
	return o
}

// MaxWidth applies a max width to a given style. This is useful in enforcing
// a certain width at render time, particularly with arbitrary strings and
// styles.
//
// Because this in intended to be used at the time of render, this method will
// not mutate the style and instead return a copy.
//
// Example:
//
//	var userInput string = "..."
//	var userStyle = text.Style{ /* ... */ }
//	fmt.Println(userStyle.MaxWidth(16).Render(userInput))
func (s Style) MaxWidth(n int) Style {
	o := s // copy
	o.set(maxWidthKey, n)
	return o
}

// MaxHeight applies a max height to a given style. This is useful in enforcing
// a certain height at render time, particularly with arbitrary strings and
// styles.
//
// Because this in intended to be used at the time of render, this method will
// not mutate the style and instead returns a copy.
func (s Style) MaxHeight(n int) Style {
	o := s // copy
	o.set(maxHeightKey, n)
	return o
}

// NoTabConversion can be passed to [Style.TabWidth] to disable the replacement
// of tabs with spaces at render time.
const NoTabConversion = -1

// TabWidth sets the number of spaces that a tab (/t) should be rendered as.
// When set to 0, tabs will be removed. To disable the replacement of tabs with
// spaces entirely, set this to [NoTabConversion].
//
// By default, tabs will be replaced with 4 spaces.
func (s Style) TabWidth(n int) Style {
	if n <= -1 {
		n = -1
	}
	s.set(tabWidthKey, n)
	return s
}

// Transform applies a given function to a string at render time, allowing for
// the string being rendered to be manipuated.
//
// Example:
//
//	s := NewStyle().Transform(strings.ToUpper)
//	fmt.Println(s.Render("raow!") // "RAOW!"
func (s Style) Transform(fn func(string) string) Style {
	s.set(transformKey, fn)
	return s
}

// Render applies the defined style formatting to a given string.
func (s Style) Render(tm *TileMap) *TileMap {
	var (
		width           = s.getAsInt(widthKey)
		height          = s.getAsInt(heightKey)
		horizontalAlign = s.getAsPosition(alignHorizontalKey)
		verticalAlign   = s.getAsPosition(alignVerticalKey)

		topPadding    = s.getAsInt(paddingTopKey)
		rightPadding  = s.getAsInt(paddingRightKey)
		bottomPadding = s.getAsInt(paddingBottomKey)
		leftPadding   = s.getAsInt(paddingLeftKey)

		inline    = s.getAsBool(inlineKey, false)
		maxWidth  = s.getAsInt(maxWidthKey)
		maxHeight = s.getAsInt(maxHeightKey)

		transform = s.getAsTransform(transformKey)
	)
	if transform != nil {
		tm = transform(tm)
	}
	if s.props == 0 {
		return tm
	}
	// Strip newlines in single line mode
	if inline {
		tm = &TileMap{W: tm.W * tm.H, H: 1, Tiles: tm.Tiles}
	}

	// Word wrap
	//if !inline && width > 0 {
	//	wrapAt := width - leftPadding - rightPadding
	//	str = cellbuf.Wrap(str, wrapAt, "")
	//}

	// Render core text
	tm.Tiles.Style(s)

	// Padding
	if !inline {
		if leftPadding > 0 {
			tm = tm.ExtendLeft(leftPadding, s)
		}
		if rightPadding > 0 {
			tm = tm.ExtendRight(rightPadding, s)
		}
		if topPadding > 0 {
			tm = tm.ExtendTop(topPadding, s)
		}
		if bottomPadding > 0 {
			tm = tm.ExtendBottom(bottomPadding, s)
		}
	}

	// Height
	if height > 0 {
		tm = alignTextVertical(tm, verticalAlign, height, s)
	}

	// Set alignment. This will also pad short lines with spaces so that all
	// lines are the same length, so we run it under a few different conditions
	// beyond alignment.
	{
		numLines := tm.H
		if numLines > 1 || width != 0 {
			tm = alignTextHorizontal(tm, horizontalAlign, width, &s)
		}
	}

	if !inline {
		tm = s.applyBorder(tm)
		tm = s.applyMargins(tm, inline)
	}

	// Truncate according to MaxWidth
	if maxWidth > 0 {
		var tiles Tiles
		for _, l := range tm.getRows() {
			tiles = append(tiles, l[:maxWidth]...)
		}
		tm = &TileMap{W: maxWidth, H: tm.H, Tiles: tiles, Zones: tm.Zones}
	}

	// Truncate according to MaxHeight
	if maxHeight > 0 {
		height := min(maxHeight, tm.H)
		if tm.H > 0 {
			tm = &TileMap{W: tm.W, H: height, Tiles: tm.Tiles[:height*tm.W], Zones: tm.Zones}
		}
	}

	return tm
}

func alignTextHorizontal(tm *TileMap, pos Position, width int, style *Style) *TileMap {
	var tiles Tiles
	for _, l := range tm.getRows() {
		lineWidth := l.StringLength()

		shortAmount := tm.W - lineWidth                      // difference from the widest line
		shortAmount += max(0, width-(shortAmount+lineWidth)) // difference from the total width, if set

		if shortAmount > 0 {
			switch pos { //nolint:exhaustive
			case Right:
				s := make(Tiles, shortAmount)
				if style != nil {
					s = s.Style(*style)
				}
				l = append(s, l[:lineWidth]...)
			case Center:
				// Note: remainder goes on the right.
				left := shortAmount / 2       //nolint:mnd
				right := left + shortAmount%2 //nolint:mnd

				leftSpaces := make(Tiles, left)
				rightSpaces := make(Tiles, right)

				if style != nil {
					leftSpaces = leftSpaces.Style(*style)
					rightSpaces = rightSpaces.Style(*style)
				}
				l = append(leftSpaces, append(l[:lineWidth], rightSpaces...)...)
			default: // Left
				s := make(Tiles, shortAmount)
				if style != nil {
					s = s.Style(*style)
				}
				l = append(l[:lineWidth], s...)
			}
		}
		tiles = append(tiles, l...)
	}

	return &TileMap{
		W:     max(tm.W, width),
		H:     tm.H,
		Tiles: tiles,
		Zones: tm.Zones,
	}
}

func alignTextVertical(tm *TileMap, pos Position, height int, s Style) *TileMap {
	strHeight := tm.H
	if height < strHeight {
		return tm
	}
	switch pos {
	case Top:
		return tm.ExtendBottom(height-strHeight, s)
	case Center:
		topPadding, bottomPadding := (height-strHeight)/2, (height-strHeight)/2 //nolint:mnd
		if strHeight+topPadding+bottomPadding > height {
			topPadding--
		} else if strHeight+topPadding+bottomPadding < height {
			bottomPadding++
		}
		return tm.ExtendTop(topPadding, s).ExtendBottom(bottomPadding, s)
	case Bottom:
		return tm.ExtendTop(height-strHeight, s)
	}
	return tm
}

func (s Style) applyMargins(tm *TileMap, inline bool) *TileMap {
	var (
		topMargin    = s.getAsInt(marginTopKey)
		rightMargin  = s.getAsInt(marginRightKey)
		bottomMargin = s.getAsInt(marginBottomKey)
		leftMargin   = s.getAsInt(marginLeftKey)
		bgc          = s.getAsColor(marginBackgroundKey)
	)
	tm = tm.ExtendLeft(leftMargin, Style{bgColor: bgc})
	tm = tm.ExtendRight(rightMargin, Style{bgColor: bgc})

	// Top/bottom margin
	if !inline {
		tm = tm.ExtendTop(topMargin, Style{bgColor: bgc})
		tm = tm.ExtendBottom(bottomMargin, Style{bgColor: bgc})
	}
	return tm
}

func (s Style) getAsBool(k propKey, defaultVal bool) bool {
	if !s.isSet(k) {
		return defaultVal
	}
	return s.attrs&int(k) != 0
}

func (s Style) getAsColor(k propKey) ColorIndex {
	if !s.isSet(k) {
		return noColor
	}
	var c ColorIndex
	switch k { //nolint:exhaustive
	case foregroundKey:
		c = s.fgColor
	case backgroundKey:
		c = s.bgColor
	case marginBackgroundKey:
		c = s.marginBgColor
	case borderTopForegroundKey:
		c = s.borderTopFgColor
	case borderRightForegroundKey:
		c = s.borderRightFgColor
	case borderBottomForegroundKey:
		c = s.borderBottomFgColor
	case borderLeftForegroundKey:
		c = s.borderLeftFgColor
	case borderTopBackgroundKey:
		c = s.borderTopBgColor
	case borderRightBackgroundKey:
		c = s.borderRightBgColor
	case borderBottomBackgroundKey:
		c = s.borderBottomBgColor
	case borderLeftBackgroundKey:
		c = s.borderLeftBgColor
	}
	if c != -1 {
		return c
	}
	return noColor
}

func (s Style) getAsInt(k propKey) int {
	if !s.isSet(k) {
		return 0
	}
	switch k { //nolint:exhaustive
	case widthKey:
		return s.width
	case heightKey:
		return s.height
	case paddingTopKey:
		return s.paddingTop
	case paddingRightKey:
		return s.paddingRight
	case paddingBottomKey:
		return s.paddingBottom
	case paddingLeftKey:
		return s.paddingLeft
	case marginTopKey:
		return s.marginTop
	case marginRightKey:
		return s.marginRight
	case marginBottomKey:
		return s.marginBottom
	case marginLeftKey:
		return s.marginLeft
	case maxWidthKey:
		return s.maxWidth
	case maxHeightKey:
		return s.maxHeight
	case tabWidthKey:
		return s.tabWidth
	}
	return 0
}

func (s Style) getAsPosition(k propKey) Position {
	if !s.isSet(k) {
		return Position(0)
	}
	switch k { //nolint:exhaustive
	case alignHorizontalKey:
		return s.alignHorizontal
	case alignVerticalKey:
		return s.alignVertical
	}
	return Position(0)
}

func (s Style) getAsTransform(propKey) func(*TileMap) *TileMap {
	if !s.isSet(transformKey) {
		return nil
	}
	return s.transform
}

func (s *Style) set(key propKey, value interface{}) {
	// We don't allow negative integers on any of our other values, so just keep
	// them at zero or above. We could use uints instead, but the
	// conversions are a little tedious, so we're sticking with ints for
	// sake of usability.
	switch key { //nolint:exhaustive
	case foregroundKey:
		s.fgColor = value.(ColorIndex)
	case backgroundKey:
		s.bgColor = value.(ColorIndex)
	case widthKey:
		s.width = max(0, value.(int))
	case heightKey:
		s.height = max(0, value.(int))
	case alignHorizontalKey:
		s.alignHorizontal = value.(Position)
	case alignVerticalKey:
		s.alignVertical = value.(Position)
	case paddingTopKey:
		s.paddingTop = max(0, value.(int))
	case paddingRightKey:
		s.paddingRight = max(0, value.(int))
	case paddingBottomKey:
		s.paddingBottom = max(0, value.(int))
	case paddingLeftKey:
		s.paddingLeft = max(0, value.(int))
	case marginTopKey:
		s.marginTop = max(0, value.(int))
	case marginRightKey:
		s.marginRight = max(0, value.(int))
	case marginBottomKey:
		s.marginBottom = max(0, value.(int))
	case marginLeftKey:
		s.marginLeft = max(0, value.(int))
	case marginBackgroundKey:
		s.marginBgColor = value.(ColorIndex)
	case borderStyleKey:
		s.borderStyle = value.(Border)
	case borderTopForegroundKey:
		s.borderTopFgColor = value.(ColorIndex)
	case borderRightForegroundKey:
		s.borderRightFgColor = value.(ColorIndex)
	case borderBottomForegroundKey:
		s.borderBottomFgColor = value.(ColorIndex)
	case borderLeftForegroundKey:
		s.borderLeftFgColor = value.(ColorIndex)
	case borderTopBackgroundKey:
		s.borderTopBgColor = value.(ColorIndex)
	case borderRightBackgroundKey:
		s.borderRightBgColor = value.(ColorIndex)
	case borderBottomBackgroundKey:
		s.borderBottomBgColor = value.(ColorIndex)
	case borderLeftBackgroundKey:
		s.borderLeftBgColor = value.(ColorIndex)
	case maxWidthKey:
		s.maxWidth = max(0, value.(int))
	case maxHeightKey:
		s.maxHeight = max(0, value.(int))
	case tabWidthKey:
		// TabWidth is the only property that may have a negative value (and
		// that negative value can be no less than -1).
		s.tabWidth = value.(int)
	case transformKey:
		s.transform = value.(func(*TileMap) *TileMap)
	default:
		if v, ok := value.(bool); ok { //nolint:nestif
			if v {
				s.attrs |= int(key)
			} else {
				s.attrs &^= int(key)
			}
		} else if attrs, ok := value.(int); ok {
			// bool attrs
			if attrs&int(key) != 0 {
				s.attrs |= int(key)
			} else {
				s.attrs &^= int(key)
			}
		}
	}

	// Set the prop on
	s.props = s.props.set(key)
}

// Returns whether or not the given property is set.
func (s Style) isSet(k propKey) bool {
	return s.props.has(k)
}

// props is a set of properties.
type props int64

// set sets a property.
func (p props) set(k propKey) props {
	return p | props(k)
}

// unset unsets a property.
func (p props) unset(k propKey) props {
	return p &^ props(k)
}

// has checks if a property is set.
func (p props) has(k propKey) bool {
	return p&props(k) != 0
}

func whichSidesInt(i ...int) (top, right, bottom, left int, ok bool) {
	switch len(i) {
	case 1:
		top = i[0]
		bottom = i[0]
		left = i[0]
		right = i[0]
		ok = true
	case 2: //nolint:mnd
		top = i[0]
		bottom = i[0]
		left = i[1]
		right = i[1]
		ok = true
	case 3: //nolint:mnd
		top = i[0]
		left = i[1]
		right = i[1]
		bottom = i[2]
		ok = true
	case 4: //nolint:mnd
		top = i[0]
		right = i[1]
		bottom = i[2]
		left = i[3]
		ok = true
	}
	return top, right, bottom, left, ok
}

// whichSidesBool is like whichSidesInt, except it operates on a series of
// boolean values. See the comment on whichSidesInt for details on how this
// works.
func whichSidesBool(i ...bool) (top, right, bottom, left bool, ok bool) {
	switch len(i) {
	case 1:
		top = i[0]
		bottom = i[0]
		left = i[0]
		right = i[0]
		ok = true
	case 2: //nolint:mnd
		top = i[0]
		bottom = i[0]
		left = i[1]
		right = i[1]
		ok = true
	case 3: //nolint:mnd
		top = i[0]
		left = i[1]
		right = i[1]
		bottom = i[2]
		ok = true
	case 4: //nolint:mnd
		top = i[0]
		right = i[1]
		bottom = i[2]
		left = i[3]
		ok = true
	}
	return top, right, bottom, left, ok
}

// whichSidesColor is like whichSides, except it operates on a series of
// boolean values. See the comment on whichSidesInt for details on how this
// works.
func whichSidesColor(i ...ColorIndex) (top, right, bottom, left ColorIndex, ok bool) {
	switch len(i) {
	case 1:
		top = i[0]
		bottom = i[0]
		left = i[0]
		right = i[0]
		ok = true
	case 2: //nolint:mnd
		top = i[0]
		bottom = i[0]
		left = i[1]
		right = i[1]
		ok = true
	case 3: //nolint:mnd
		top = i[0]
		left = i[1]
		right = i[1]
		bottom = i[2]
		ok = true
	case 4: //nolint:mnd
		top = i[0]
		right = i[1]
		bottom = i[2]
		left = i[3]
		ok = true
	}
	return top, right, bottom, left, ok
}
