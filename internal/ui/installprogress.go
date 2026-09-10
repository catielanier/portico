// internal/ui/installprogress.go
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
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
	spin           spinner.Model
	events         <-chan InstallProgressEvent
	done           <-chan error
	ctx            context.Context
	cancel         context.CancelFunc
	currentPackage string
	currentIndex   int
	total          int
	err            error
	doneRendering  bool
	succeeded      bool
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
		spin:   spinner.New(),
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
	return tea.Batch(
		m.spin.Tick,
		m.waitForInstallMessage(),
	)
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
			m.succeeded = false

			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)

		if m.doneRendering {
			return m, nil
		}

		return m, cmd

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
		m.succeeded = msg.err == nil

		if m.succeeded && m.total > 0 {
			m.currentIndex = m.total
		}

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

	activePackageIndex := m.currentIndex
	if activePackageIndex < 0 {
		activePackageIndex = 0
	}

	totalPackages := m.total
	if totalPackages < 0 {
		totalPackages = 0
	}

	completedPackages := m.completedPackages()
	percent := m.percentComplete()

	packageLine := "Package progress unavailable"
	if totalPackages > 0 && activePackageIndex > 0 {
		packageLine = fmt.Sprintf("Package %d of %d", activePackageIndex, totalPackages)
	} else if totalPackages > 0 {
		packageLine = fmt.Sprintf("Package 0 of %d", totalPackages)
	}

	return fmt.Sprintf(
		"%s\n\n%s\n\n%s Installing %s\n%s\n\nProgress: %.0f%%\n%s\n\nCompleted: %d / %d\n\nPress Ctrl+C to cancel.\n",
		m.label,
		"Install could take some time due to compilation. Grab a snack!",
		m.spin.View(),
		currentPackage,
		packageLine,
		percent*100,
		m.bar.ViewAs(percent),
		completedPackages,
		totalPackages,
	)
}

func (m installProgressModel) completedPackages() int {
	if m.total <= 0 {
		return 0
	}

	if m.succeeded {
		return m.total
	}

	if m.currentIndex <= 0 {
		return 0
	}

	completed := m.currentIndex - 1

	if completed < 0 {
		return 0
	}

	if completed > m.total {
		return m.total
	}

	return completed
}

func (m installProgressModel) percentComplete() float64 {
	if m.total <= 0 {
		return 0
	}

	completed := m.completedPackages()
	percent := float64(completed) / float64(m.total)

	if percent < 0 {
		return 0
	}

	if percent > 1 {
		return 1
	}

	return percent
}

func (m installProgressModel) waitForInstallMessage() tea.Cmd {
	return func() tea.Msg {
		select {
		case event, ok := <-m.events:
			if !ok {
				err := <-m.done
				return installProgressDoneMsg{err: err}
			}

			return installProgressEventMsg{event: event}

		case err := <-m.done:
			return installProgressDoneMsg{err: err}

		case <-m.ctx.Done():
			return installProgressDoneMsg{err: m.ctx.Err()}
		}
	}
}