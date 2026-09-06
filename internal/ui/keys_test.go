package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestIsCtrl(t *testing.T) {
	msg := tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 'g'}
	if !isCtrl(msg, 'g') {
		t.Fatalf("ctrl+g by code: %q", msg.String())
	}
	if isCtrl(msg, 'q') {
		t.Fatal("ctrl+g matched q")
	}
	q := tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 'q'}
	if !isCtrl(q, 'q') {
		t.Fatal("ctrl+q")
	}
	bell := tea.KeyPressMsg{Code: 7} // C0 BEL is often ctrl+g
	if !isCtrl(bell, 'g') {
		t.Fatalf("BEL should count as ctrl+g, got %q", bell.String())
	}
}
