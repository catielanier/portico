package cli

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type confirmChoice int

const (
	confirmChoiceConfirm confirmChoice = iota
	confirmChoiceCancel
)

type confirmModel struct {
	prompt    string
	focused   confirmChoice
	done      bool
	confirmed bool
}

func confirmDefaultNo(prompt string) (bool, error) {
	// Legacy function name. Current Portico UX defaults focus to Confirm.
	return confirmWithButtons(prompt, confirmChoiceConfirm)
}

func confirmDefaultYes(prompt string) (bool, error) {
	return confirmWithButtons(prompt, confirmChoiceConfirm)
}

func confirmWithButtons(prompt string, defaultChoice confirmChoice) (bool, error) {
	model := confirmModel{
		prompt:  prompt,
		focused: defaultChoice,
	}

	program := tea.NewProgram(model)

	finalModel, err := program.Run()
	if err != nil {
		return false, err
	}

	result, ok := finalModel.(confirmModel)
	if !ok {
		return false, fmt.Errorf("unexpected confirm model type")
	}

	return result.confirmed, nil
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q", "n":
			m.done = true
			m.confirmed = false
			return m, tea.Quit

		case "y":
			m.done = true
			m.confirmed = true
			return m, tea.Quit

		case "enter":
			m.done = true
			m.confirmed = m.focused == confirmChoiceConfirm
			return m, tea.Quit

		case "left", "h", "shift+tab":
			m.focused = confirmChoiceConfirm

		case "right", "l", "tab":
			m.focused = confirmChoiceCancel
		}
	}

	return m, nil
}

func (m confirmModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(m.prompt)
	b.WriteString("\n\n")

	b.WriteString("  ")
	b.WriteString(renderConfirmButton("Confirm", m.focused == confirmChoiceConfirm))
	b.WriteString("    ")
	b.WriteString(renderConfirmButton("Cancel", m.focused == confirmChoiceCancel))
	b.WriteString("\n\n")

	b.WriteString("  ←/→, h/l, tab/shift+tab to move • enter to select\n")

	return b.String()
}

func renderConfirmButton(label string, focused bool) string {
	if focused {
		return "[ " + label + " ]"
	}

	return "  " + label + "  "
}
