package ui

import "testing"

func TestPlotPathMapsExtremes(t *testing.T) {
	// values: min=10 (should map to bottom), max=20 (top).
	pts := plotPath([]float64{10, 20, 15}, 100, 50, 0)
	if len(pts) != 3 {
		t.Fatalf("got %d points, want 3", len(pts))
	}
	if pts[0].X != 0 {
		t.Errorf("first x = %v, want 0", pts[0].X)
	}
	if pts[2].X != 100 {
		t.Errorf("last x = %v, want 100 (spans full width)", pts[2].X)
	}
	if pts[0].Y != 50 {
		t.Errorf("min value y = %v, want 50 (bottom)", pts[0].Y)
	}
	if pts[1].Y != 0 {
		t.Errorf("max value y = %v, want 0 (top)", pts[1].Y)
	}
}

func TestPlotPathPadding(t *testing.T) {
	// With padding 5 in a 100-wide box, x spans [5, 95].
	pts := plotPath([]float64{1, 2}, 100, 50, 5)
	if pts[0].X != 5 || pts[1].X != 95 {
		t.Errorf("x range = [%v, %v], want [5, 95]", pts[0].X, pts[1].X)
	}
}

func TestPlotPathFlatSeriesCentered(t *testing.T) {
	pts := plotPath([]float64{5, 5, 5}, 100, 50, 0)
	for i, p := range pts {
		if p.Y != 25 {
			t.Errorf("flat series point %d y = %v, want 25 (mid)", i, p.Y)
		}
	}
}

func TestPlotPathDegenerate(t *testing.T) {
	if plotPath([]float64{1}, 100, 50, 0) != nil {
		t.Error("expected nil for a single point")
	}
	if plotPath(nil, 100, 50, 0) != nil {
		t.Error("expected nil for no points")
	}
	if plotPath([]float64{1, 2}, 4, 50, 4) != nil {
		t.Error("expected nil when padding leaves no width")
	}
}
