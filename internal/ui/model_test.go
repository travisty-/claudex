package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestModel_Update(t *testing.T) {
	cases := []struct {
		name string
		msg  tea.Msg
		want bool
	}{
		{"quits on q", tea.KeyPressMsg{Code: 'q'}, true},
		{"quits on ctrl+c", tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 'c'}, true},
		{"ignores other keys", tea.KeyPressMsg{Code: 'x'}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := New()

			_, cmd := m.Update(c.msg)
			if got := isQuitCmd(cmd); got != c.want {
				t.Errorf("Update(%v): got=%v, want=%v", c.msg, got, c.want)
			}
		})
	}
}

func isQuitCmd(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}
