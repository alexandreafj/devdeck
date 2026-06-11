package ui

import "testing"

func TestLayoutEmpty(t *testing.T) {
	if got := Layout(100, 40, 0); len(got) != 0 {
		t.Errorf("Layout with n=0 returned %d rects, want 0", len(got))
	}
}

func TestLayoutReturnsOneRectPerWidget(t *testing.T) {
	for n := 1; n <= 4; n++ {
		if got := Layout(160, 40, n); len(got) != n {
			t.Errorf("Layout(160,40,%d) returned %d rects, want %d", n, len(got), n)
		}
	}
}

func TestLayoutWideSingleRow(t *testing.T) {
	// 200 wide / 40 min col = 5 cols, capped to n=4 → one row of 4.
	got := Layout(200, 50, 4)
	for i, r := range got {
		if r.Y != 0 {
			t.Errorf("rect[%d].Y = %d, want 0 (single row)", i, r.Y)
		}
		if r.Width != 50 {
			t.Errorf("rect[%d].Width = %d, want 50", i, r.Width)
		}
		if wantX := i * 50; r.X != wantX {
			t.Errorf("rect[%d].X = %d, want %d", i, r.X, wantX)
		}
		if r.Height != 50 {
			t.Errorf("rect[%d].Height = %d, want 50", i, r.Height)
		}
	}
}

func TestLayoutMediumTwoByTwo(t *testing.T) {
	// 100 wide / 40 = 2 cols, 4 widgets → 2 rows of 2.
	got := Layout(100, 40, 4)
	want := []Rect{
		{X: 0, Y: 0, Width: 50, Height: 20},
		{X: 50, Y: 0, Width: 50, Height: 20},
		{X: 0, Y: 20, Width: 50, Height: 20},
		{X: 50, Y: 20, Width: 50, Height: 20},
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("rect[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestLayoutNarrowStacks(t *testing.T) {
	// 30 wide < 40 min col → single column, widgets stacked.
	got := Layout(30, 40, 3)
	for i, r := range got {
		if r.X != 0 {
			t.Errorf("rect[%d].X = %d, want 0 (stacked)", i, r.X)
		}
		if r.Width != 30 {
			t.Errorf("rect[%d].Width = %d, want 30", i, r.Width)
		}
	}
	if got[0].Y != 0 || got[1].Y <= got[0].Y || got[2].Y <= got[1].Y {
		t.Errorf("stacked Y values not increasing: %d, %d, %d", got[0].Y, got[1].Y, got[2].Y)
	}
}
