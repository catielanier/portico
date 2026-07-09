package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type stepDoneMsg struct {
	err error
}

type stepModel struct {
	label  string
	spin   spinner.Model
	run    func(context.Context) error
	ctx    context.Context
	cancel context.CancelFunc
	err    error
	done   bool
}

func RunStep(label string, run func() error) error {
	return RunStepContext(label, func(context.Context) error {
		return run()
	})
}

func RunStepContext(label string, run func(context.Context) error) error {
	ctx, cancel := context.WithCancel(context.Background())

	model := stepModel{
		label:  label,
		spin:   spinner.New(),
		run:    run,
		ctx:    ctx,
		cancel: cancel,
	}

	program := tea.NewProgram(model)

	finalModel, err := program.Run()
	cancel()

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
		m.spin.Tick,
		func() tea.Msg {
			return stepDoneMsg{err: m.run(m.ctx)}
		},
	)
}

func (m stepModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			if m.cancel != nil {
				m.cancel()
			}

			m.err = context.Canceled
			m.done = true

			return m, tea.Quit
		}

	case stepDoneMsg:
		m.err = msg.err
		m.done = true
		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
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

	return fmt.Sprintf("%s %s\n", m.spin.View(), m.label)
}
