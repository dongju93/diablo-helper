//go:build windows

package app

import "testing"

func TestComputeLayoutKeepsDesignSizeAtMax(t *testing.T) {
	lo := computeLayout(layoutDesignW, layoutDesignH, defaultDPI)

	if lo.leftX != layoutLX || lo.leftW != layoutLW || lo.rx != layoutRX {
		t.Fatalf("left column = x:%d w:%d rx:%d, want x:%d w:%d rx:%d", lo.leftX, lo.leftW, lo.rx, layoutLX, layoutLW, layoutRX)
	}
	if lo.rw != layoutDesignW-layoutRX-layoutMX {
		t.Fatalf("right width = %d, want %d", lo.rw, layoutDesignW-layoutRX-layoutMX)
	}
}

func TestComputeLayoutScalesDownForSmallClient(t *testing.T) {
	max := computeLayout(layoutDesignW, layoutDesignH, defaultDPI)
	small := computeLayout(744, 561, defaultDPI)

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
	base := computeLayout(layoutDesignW, layoutDesignH, defaultDPI)
	large := computeLayout(layoutDesignW*2, layoutDesignH*2, defaultDPI)

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

func TestComputeLayoutIncludesDPIScale(t *testing.T) {
	lo := computeLayout(layoutDesignW*2, layoutDesignH*2, defaultDPI*2)

	if lo.leftW != layoutLW*2 {
		t.Fatalf("DPI-scaled left width = %d, want %d", lo.leftW, layoutLW*2)
	}
	if lo.uiScale() != 2 {
		t.Fatalf("DPI-scaled UI scale = %v, want 2", lo.uiScale())
	}
}

func TestComputeWindowBoundsUsesQHDAsReference(t *testing.T) {
	bounds := computeWindowBounds(testMonitorMetrics(2560, 1440, 2560, 1400), windowFrame{})

	if bounds.minW != windowMinW || bounds.minH != windowMinH {
		t.Fatalf("min bounds = %dx%d, want %dx%d", bounds.minW, bounds.minH, windowMinW, windowMinH)
	}
	if bounds.maxW != windowMaxW || bounds.maxH != windowMaxH {
		t.Fatalf("max bounds = %dx%d, want %dx%d", bounds.maxW, bounds.maxH, windowMaxW, windowMaxH)
	}
	if bounds.maximizedW != 2560 || bounds.maximizedH != 1400 {
		t.Fatalf("maximized bounds = %dx%d, want 2560x1400", bounds.maximizedW, bounds.maximizedH)
	}
}

func TestComputeWindowBoundsScalesForFHDAnd5K(t *testing.T) {
	fhd := computeWindowBounds(testMonitorMetrics(1920, 1080, 1920, 1040), windowFrame{})
	if fhd.minW != 722 || fhd.minH != 741 || fhd.maxW != 931 || fhd.maxH != 912 {
		t.Fatalf("FHD bounds = min %dx%d max %dx%d, want min 722x741 max 931x912", fhd.minW, fhd.minH, fhd.maxW, fhd.maxH)
	}

	fiveK := computeWindowBounds(testMonitorMetrics(5120, 2880, 5120, 2800), windowFrame{})
	if fiveK.minW != 1216 || fiveK.minH != 1248 || fiveK.maxW != 1568 || fiveK.maxH != 1536 {
		t.Fatalf("5K bounds = min %dx%d max %dx%d, want min 1216x1248 max 1568x1536", fiveK.minW, fiveK.minH, fiveK.maxW, fiveK.maxH)
	}
}

func TestComputeWindowBoundsUsesClientAreaAndDPI(t *testing.T) {
	frame := windowFrame{width: 16, height: 39}
	bounds := computeWindowBounds(testMonitorMetrics(1920, 1080, 1920, 1040), frame)

	if bounds.minW != 738 || bounds.minH != 780 {
		t.Fatalf("min bounds = %dx%d, want 738x780", bounds.minW, bounds.minH)
	}
	if bounds.maxW != 947 || bounds.maxH != 951 {
		t.Fatalf("max bounds = %dx%d, want 947x951", bounds.maxW, bounds.maxH)
	}
}

func TestComputeWindowBoundsDoesNotCollapseMinimumToSmallWorkArea(t *testing.T) {
	bounds := computeWindowBounds(testMonitorMetrics(2560, 1440, 900, 700), windowFrame{})

	if bounds.minW != 760 || bounds.minH != 780 {
		t.Fatalf("min bounds = %dx%d, want 760x780", bounds.minW, bounds.minH)
	}
	if bounds.maxW < bounds.minW || bounds.maxH < bounds.minH {
		t.Fatalf("max bounds = %dx%d must not be smaller than min %dx%d", bounds.maxW, bounds.maxH, bounds.minW, bounds.minH)
	}
}

func TestComputeWindowBoundsSetsMaximizePositionForOffsetWorkArea(t *testing.T) {
	metrics := testMonitorMetrics(1920, 1080, 1840, 1000)
	metrics.monitorX = -1920
	metrics.monitorY = 0
	metrics.workX = -1840
	metrics.workY = 40

	bounds := computeWindowBounds(metrics, windowFrame{})

	if bounds.maxPositionX != 80 || bounds.maxPositionY != 40 {
		t.Fatalf("max position = %d,%d, want 80,40", bounds.maxPositionX, bounds.maxPositionY)
	}
	if bounds.maximizedW != 1840 || bounds.maximizedH != 1000 {
		t.Fatalf("maximized size = %dx%d, want 1840x1000", bounds.maximizedW, bounds.maximizedH)
	}
}

func TestComputeLayoutKeepsCriticalRectsContained(t *testing.T) {
	tests := []struct {
		name string
		cw   int
		ch   int
		dpi  int
	}{
		{name: "small FHD client", cw: 744, ch: 561, dpi: defaultDPI},
		{name: "QHD reference client", cw: layoutDesignW, ch: layoutDesignH, dpi: defaultDPI},
		{name: "high DPI client", cw: layoutDesignW * 2, ch: layoutDesignH * 2, dpi: defaultDPI * 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lo := computeLayout(tt.cw, tt.ch, tt.dpi)
			headerW := lo.w(headerBtnW)
			if lo.loadX < 0 || lo.loadX+headerW > lo.saveX || lo.saveX+headerW > tt.cw {
				t.Fatalf("header buttons out of order or bounds: loadX=%d saveX=%d width=%d cw=%d", lo.loadX, lo.saveX, headerW, tt.cw)
			}
			if lo.skillBtnX+lo.skillBtnW > lo.skillIntervalX || lo.skillIntervalX+lo.w(skillEditW) > lo.skillMsX || lo.skillMsX+lo.w(skillMsW) > lo.rx+lo.rw {
				t.Fatalf("skill row overlaps: btn=%d..%d interval=%d..%d ms=%d..%d panelRight=%d",
					lo.skillBtnX, lo.skillBtnX+lo.skillBtnW,
					lo.skillIntervalX, lo.skillIntervalX+lo.w(skillEditW),
					lo.skillMsX, lo.skillMsX+lo.w(skillMsW),
					lo.rx+lo.rw)
			}
			if lo.pauseBtnX+lo.pauseBtnW > lo.rx+lo.rw {
				t.Fatalf("pause button overflows: %d..%d panelRight=%d", lo.pauseBtnX, lo.pauseBtnX+lo.pauseBtnW, lo.rx+lo.rw)
			}
			if lo.y(statusBarY)+lo.h(40) > tt.ch {
				t.Fatalf("status bar overflows: bottom=%d clientH=%d", lo.y(statusBarY)+lo.h(40), tt.ch)
			}
		})
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

func testMonitorMetrics(monitorW, monitorH int, workW, workH int) monitorMetrics {
	return monitorMetrics{
		monitorW: monitorW,
		monitorH: monitorH,
		workW:    workW,
		workH:    workH,
		dpi:      defaultDPI,
	}
}
