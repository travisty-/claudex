package ui

import tea "charm.land/bubbletea/v2"

// Model represents the top-level Bubbletea model.
type Model struct{}

// New constructs an instance of the top-level model.
func New() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case msg.Code == 'q':
			return m, tea.Quit
		case msg.Mod == tea.ModCtrl && msg.Code == 'c':
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	return tea.NewView("Claudex: Press 'q' to quit\n")
}
