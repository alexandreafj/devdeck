package ui

// minColWidth is the narrowest a widget column may be before the layout drops
// to fewer columns. Tuned so a standard terminal shows widgets side by side and
// a narrow one stacks them.
const minColWidth = 40

// Rect is a widget's allotted area within the terminal, in cells.
type Rect struct {
	X, Y          int
	Width, Height int
}

// Layout arranges n widgets responsively within the given terminal size and
// returns one Rect per widget, in order. It is pure (no Bubble Tea state), so
// the column/row maths is exhaustively testable.
//
// The number of columns is width/minColWidth, clamped to [1, n]; rows follow.
// Wide terminals yield a single row; medium ones a grid; narrow ones a stack.
func Layout(width, height, n int) []Rect {
	if n <= 0 {
		return nil
	}

	cols := min(max(width/minColWidth, 1), n)
	rows := (n + cols - 1) / cols

	cellW := width / cols
	cellH := height / rows

	rects := make([]Rect, n)
	for i := range rects {
		r, c := i/cols, i%cols
		rects[i] = Rect{X: c * cellW, Y: r * cellH, Width: cellW, Height: cellH}
	}
	return rects
}
