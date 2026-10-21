package main

import (
	"bytes"
	"testing"
)

func TestApplyReport(t *testing.T) {
	state := make([]byte, stateSize)
	// A pressed, RT full, left stick pushed right (0x7fff), rest centered.
	report := []byte{0x00, 0x14, 0x00, 0x10, 0x00, 0xff, 0xff, 0x7f, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	if !applyReport(state, report) {
		t.Fatal("valid report rejected")
	}
	if got, want := state[offGamepad:offGamepad+gamepadSize], report[2:14]; !bytes.Equal(got, want) {
		t.Errorf("gamepad = % x, want % x", got, want)
	}
	if got := state[offPacket]; got != 1 {
		t.Errorf("packet = %d, want 1", got)
	}

	if applyReport(state, []byte{0x01, 0x03, 0x02}) {
		t.Error("LED status report accepted as input")
	}
	if got := state[offPacket]; got != 1 {
		t.Errorf("packet changed on rejected report: %d", got)
	}
}
