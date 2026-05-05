//go:build windows

package app

import (
	"math"
	"testing"

	"github.com/dongju93/diablo-helper/internal/config"
)

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

func TestMonitorResolutionScaleUsesVerticalAnchors(t *testing.T) {
	tests := []struct {
		name   string
		widthA int
		widthB int
		height int
		want   float64
	}{
		{name: "HD minimum", widthA: 1280, widthB: 2560, height: 720, want: 0.75},
		{name: "FHD lower interpolation", widthA: 1920, widthB: 3840, height: 1080, want: 0.875},
		{name: "QHD reference", widthA: 2560, widthB: 5120, height: 1440, want: 1.0},
		{name: "4K upper interpolation", widthA: 3840, widthB: 7680, height: 2160, want: 1.24},
		{name: "5K upper interpolation", widthA: 5120, widthB: 7680, height: 2880, want: 1.48},
		{name: "6K maximum", widthA: 5760, widthB: 7680, height: 3240, want: 1.6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotA := monitorVisualScale(tt.widthA, tt.height)
			gotB := monitorVisualScale(tt.widthB, tt.height)

			assertFloatClose(t, gotA, tt.want)
			assertFloatClose(t, gotB, tt.want)
		})
	}
}

func TestComputeWindowBoundsUses1440pAsReference(t *testing.T) {
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

func TestComputeWindowBoundsScalesForHDAnd6K(t *testing.T) {
	hd := computeWindowBounds(testMonitorMetrics(1280, 720, 1280, 700), windowFrame{})
	if hd.minW != 570 || hd.minH != 585 || hd.maxW != 735 || hd.maxH != 700 {
		t.Fatalf("HD bounds = min %dx%d max %dx%d, want min 570x585 max 735x700", hd.minW, hd.minH, hd.maxW, hd.maxH)
	}
	if hd.minH > hd.maximizedH || hd.maxH > hd.maximizedH {
		t.Fatalf("HD bounds heights min=%d max=%d must fit work height %d", hd.minH, hd.maxH, hd.maximizedH)
	}

	sixK := computeWindowBounds(testMonitorMetrics(5760, 3240, 5760, 3200), windowFrame{})
	if sixK.minW != 1216 || sixK.minH != 1248 || sixK.maxW != 1568 || sixK.maxH != 1536 {
		t.Fatalf("6K bounds = min %dx%d max %dx%d, want min 1216x1248 max 1568x1536", sixK.minW, sixK.minH, sixK.maxW, sixK.maxH)
	}
}

func TestComputeWindowBoundsInterpolatesBetweenAnchors(t *testing.T) {
	fhd := computeWindowBounds(testMonitorMetrics(1920, 1080, 1920, 1040), windowFrame{})
	if fhd.minW != 665 || fhd.minH != 683 || fhd.maxW != 858 || fhd.maxH != 840 {
		t.Fatalf("FHD bounds = min %dx%d max %dx%d, want min 665x683 max 858x840", fhd.minW, fhd.minH, fhd.maxW, fhd.maxH)
	}

	fourK := computeWindowBounds(testMonitorMetrics(3840, 2160, 3840, 2080), windowFrame{})
	if fourK.minW != 942 || fourK.minH != 967 || fourK.maxW != 1215 || fourK.maxH != 1190 {
		t.Fatalf("4K bounds = min %dx%d max %dx%d, want min 942x967 max 1215x1190", fourK.minW, fourK.minH, fourK.maxW, fourK.maxH)
	}

	fiveK := computeWindowBounds(testMonitorMetrics(5120, 2880, 5120, 2800), windowFrame{})
	if fiveK.minW != 1125 || fiveK.minH != 1154 || fiveK.maxW != 1450 || fiveK.maxH != 1421 {
		t.Fatalf("5K bounds = min %dx%d max %dx%d, want min 1125x1154 max 1450x1421", fiveK.minW, fiveK.minH, fiveK.maxW, fiveK.maxH)
	}
}

func TestComputeWindowBoundsIncludesWindowFrame(t *testing.T) {
	frame := windowFrame{width: 16, height: 39}
	bounds := computeWindowBounds(testMonitorMetrics(1920, 1080, 1920, 1040), frame)

	if bounds.minW != 681 || bounds.minH != 722 {
		t.Fatalf("min bounds = %dx%d, want 681x722", bounds.minW, bounds.minH)
	}
	if bounds.maxW != 874 || bounds.maxH != 879 {
		t.Fatalf("max bounds = %dx%d, want 874x879", bounds.maxW, bounds.maxH)
	}
}

func TestComputeWindowBoundsDoesNotApplyDPITwiceForFHDWorkArea(t *testing.T) {
	tests := []struct {
		name  string
		dpi   int
		frame windowFrame
	}{
		{name: "150 percent", dpi: defaultDPI * 3 / 2, frame: windowFrame{width: 24, height: 58}},
		{name: "200 percent", dpi: defaultDPI * 2, frame: windowFrame{width: 32, height: 78}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := testMonitorMetrics(1920, 1080, 1920, 1040)
			metrics.dpi = tt.dpi

			bounds := computeWindowBounds(metrics, tt.frame)
			wantMinW := int32(665 + tt.frame.width)
			wantMinH := int32(683 + tt.frame.height)
			if bounds.minW != wantMinW || bounds.minH != wantMinH {
				t.Fatalf("min bounds = %dx%d, want %dx%d", bounds.minW, bounds.minH, wantMinW, wantMinH)
			}
			if bounds.minW > int32(metrics.workW) || bounds.minH > int32(metrics.workH) {
				t.Fatalf("min bounds = %dx%d must fit work area %dx%d", bounds.minW, bounds.minH, metrics.workW, metrics.workH)
			}
		})
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
		{name: "HD compact minimum client", cw: scaledWindowBound(windowMinW, windowResolutionMinScale), ch: scaledWindowBound(windowMinH, windowResolutionMinScale), dpi: defaultDPI},
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

func TestComputeLayoutKeepsControlRowsReadable(t *testing.T) {
	tests := []struct {
		name string
		cw   int
		ch   int
		dpi  int
	}{
		{name: "HD compact minimum client", cw: scaledWindowBound(windowMinW, windowResolutionMinScale), ch: scaledWindowBound(windowMinH, windowResolutionMinScale), dpi: defaultDPI},
		{name: "QHD reference client", cw: layoutDesignW, ch: layoutDesignH, dpi: defaultDPI},
		{name: "maximum scale client", cw: scaledWindowBound(layoutDesignW, layoutMaxScale), ch: scaledWindowBound(layoutDesignH, layoutMaxScale), dpi: defaultDPI},
		{name: "high DPI client", cw: layoutDesignW * 2, ch: layoutDesignH * 2, dpi: defaultDPI * 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lo := computeLayout(tt.cw, tt.ch, tt.dpi)

			leftPanel := testRect("left key panel", lo.leftX, lo.y(92), lo.leftW, lo.h(126))
			menuPanel := testRect("menu panel", lo.leftX, lo.y(menuPanelY), lo.leftW, lo.h(menuPanelH))
			skillPanel := testRect("skill panel", lo.rx, lo.y(92), lo.rw, lo.h(498))
			clickerPanel := testRect("clicker panel", lo.rx, lo.y(clickerPanelY), lo.rw, lo.h(clickerPanelH))
			pausePanel := testRect("pause panel", lo.rx, lo.y(pausePanelY), lo.rw, lo.h(pausePanelH))

			startLabel := testRect("start label", lo.x(layoutLX+24), lo.y(139), lo.w(95), lo.h(24))
			startButton := testRect("start button", lo.x(layoutLX+130), lo.y(134), lo.w(190), lo.h(34))
			stopLabel := testRect("stop label", lo.x(layoutLX+24), lo.y(181), lo.w(95), lo.h(24))
			stopButton := testRect("stop button", lo.x(layoutLX+130), lo.y(176), lo.w(190), lo.h(34))
			assertContained(t, leftPanel, startLabel, startButton, stopLabel, stopButton)
			assertBefore(t, startLabel, startButton, lo.w(4))
			assertBefore(t, stopLabel, stopButton, lo.w(4))

			menuY := menuFirstY
			for _, menu := range menuControls {
				label := testRect(menu.id+" menu label", lo.x(layoutLX+24), lo.y(menuY+5), lo.w(120), lo.h(24))
				button := testRect(menu.id+" menu button", lo.x(layoutLX+150), lo.y(menuY), lo.w(170), lo.h(34))
				assertContained(t, menuPanel, label, button)
				assertBefore(t, label, button, lo.w(4))
				menuY += 40
			}

			bulkLabel := testRect("bulk label", lo.bulkLabelX, lo.y(bulkIntervalLabelY), lo.w(78), lo.h(24))
			bulkEdit := testRect("bulk edit", lo.bulkEditX, lo.y(bulkIntervalEditY), lo.w(bulkEditW), lo.h(22))
			bulkFrame := testRect("bulk edit frame", lo.bulkEditX-lo.w(8), lo.y(bulkIntervalEditY-6), lo.w(inputFrameWidth(bulkEditW)), lo.h(32))
			bulkMS := testRect("bulk ms", lo.bulkMsX, lo.y(bulkIntervalLabelY), lo.w(bulkMsW), lo.h(24))
			bulkApply := testRect("bulk apply", lo.bulkApplyX, lo.y(bulkApplyY), lo.w(bulkApplyW), lo.h(bulkApplyH))
			assertContained(t, skillPanel, bulkLabel, bulkEdit, bulkFrame, bulkMS, bulkApply)
			assertBefore(t, bulkLabel, bulkEdit, lo.w(6))
			assertBefore(t, bulkFrame, bulkMS, lo.w(4))
			assertBefore(t, bulkMS, bulkApply, lo.w(4))

			y := skillFirstRowY
			for range config.MaxSkills {
				toggle := testRect("skill toggle", lo.skillChkX, lo.y(y+4), lo.w(52), lo.h(26))
				num := testRect("skill number", lo.skillNumX, lo.y(y+7), lo.w(skillNumW), lo.h(22))
				key := testRect("skill key", lo.skillBtnX, lo.y(y), lo.skillBtnW, lo.h(34))
				edit := testRect("skill interval", lo.skillIntervalX, lo.y(y+7), lo.w(skillEditW), lo.h(22))
				frame := testRect("skill interval frame", lo.skillIntervalX-lo.w(8), lo.y(y+1), lo.w(inputFrameWidth(skillEditW)), lo.h(32))
				ms := testRect("skill ms", lo.skillMsX, lo.y(y+6), lo.w(skillMsW), lo.h(22))
				assertContained(t, skillPanel, toggle, num, key, edit, frame, ms)
				assertBefore(t, toggle, num, lo.w(4))
				assertBefore(t, num, key, lo.w(8))
				assertBefore(t, key, edit, lo.w(8))
				assertBefore(t, frame, ms, lo.w(4))
				assertMinWidth(t, key, lo.w(120))
				y += skillRowGap
			}

			clickerStartLabel := testRect("clicker start label", lo.clickerStartLabelX, lo.y(clickerHotkeyY+6), lo.w(44), lo.h(24))
			clickerStartButton := testRect("clicker start button", lo.clickerStartBtnX, lo.y(clickerHotkeyY), lo.w(clickerStartBtnW), lo.h(34))
			clickerStopLabel := testRect("clicker stop label", lo.clickerStopLabelX, lo.y(clickerHotkeyY+6), lo.w(44), lo.h(24))
			clickerStopButton := testRect("clicker stop button", lo.clickerStopBtnX, lo.y(clickerHotkeyY), lo.w(clickerStopBtnW), lo.h(34))
			clickerKeyLabel := testRect("clicker key label", lo.clickerKeyLabelX, lo.y(clickerSettingY+6), lo.w(44), lo.h(24))
			clickerKeyButton := testRect("clicker key button", lo.clickerKeyBtnX, lo.y(clickerSettingY), lo.w(clickerKeyBtnW), lo.h(34))
			clickerIntLabel := testRect("clicker interval label", lo.clickerIntLabelX, lo.y(clickerSettingY+6), lo.w(44), lo.h(24))
			clickerIntEdit := testRect("clicker interval edit", lo.clickerIntEditX, lo.y(clickerSettingY+7), lo.w(clickerIntEditW), lo.h(22))
			clickerIntFrame := testRect("clicker interval frame", lo.clickerIntEditX-lo.w(8), lo.y(clickerSettingY+1), lo.w(inputFrameWidth(clickerIntEditW)), lo.h(32))
			clickerMS := testRect("clicker ms", lo.clickerMsLabelX, lo.y(clickerSettingY+6), lo.w(32), lo.h(24))
			assertContained(t, clickerPanel, clickerStartLabel, clickerStartButton, clickerStopLabel, clickerStopButton, clickerKeyLabel, clickerKeyButton, clickerIntLabel, clickerIntEdit, clickerIntFrame, clickerMS)
			assertBefore(t, clickerStartLabel, clickerStartButton, lo.w(4))
			assertBefore(t, clickerStartButton, clickerStopLabel, lo.w(12))
			assertBefore(t, clickerStopLabel, clickerStopButton, lo.w(4))
			assertBefore(t, clickerKeyLabel, clickerKeyButton, lo.w(4))
			assertBefore(t, clickerKeyButton, clickerIntLabel, lo.w(12))
			assertBefore(t, clickerIntLabel, clickerIntEdit, lo.w(4))
			assertBefore(t, clickerIntFrame, clickerMS, lo.w(4))

			pauseLabel := testRect("pause label", lo.pauseLabelX, lo.y(pauseRowY+6), lo.w(45), lo.h(24))
			pauseButton := testRect("pause button", lo.pauseBtnX, lo.y(pauseRowY), lo.pauseBtnW, lo.h(34))
			assertContained(t, pausePanel, pauseLabel, pauseButton)
			assertBefore(t, pauseLabel, pauseButton, lo.w(40))
			assertMinWidth(t, pauseButton, lo.w(180))
		})
	}
}

func TestButtonTextContentRectKeepsTallButtonsReadable(t *testing.T) {
	tests := []struct {
		name         string
		width        int32
		height       int32
		wantMinWidth int32
	}{
		{name: "bulk apply design size", width: bulkApplyW, height: bulkApplyH, wantMinWidth: 76},
		{name: "bulk apply maximum scale", width: int32(scaled(bulkApplyW, layoutMaxScale)), height: int32(scaled(bulkApplyH, layoutMaxScale)), wantMinWidth: 120},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := buttonTextContentRect(rect{Right: tt.width, Bottom: tt.height})
			if got := rc.Right - rc.Left; got < tt.wantMinWidth {
				t.Fatalf("content width = %d, want at least %d", got, tt.wantMinWidth)
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

type testUIRect struct {
	name                     string
	left, top, right, bottom int
}

func testRect(name string, x, y, width, height int) testUIRect {
	return testUIRect{
		name:   name,
		left:   x,
		top:    y,
		right:  x + width,
		bottom: y + height,
	}
}

func (r testUIRect) width() int {
	return r.right - r.left
}

func assertContained(t *testing.T, outer testUIRect, children ...testUIRect) {
	t.Helper()
	for _, child := range children {
		if child.left < outer.left || child.top < outer.top || child.right > outer.right || child.bottom > outer.bottom {
			t.Fatalf("%s = %+v is outside %s = %+v", child.name, child, outer.name, outer)
		}
	}
}

func assertBefore(t *testing.T, left testUIRect, right testUIRect, minGap int) {
	t.Helper()
	if minGap < 0 {
		minGap = 0
	}
	if left.right+minGap > right.left {
		t.Fatalf("%s right=%d and %s left=%d overlap or gap < %d", left.name, left.right, right.name, right.left, minGap)
	}
}

func assertMinWidth(t *testing.T, rect testUIRect, minWidth int) {
	t.Helper()
	if rect.width() < minWidth {
		t.Fatalf("%s width=%d, want at least %d", rect.name, rect.width(), minWidth)
	}
}

func assertFloatClose(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.000001 {
		t.Fatalf("scale = %v, want %v", got, want)
	}
}
