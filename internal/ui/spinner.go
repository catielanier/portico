package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type stepDoneMsg struct {
	err error
}

type stepModel struct {
	label   string
	spinner spinner.Model
	run     func() error
	err     error
	done    bool
}

func RunStep(label string, run func() error) error {
	model := stepModel{
		label:   label,
		spinner: spinner.New(),
		run:     run,
	}

	program := tea.NewProgram(model)

	finalModel, err := program.Run()
	if err != nil {
		return err
	}

	result, ok := finalModel.(stepModel)
	if !ok {
		return fmt.Errorf("unexpected spinner model type")
	}

	return result.err
}

func (m stepModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			return stepDoneMsg{err: m.run()}
		},
	)
}

func (m stepModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case stepDoneMsg:
		m.err = msg.err
		m.done = true
		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m stepModel) View() string {
	if m.done {
		if m.err != nil {
			return fmt.Sprintf("✗ %s\n", m.label)
		}

		return fmt.Sprintf("✓ %s\n", m.label)
	}

	return fmt.Sprintf("%s %s\n", m.spinner.View(), m.label)
}
