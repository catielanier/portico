package ui

import (
	"fmt"
	"strings"

	"github.com/catielanier/portico/internal/useflags"
	tea "github.com/charmbracelet/bubbletea"
)

type UsePickerModel struct {
	Atom       string
	Selections []useflags.FlagSelection
	Cursor     int
	Done       bool
	Cancelled  bool
}

func NewUsePickerModel(atom string, selections []useflags.FlagSelection) UsePickerModel {
	return UsePickerModel{
		Atom:       atom,
		Selections: selections,
	}
}

func (m UsePickerModel) Init() tea.Cmd {
	return nil
}

func (m UsePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.Cancelled = true
			return m, tea.Quit

		case "enter":
			m.Done = true
			return m, tea.Quit

		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}

		case "down", "j":
			if m.Cursor < len(m.Selections)-1 {
				m.Cursor++
			}

		case " ":
			if len(m.Selections) > 0 {
				m.Selections[m.Cursor].Selection = m.Selections[m.Cursor].Selection.Next()
			}
		}
	}

	return m, nil
}

func (m UsePickerModel) View() string {
	var b strings.Builder

	b.WriteString("Portico USE Flag Selection\n\n")
	b.WriteString("Package:\n")
	b.WriteString(fmt.Sprintf("  %s\n\n", m.Atom))

	b.WriteString("Controls:\n")
	b.WriteString("  ↑/↓ or k/j  move\n")
	b.WriteString("  space       unset → enabled → disabled\n")
	b.WriteString("  enter       continue\n")
	b.WriteString("  q/esc       cancel\n\n")

	b.WriteString("Legend:\n")
	b.WriteString("  U = current flag setting for next build\n")
	b.WriteString("  Choice: [ ] unset, [+] force enable, [-] force disable\n\n")

	b.WriteString("  U Choice  Flag\n")

	for i, flag := range m.Selections {
		cursor := " "
		if i == m.Cursor {
			cursor = ">"
		}

		current := "-"
		if flag.CurrentEnabled {
			current = "+"
		}

		choice := "[ ]"
		switch flag.Selection {
		case useflags.SelectionEnabled:
			choice = "[+]"
		case useflags.SelectionDisabled:
			choice = "[-]"
		}

		b.WriteString(fmt.Sprintf("%s %s %-6s %-34s\n", cursor, current, choice, flag.Name))

		if i == m.Cursor && flag.Description != "" {
			b.WriteString(fmt.Sprintf("         %s\n", flag.Description))
		}
	}

	return b.String()
}

func RunUsePicker(atom string, selections []useflags.FlagSelection) ([]useflags.FlagSelection, bool, error) {
	model := NewUsePickerModel(atom, selections)

	program := tea.NewProgram(model)

	finalModel, err := program.Run()
	if err != nil {
		return nil, false, err
	}

	picker, ok := finalModel.(UsePickerModel)
	if !ok {
		return nil, false, fmt.Errorf("unexpected USE picker model type")
	}

	if picker.Cancelled {
		return nil, false, nil
	}

	return picker.Selections, true, nil
}
