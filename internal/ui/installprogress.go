package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

type InstallProgressEvent struct {
	CurrentPackage string
	CurrentIndex   int
	Total          int
}

type installProgressDoneMsg struct {
	err error
}

type installProgressEventMsg struct {
	event InstallProgressEvent
}

type installProgressModel struct {
	label          string
	bar            progress.Model
	events         <-chan InstallProgressEvent
	done           <-chan error
	ctx            context.Context
	cancel         context.CancelFunc
	currentPackage string
	currentIndex   int
	total          int
	err            error
	doneRendering  bool
}

func RunInstallProgress(
	label string,
	total int,
	run func(context.Context, chan<- InstallProgressEvent) error,
) error {
	ctx, cancel := context.WithCancel(context.Background())

	events := make(chan InstallProgressEvent)
	done := make(chan error, 1)

	go func() {
		done <- run(ctx, events)
		close(events)
	}()

	model := installProgressModel{
		label:  label,
		bar:    progress.New(),
		events: events,
		done:   done,
		ctx:    ctx,
		cancel: cancel,
		total:  total,
	}

	program := tea.NewProgram(model)

	finalModel, err := program.Run()
	cancel()

	if err != nil {
		return err
	}

	result, ok := finalModel.(installProgressModel)
	if !ok {
		return fmt.Errorf("unexpected install progress model type")
	}

	return result.err
}

func (m installProgressModel) Init() tea.Cmd {
	return m.waitForInstallMessage()
}

func (m installProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			if m.cancel != nil {
				m.cancel()
			}

			m.err = context.Canceled
			m.doneRendering = true

			return m, tea.Quit
		}

	case installProgressEventMsg:
		m.currentPackage = msg.event.CurrentPackage
		m.currentIndex = msg.event.CurrentIndex

		if msg.event.Total > 0 {
			m.total = msg.event.Total
		}

		return m, m.waitForInstallMessage()

	case installProgressDoneMsg:
		m.err = msg.err
		m.doneRendering = true

		return m, tea.Quit
	}

	return m, nil
}

func (m installProgressModel) View() string {
	if m.doneRendering {
		if m.err != nil {
			return fmt.Sprintf("✗ %s\n", m.label)
		}

		return fmt.Sprintf("✓ %s\n", m.label)
	}

	currentPackage := m.currentPackage
	if currentPackage == "" {
		currentPackage = "Preparing emerge transaction..."
	}

	percent := 0.0
	if m.total > 0 && m.currentIndex > 0 {
		percent = float64(m.currentIndex) / float64(m.total)
	}

	if percent > 1 {
		percent = 1
	}

	return fmt.Sprintf(
		"%s\n\n%s\n%s\n\n%d / %d\n\nPress Ctrl+C to cancel.\n",
		m.label,
		currentPackage,
		m.bar.ViewAs(percent),
		m.currentIndex,
		m.total,
	)
}

func (m installProgressModel) waitForInstallMessage() tea.Cmd {
	return func() tea.Msg {
		select {
		case event, ok := <-m.events:
			if !ok {
				return installProgressDoneMsg{err: nil}
			}

			return installProgressEventMsg{event: event}

		case err := <-m.done:
			return installProgressDoneMsg{err: err}

		case <-m.ctx.Done():
			return installProgressDoneMsg{err: m.ctx.Err()}
		}
	}
}
