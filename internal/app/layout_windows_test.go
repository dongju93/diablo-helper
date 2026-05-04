//go:build windows

package app

import "testing"

func TestComputeLayoutKeepsDesignSizeAtMax(t *testing.T) {
	lo := computeLayout(layoutDesignW, layoutDesignH)

	if lo.leftX != layoutLX || lo.leftW != layoutLW || lo.rx != layoutRX {
		t.Fatalf("left column = x:%d w:%d rx:%d, want x:%d w:%d rx:%d", lo.leftX, lo.leftW, lo.rx, layoutLX, layoutLW, layoutRX)
	}
	if lo.rw != layoutDesignW-layoutRX-layoutMX {
		t.Fatalf("right width = %d, want %d", lo.rw, layoutDesignW-layoutRX-layoutMX)
	}
}

func TestComputeLayoutScalesDownForSmallClient(t *testing.T) {
	max := computeLayout(layoutDesignW, layoutDesignH)
	small := computeLayout(744, 561)

	if small.leftW >= max.leftW {
		t.Fatalf("small left width = %d, want less than %d", small.leftW, max.leftW)
	}
	if small.skillBtnW >= max.skillBtnW {
		t.Fatalf("small skill button width = %d, want less than %d", small.skillBtnW, max.skillBtnW)
	}
	if small.statusBarW >= max.statusBarW {
		t.Fatalf("small status width = %d, want less than %d", small.statusBarW, max.statusBarW)
	}
	if small.skillBtnW <= 0 || small.statusTextW <= 0 || small.pauseBtnW <= 0 {
		t.Fatalf("small layout has non-positive widths: skill=%d status=%d pause=%d", small.skillBtnW, small.statusTextW, small.pauseBtnW)
	}
}

func TestComputeLayoutScalesUpForLargeClient(t *testing.T) {
	base := computeLayout(layoutDesignW, layoutDesignH)
	large := computeLayout(layoutDesignW*2, layoutDesignH*2)

	if large.leftW <= base.leftW {
		t.Fatalf("large left width = %d, want greater than %d", large.leftW, base.leftW)
	}
	if large.skillBtnW <= base.skillBtnW {
		t.Fatalf("large skill button width = %d, want greater than %d", large.skillBtnW, base.skillBtnW)
	}
	if large.uiScale() != layoutMaxScale {
		t.Fatalf("large UI scale = %v, want %v", large.uiScale(), layoutMaxScale)
	}
}

func TestComputeWindowBoundsUsesQHDAsReference(t *testing.T) {
	bounds := computeWindowBounds(2560, 1440, 2560, 1400)

	if bounds.minW != windowMinW || bounds.minH != windowMinH {
		t.Fatalf("min bounds = %dx%d, want %dx%d", bounds.minW, bounds.minH, windowMinW, windowMinH)
	}
	if bounds.maxW != windowMaxW || bounds.maxH != windowMaxH {
		t.Fatalf("max bounds = %dx%d, want %dx%d", bounds.maxW, bounds.maxH, windowMaxW, windowMaxH)
	}
}

func TestComputeWindowBoundsScalesForFHDAnd5K(t *testing.T) {
	fhd := computeWindowBounds(1920, 1080, 1920, 1040)
	if fhd.minW != 722 || fhd.minH != 741 || fhd.maxW != 931 || fhd.maxH != 912 {
		t.Fatalf("FHD bounds = min %dx%d max %dx%d, want min 722x741 max 931x912", fhd.minW, fhd.minH, fhd.maxW, fhd.maxH)
	}

	fiveK := computeWindowBounds(5120, 2880, 5120, 2800)
	if fiveK.minW != 1216 || fiveK.minH != 1248 || fiveK.maxW != 1568 || fiveK.maxH != 1536 {
		t.Fatalf("5K bounds = min %dx%d max %dx%d, want min 1216x1248 max 1568x1536", fiveK.minW, fiveK.minH, fiveK.maxW, fiveK.maxH)
	}
}

func TestComputeWindowBoundsCapsMaxToWorkArea(t *testing.T) {
	bounds := computeWindowBounds(2560, 1440, 900, 700)

	if bounds.maxW != 900 || bounds.maxH != 700 {
		t.Fatalf("max bounds = %dx%d, want work area 900x700", bounds.maxW, bounds.maxH)
	}
	if bounds.minW != 760 || bounds.minH != 700 {
		t.Fatalf("min bounds = %dx%d, want 760x700", bounds.minW, bounds.minH)
	}
}

func TestScaledFontHeightKeepsReadableMinimum(t *testing.T) {
	if got := scaledFontHeight(uiFontBaseHeight, uiFontMinScale); got != -12 {
		t.Fatalf("base font height = %d, want -12", got)
	}
	if got := scaledFontHeight(uiTitleFontBaseHeight, uiFontMinScale); got != -22 {
		t.Fatalf("title font height = %d, want -22", got)
	}
	if got := scaledFontHeight(uiSectionFontBaseHeight, uiFontMinScale); got != -13 {
		t.Fatalf("section font height = %d, want -13", got)
	}
}
