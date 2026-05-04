//go:build windows

package app

import "math"

const (
	defaultDPI = 96

	windowMaxW = 980
	windowMaxH = 960
	windowMinW = 760
	windowMinH = 780

	windowReferenceMonitorW  = 2560
	windowReferenceMonitorH  = 1440
	windowResolutionMinScale = 0.95
	windowResolutionMaxScale = 1.6

	layoutDesignW  = 964
	layoutDesignH  = 920
	layoutMinScale = 0.58
	layoutMaxScale = windowResolutionMaxScale

	layoutLX  = 24
	layoutLW  = 348
	layoutGap = 24
	layoutRX  = layoutLX + layoutLW + layoutGap // 396
	layoutMX  = 24                              // right outer margin

	// Bulk section (right-anchored within right panel)
	bulkApplyMarR = 32
	bulkApplyW    = 92
	bulkMsGapL    = 16
	bulkMsW       = 30
	bulkEditGapL  = 18
	bulkEditW     = 62
	bulkLabelOffL = 202 // fixed offset from rx

	// Skill grid (right-anchored)
	skillMsMarR  = 82
	skillMsW     = 32
	skillMsGapL  = 26
	skillEditW   = 56
	skillBtnOffL = 152
	skillBtnGapR = 32
	skillChkOffL = 42
	skillNumOffL = 105
	skillNumW    = 30

	// Skill header offsets from rx
	skillUseHdrOffL  = 34
	skillNumHdrOffL  = 96
	skillKeyHdrOffL  = 196
	skillIntHdrShift = 10 // skillIntHdrX = skillIntervalX - skillIntHdrShift

	// Clicker (right column, rx-anchored)
	clickerStartLabelOffL = 24
	clickerStartBtnOffL   = 72
	clickerStopLabelOffL  = 186
	clickerStopBtnOffL    = 234
	clickerStopBtnW       = 100
	clickerKeyLabelOffL   = 24
	clickerKeyBtnOffL     = 72
	clickerIntLabelOffL   = 186
	clickerIntEditOffL    = 234
	clickerIntEditW       = 62
	clickerMsLabelOffL    = 306

	// Pause
	pauseLabelOffL = 30
	pauseBtnOffL   = 152
	pauseBtnMarR   = 130

	// Header buttons (right-anchored to client)
	headerBtnW    = 120
	headerBtnGap  = 12
	headerBtnMarR = 32

	// Status bar
	statusDotOffL  = 68
	statusTextOffL = 88
	statusTextMarR = 132
)

type uiLayout struct {
	cw, ch int
	sx, sy float64

	leftX, leftW int
	gap          int
	rx, rw       int

	saveX, loadX int

	bulkLabelX, bulkEditX, bulkMsX, bulkApplyX int

	skillUseHdrX, skillNumHdrX, skillKeyHdrX, skillIntHdrX int
	skillChkX, skillNumX                                   int
	skillBtnX, skillBtnW                                   int
	skillIntervalX, skillMsX                               int

	clickerStartLabelX, clickerStartBtnX               int
	clickerStopLabelX, clickerStopBtnX                 int
	clickerKeyLabelX, clickerKeyBtnX                   int
	clickerIntLabelX, clickerIntEditX, clickerMsLabelX int

	pauseLabelX, pauseBtnX, pauseBtnW int

	statusBarW               int
	statusDotX               int
	statusTextX, statusTextW int
}

type windowBounds struct {
	minW         int32
	minH         int32
	maxW         int32
	maxH         int32
	maxPositionX int32
	maxPositionY int32
	maximizedW   int32
	maximizedH   int32
}

type windowFrame struct {
	width  int
	height int
}

func computeWindowBounds(metrics monitorMetrics, frame windowFrame) windowBounds {
	scale := monitorVisualScale(metrics.monitorW, metrics.monitorH)
	minClientW := scaledWindowBound(windowMinW, scale)
	minClientH := scaledWindowBound(windowMinH, scale)
	maxClientW := scaledWindowBound(windowMaxW, scale)
	maxClientH := scaledWindowBound(windowMaxH, scale)

	frameW := maxInt(0, frame.width)
	frameH := maxInt(0, frame.height)
	if metrics.workW > frameW {
		maxClientW = minInt(maxClientW, metrics.workW-frameW)
	}
	if metrics.workH > frameH {
		maxClientH = minInt(maxClientH, metrics.workH-frameH)
	}
	maxClientW = maxInt(minClientW, maxClientW)
	maxClientH = maxInt(minClientH, maxClientH)

	minW := minClientW + frameW
	minH := minClientH + frameH
	maxW := maxClientW + frameW
	maxH := maxClientH + frameH

	maximizedW := metrics.workW
	maximizedH := metrics.workH
	if maximizedW <= 0 {
		maximizedW = metrics.monitorW
	}
	if maximizedH <= 0 {
		maximizedH = metrics.monitorH
	}
	if maximizedW <= 0 {
		maximizedW = maxW
	}
	if maximizedH <= 0 {
		maximizedH = maxH
	}

	return windowBounds{
		minW:         int32(maxInt(1, minW)),
		minH:         int32(maxInt(1, minH)),
		maxW:         int32(maxInt(1, maxW)),
		maxH:         int32(maxInt(1, maxH)),
		maxPositionX: int32(metrics.workX - metrics.monitorX),
		maxPositionY: int32(metrics.workY - metrics.monitorY),
		maximizedW:   int32(maxInt(1, maximizedW)),
		maximizedH:   int32(maxInt(1, maximizedH)),
	}
}

func monitorVisualScale(monitorW, monitorH int) float64 {
	return monitorResolutionScale(monitorW, monitorH)
}

func monitorResolutionScale(monitorW, monitorH int) float64 {
	if monitorW <= 0 || monitorH <= 0 {
		return 1
	}
	scale := math.Min(
		float64(monitorW)/float64(windowReferenceMonitorW),
		float64(monitorH)/float64(windowReferenceMonitorH),
	)
	return clampFloat(scale, windowResolutionMinScale, windowResolutionMaxScale)
}

func normalizedDPI(dpi int) int {
	if dpi <= 0 {
		return defaultDPI
	}
	return dpi
}

func dpiScale(dpi int) float64 {
	return float64(normalizedDPI(dpi)) / float64(defaultDPI)
}

func logicalPixels(physical int, dpi int) int {
	if physical <= 0 {
		return physical
	}
	return maxInt(1, int(math.Round(float64(physical)/dpiScale(dpi))))
}

func scaledWindowBound(value int, scale float64) int {
	return maxInt(1, int(math.Round(float64(value)*scale)))
}

func computeLayout(cw, ch int, dpi int) uiLayout {
	scale := dpiScale(dpi)
	logicalW := logicalPixels(cw, dpi)
	logicalH := logicalPixels(ch, dpi)
	sx := layoutScale(logicalW, layoutDesignW) * scale
	sy := layoutScale(logicalH, layoutDesignH) * scale

	leftX := scaled(layoutLX, sx)
	leftW := scaled(layoutLW, sx)
	gap := scaled(layoutGap, sx)
	rightMargin := scaled(layoutMX, sx)
	rx := leftX + leftW + gap
	rw := maxInt(1, cw-rx-rightMargin)

	headerButtonW := scaled(headerBtnW, sx)

	saveX := cw - scaled(headerBtnMarR, sx) - headerButtonW
	loadX := saveX - scaled(headerBtnGap, sx) - headerButtonW

	bulkApplyX := rx + rw - scaled(bulkApplyMarR, sx) - scaled(bulkApplyW, sx)
	bulkMsX := bulkApplyX - scaled(bulkMsGapL, sx) - scaled(bulkMsW, sx)
	bulkEditX := bulkMsX - scaled(bulkEditGapL, sx) - scaled(bulkEditW, sx)
	bulkLabelX := rx + scaled(bulkLabelOffL, sx)

	skillUseHdrX := rx + scaled(skillUseHdrOffL, sx)
	skillNumHdrX := rx + scaled(skillNumHdrOffL, sx)
	skillKeyHdrX := rx + scaled(skillKeyHdrOffL, sx)

	skillMsX := rx + rw - scaled(skillMsMarR, sx) - scaled(skillMsW, sx)
	skillIntervalX := skillMsX - scaled(skillMsGapL, sx) - scaled(skillEditW, sx)
	skillBtnX := rx + scaled(skillBtnOffL, sx)
	skillBtnW := maxInt(1, skillIntervalX-scaled(skillBtnGapR, sx)-skillBtnX)
	skillIntHdrX := skillIntervalX - scaled(skillIntHdrShift, sx)
	skillChkX := rx + scaled(skillChkOffL, sx)
	skillNumX := rx + scaled(skillNumOffL, sx)

	clickerStartLabelX := rx + scaled(clickerStartLabelOffL, sx)
	clickerStartBtnX := rx + scaled(clickerStartBtnOffL, sx)
	clickerStopLabelX := rx + scaled(clickerStopLabelOffL, sx)
	clickerStopBtnX := rx + scaled(clickerStopBtnOffL, sx)
	clickerKeyLabelX := rx + scaled(clickerKeyLabelOffL, sx)
	clickerKeyBtnX := rx + scaled(clickerKeyBtnOffL, sx)
	clickerIntLabelX := rx + scaled(clickerIntLabelOffL, sx)
	clickerIntEditX := rx + scaled(clickerIntEditOffL, sx)
	clickerMsLabelX := rx + scaled(clickerMsLabelOffL, sx)

	pauseLabelX := rx + scaled(pauseLabelOffL, sx)
	pauseBtnX := rx + scaled(pauseBtnOffL, sx)
	pauseBtnW := maxInt(1, (rx+rw-scaled(pauseBtnMarR, sx))-pauseBtnX)

	statusBarW := leftW + gap + rw
	statusDotX := leftX + scaled(statusDotOffL, sx)
	statusTextX := leftX + scaled(statusTextOffL, sx)
	statusTextW := maxInt(1, statusBarW-scaled(statusTextMarR, sx))

	return uiLayout{
		cw: cw, ch: ch,
		sx: sx, sy: sy,
		leftX: leftX, leftW: leftW,
		gap: gap,
		rx:  rx, rw: rw,
		saveX: saveX, loadX: loadX,
		bulkLabelX: bulkLabelX, bulkEditX: bulkEditX, bulkMsX: bulkMsX, bulkApplyX: bulkApplyX,
		skillUseHdrX: skillUseHdrX, skillNumHdrX: skillNumHdrX, skillKeyHdrX: skillKeyHdrX, skillIntHdrX: skillIntHdrX,
		skillChkX: skillChkX, skillNumX: skillNumX,
		skillBtnX: skillBtnX, skillBtnW: skillBtnW,
		skillIntervalX: skillIntervalX, skillMsX: skillMsX,
		clickerStartLabelX: clickerStartLabelX, clickerStartBtnX: clickerStartBtnX,
		clickerStopLabelX: clickerStopLabelX, clickerStopBtnX: clickerStopBtnX,
		clickerKeyLabelX: clickerKeyLabelX, clickerKeyBtnX: clickerKeyBtnX,
		clickerIntLabelX: clickerIntLabelX, clickerIntEditX: clickerIntEditX, clickerMsLabelX: clickerMsLabelX,
		pauseLabelX: pauseLabelX, pauseBtnX: pauseBtnX, pauseBtnW: pauseBtnW,
		statusBarW: statusBarW, statusDotX: statusDotX,
		statusTextX: statusTextX, statusTextW: statusTextW,
	}
}

func layoutScale(size, design int) float64 {
	if size <= 0 || design <= 0 {
		return 1
	}
	scale := float64(size) / float64(design)
	if scale > layoutMaxScale {
		return layoutMaxScale
	}
	if scale < layoutMinScale {
		return layoutMinScale
	}
	return scale
}

func scaled(value int, scale float64) int {
	if value == 0 {
		return 0
	}
	result := int(math.Round(float64(value) * scale))
	if value > 0 && result < 1 {
		return 1
	}
	if value < 0 && result > -1 {
		return -1
	}
	return result
}

func (lo uiLayout) x(value int) int {
	return scaled(value, lo.sx)
}

func (lo uiLayout) y(value int) int {
	return scaled(value, lo.sy)
}

func (lo uiLayout) w(value int) int {
	return scaled(value, lo.sx)
}

func (lo uiLayout) h(value int) int {
	return scaled(value, lo.sy)
}

func (lo uiLayout) s(value int) int {
	if lo.sx < lo.sy {
		return scaled(value, lo.sx)
	}
	return scaled(value, lo.sy)
}

func (lo uiLayout) uiScale() float64 {
	if lo.sx < lo.sy {
		return lo.sx
	}
	return lo.sy
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampFloat(value float64, minValue float64, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
