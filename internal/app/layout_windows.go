//go:build windows

package app

import "math"

const (
	windowMaxW = 980
	windowMaxH = 880
	windowMinW = 760
	windowMinH = 700

	layoutDesignW  = 964
	layoutDesignH  = 840
	layoutMinScale = 0.7

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

	pauseLabelX, pauseBtnX, pauseBtnW int

	statusBarW               int
	statusDotX               int
	statusTextX, statusTextW int
}

func computeLayout(cw, ch int) uiLayout {
	sx := layoutScale(cw, layoutDesignW)
	sy := layoutScale(ch, layoutDesignH)

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
	if scale > 1 {
		return 1
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
