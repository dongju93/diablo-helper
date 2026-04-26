//go:build windows

package app

import (
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

	for _, id := range []int{idStartKey, idStopKey, idPauseKey, idSkillKeyBase, idSkillKeyBase + config.MaxSkills - 1, idMenuInventory, idMenuWhisper} {
		if !a.isBindingButton(id) {
			t.Fatalf("id %d should be a binding button", id)
		}
	}
	for _, id := range []int{idSave, idApplyBulk, idSkillKeyBase + config.MaxSkills} {
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
			got := bulkIntervalForSkill(tt.baseInterval, tt.skillGap, tt.index)
			if got != tt.want {
				t.Fatalf("bulkIntervalForSkill() = %d, want %d", got, tt.want)
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
		{name: "skill", id: idSkillKeyBase + 2, want: captureTarget{kind: captureSkill, index: 2}},
		{name: "menu", id: idMenuWorldMap, want: captureTarget{kind: captureMenu, menuID: "world_map"}},
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
