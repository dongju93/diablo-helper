//go:build windows

package app

import "testing"

func TestSavedWindowPlacementRoundTrip(t *testing.T) {
	want := savedWindowPlacement{
		x:         -120,
		y:         80,
		width:     900,
		height:    700,
		maximized: true,
	}

	got, ok := parseSavedWindowPlacement(formatSavedWindowPlacement(want))
	if !ok {
		t.Fatal("parseSavedWindowPlacement() ok = false, want true")
	}
	if got != want {
		t.Fatalf("placement = %+v, want %+v", got, want)
	}
}

func TestParseSavedWindowPlacementRejectsInvalidValues(t *testing.T) {
	tests := []string{
		"",
		"1,2,3",
		"1,2,0,4,0",
		"1,2,3,-4,0",
		"1,2,3,4,2",
		"1000001,2,3,4,0",
		"-2147483648,2,3,4,0",
		"1,2,100001,4,0",
	}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			if got, ok := parseSavedWindowPlacement(tt); ok {
				t.Fatalf("parseSavedWindowPlacement(%q) = %+v, true; want false", tt, got)
			}
		})
	}
}

func TestNormalizeWindowPlacementClampsToBounds(t *testing.T) {
	bounds := windowBounds{
		minW: 400,
		minH: 300,
		maxW: 900,
		maxH: 700,
	}
	placement := savedWindowPlacement{
		x:      50,
		y:      900,
		width:  1200,
		height: 100,
	}

	got, ok := normalizeWindowPlacement(placement, bounds)
	if !ok {
		t.Fatal("normalizeWindowPlacement() ok = false, want true")
	}
	if got.x != 50 || got.y != 900 || got.width != 900 || got.height != 300 {
		t.Fatalf("normalized placement = %+v, want x=50 y=900 width=900 height=300", got)
	}
}

func TestNormalizeWindowPlacementRejectsInvalidSize(t *testing.T) {
	if got, ok := normalizeWindowPlacement(savedWindowPlacement{width: 0, height: 100}, windowBounds{}); ok {
		t.Fatalf("normalizeWindowPlacement() = %+v, true; want false", got)
	}
}
