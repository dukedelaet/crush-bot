package ui

import (
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
)

func isCtrl(msg tea.KeyPressMsg, r rune) bool {
	want := "ctrl+" + string(unicode.ToLower(r))
	if strings.ToLower(msg.String()) == want {
		return true
	}
	if strings.ToLower(msg.Keystroke()) == want {
		return true
	}
	k := msg.Key()
	if r >= 'a' && r <= 'z' && k.Code == r-'a'+1 {
		return true
	}
	if k.Mod&tea.ModCtrl == 0 {
		return false
	}
	return unicode.ToLower(k.Code) == unicode.ToLower(r)
}
