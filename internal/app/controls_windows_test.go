//go:build windows

package app

import (
	"strings"
	"testing"

	"github.com/dongju93/diablo-helper/internal/config"
)

func TestApplicationControlClassification(t *testing.T) {
	a := newApplication()

	if !a.isPrimaryButton(idSave) {
		t.Fatal("idSave should be primary")
	}
	if !a.isPrimaryButton(idApplyBulk) {
		t.Fatal("idApplyBulk should be primary")
	}
	if a.isPrimaryButton(idLoad) {
		t.Fatal("idLoad should not be primary")
	}

	for _, id := range []int{idStartKey, idStopKey, idPauseKey, idClickerStartKey, idClickerStopKey, idClickerKey, idSkillKeyBase, idSkillKeyBase + config.MaxSkills - 1, idMenuCharacter, idMenuShop} {
		if !a.isBindingButton(id) {
			t.Fatalf("id %d should be a binding button", id)
		}
	}
	for _, id := range []int{idSave, idApplyBulk, idClickerInterval, idSkillKeyBase + config.MaxSkills} {
		if a.isBindingButton(id) {
			t.Fatalf("id %d should not be a binding button", id)
		}
	}
}

func TestBulkIntervalForSkillAppliesGapByRow(t *testing.T) {
	tests := []struct {
		name         string
		baseInterval int
		skillGap     int
		index        int
		want         int
	}{
		{name: "first skill uses base interval", baseInterval: 1000, skillGap: 50, index: 0, want: 1000},
		{name: "second skill adds one gap", baseInterval: 1000, skillGap: 50, index: 1, want: 1050},
		{name: "eighth skill adds seven gaps", baseInterval: 1000, skillGap: 50, index: 7, want: 1350},
		{name: "zero gap keeps same interval", baseInterval: 1000, skillGap: 0, index: 7, want: 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bulkIntervalForSkill(tt.baseInterval, tt.skillGap, tt.index)
			if err != nil {
				t.Fatalf("bulkIntervalForSkill() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("bulkIntervalForSkill() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestBulkIntervalForSkillRejectsInvalidResults(t *testing.T) {
	tests := []struct {
		name         string
		baseInterval int
		skillGap     int
		index        int
		wantError    string
	}{
		{name: "base below minimum", baseInterval: config.MinimumIntervalMS - 1, skillGap: 0, index: 0, wantError: "최소"},
		{name: "base above maximum", baseInterval: config.MaximumIntervalMS + 1, skillGap: 0, index: 0, wantError: "최대"},
		{name: "gap below zero", baseInterval: config.MinimumIntervalMS, skillGap: -1, index: 0, wantError: "0ms 이상"},
		{name: "gap above maximum", baseInterval: config.MinimumIntervalMS, skillGap: config.MaximumSkillGapMS + 1, index: 0, wantError: "최대"},
		{name: "negative index", baseInterval: config.MinimumIntervalMS, skillGap: 0, index: -1, wantError: "기술 번호"},
		{name: "result above maximum", baseInterval: config.MaximumIntervalMS, skillGap: 1, index: 1, wantError: "최대"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := bulkIntervalForSkill(tt.baseInterval, tt.skillGap, tt.index)
			if err == nil {
				t.Fatal("bulkIntervalForSkill() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("bulkIntervalForSkill() error = %v, want %q", err, tt.wantError)
			}
		})
	}
}

func TestHandleCommandStartsKeyCapture(t *testing.T) {
	tests := []struct {
		name string
		id   int
		want captureTarget
	}{
		{name: "start", id: idStartKey, want: captureTarget{kind: captureStart}},
		{name: "stop", id: idStopKey, want: captureTarget{kind: captureStop}},
		{name: "pause", id: idPauseKey, want: captureTarget{kind: capturePause}},
		{name: "clicker start", id: idClickerStartKey, want: captureTarget{kind: captureClickerStart}},
		{name: "clicker stop", id: idClickerStopKey, want: captureTarget{kind: captureClickerStop}},
		{name: "clicker key", id: idClickerKey, want: captureTarget{kind: captureClickerKey}},
		{name: "skill", id: idSkillKeyBase + 2, want: captureTarget{kind: captureSkill, index: 2}},
		{name: "menu", id: idMenuTownPortal, want: captureTarget{kind: captureMenu, menuID: "town_portal"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newApplication()
			if !a.handleCommand(makeLong(tt.id, bnClicked)) {
				t.Fatal("handleCommand() = false, want true")
			}
			if a.capture != tt.want {
				t.Fatalf("capture = %+v, want %+v", a.capture, tt.want)
			}
		})
	}
}

func TestHandleCommandIgnoresNonClickOrUnknownCommand(t *testing.T) {
	a := newApplication()
	if a.handleCommand(makeLong(idStartKey, 1)) {
		t.Fatal("handleCommand() = true for non-click notification")
	}
	if a.handleCommand(makeLong(9999, bnClicked)) {
		t.Fatal("handleCommand() = true for unknown command")
	}
	if a.capture.valid() {
		t.Fatalf("capture = %+v, want unchanged", a.capture)
	}
}
