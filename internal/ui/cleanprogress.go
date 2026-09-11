package ui

import (
	"context"
	"fmt"

	"github.com/catielanier/portico/internal/i18n"
	"github.com/catielanier/portico/internal/portage"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type CleanupProgressEvent struct {
	Mode           portage.CleanupMode
	CurrentPackage string
	CurrentIndex   int
	Total          int
	Waiting        bool
	Message        string
}

type cleanupProgressDoneMsg struct {
	err error
}

type cleanupProgressEventMsg struct {
	event CleanupProgressEvent
}

type cleanupProgressModel struct {
	label          string
	translator     *i18n.Translator
	bar            progress.Model
	spin           spinner.Model
	events         <-chan CleanupProgressEvent
	done           <-chan error
	ctx            context.Context
	cancel         context.CancelFunc
	mode           portage.CleanupMode
	currentPackage string
	currentIndex   int
	total          int
	waiting        bool
	message        string
	err            error
	doneRendering  bool
	succeeded      bool
}

func RunCleanupProgress(
	label string,
	mode portage.CleanupMode,
	total int,
	run func(context.Context, chan<- CleanupProgressEvent) error,
) error {
	ctx, cancel := context.WithCancel(context.Background())

	events := make(chan CleanupProgressEvent)
	done := make(chan error, 1)

	go func() {
		done <- run(ctx, events)
		close(events)
	}()

	model := cleanupProgressModel{
		label:      label,
		translator: i18n.MustDefault(),
		bar:        progress.New(),
		spin:       spinner.New(),
		events:     events,
		done:       done,
		ctx:        ctx,
		cancel:     cancel,
		mode:       mode,
		total:      total,
	}

	program := tea.NewProgram(model)

	finalModel, err := program.Run()
	cancel()

	if err != nil {
		return err
	}

	result, ok := finalModel.(cleanupProgressModel)
	if !ok {
		return fmt.Errorf("unexpected cleanup progress model type")
	}

	return result.err
}

func (m cleanupProgressModel) Init() tea.Cmd {
	return tea.Batch(
		m.spin.Tick,
		m.waitForCleanupMessage(),
	)
}

func (m cleanupProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q", "c":
			if m.cancel != nil {
				m.cancel()
			}

			m.err = context.Canceled
			m.doneRendering = true
			m.succeeded = false

			return m, tea.Quit

		case "enter":
			if m.waiting {
				if m.cancel != nil {
					m.cancel()
				}

				m.err = context.Canceled
				m.doneRendering = true
				m.succeeded = false

				return m, tea.Quit
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)

		if m.doneRendering {
			return m, nil
		}

		return m, cmd

	case cleanupProgressEventMsg:
		m.mode = msg.event.Mode
		m.currentPackage = msg.event.CurrentPackage
		m.currentIndex = msg.event.CurrentIndex
		m.waiting = msg.event.Waiting
		m.message = msg.event.Message

		if msg.event.Total > 0 {
			m.total = msg.event.Total
		}

		return m, m.waitForCleanupMessage()

	case cleanupProgressDoneMsg:
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

func (m cleanupProgressModel) View() string {
	if m.doneRendering {
		if m.err != nil {
			return fmt.Sprintf("✗ %s\n", m.label)
		}

		return fmt.Sprintf("✓ %s\n", m.label)
	}

	if m.waiting {
		return m.waitingView()
	}

	currentPackage := m.currentPackage
	if currentPackage == "" {
		currentPackage = m.t("cleanup_progress_preparing", nil)
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

	packageLine := m.t("cleanup_progress_package_unavailable", nil)
	if totalPackages > 0 && activePackageIndex > 0 {
		packageLine = m.t("cleanup_progress_package_count", map[string]any{
			"Current": activePackageIndex,
			"Total":   totalPackages,
		})
	} else if totalPackages > 0 {
		packageLine = m.t("cleanup_progress_package_count", map[string]any{
			"Current": 0,
			"Total":   totalPackages,
		})
	}

	actionLine := m.t("cleanup_progress_action", map[string]any{
		"Spinner": m.spin.View(),
		"Package": currentPackage,
	})

	progressLine := m.t("cleanup_progress_percent", map[string]any{
		"Percent": fmt.Sprintf("%.0f", percent*100),
	})

	completedLine := m.t("cleanup_progress_completed", map[string]any{
		"Completed": completedPackages,
		"Total":     totalPackages,
	})

	return fmt.Sprintf(
		"%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n\n%s\n",
		m.label,
		actionLine,
		packageLine,
		progressLine,
		m.bar.ViewAs(percent),
		completedLine,
		m.t("cleanup_progress_cancel_hint", nil),
	)
}

func (m cleanupProgressModel) waitingView() string {
	message := m.message
	if message == "" {
		message = m.t("cleanup_progress_waiting_message", nil)
	}

	return fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n%s\n\n[ %s ]\n",
		m.label,
		m.t("cleanup_progress_waiting_title", nil),
		message,
		m.t("cleanup_progress_waiting_hint", nil),
		m.t("common_cancel", nil),
	)
}

func (m cleanupProgressModel) completedPackages() int {
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

func (m cleanupProgressModel) percentComplete() float64 {
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

func (m cleanupProgressModel) waitForCleanupMessage() tea.Cmd {
	return func() tea.Msg {
		select {
		case event, ok := <-m.events:
			if !ok {
				err := <-m.done
				return cleanupProgressDoneMsg{err: err}
			}

			return cleanupProgressEventMsg{event: event}

		case err := <-m.done:
			return cleanupProgressDoneMsg{err: err}

		case <-m.ctx.Done():
			return cleanupProgressDoneMsg{err: m.ctx.Err()}
		}
	}
}

func (m cleanupProgressModel) t(id string, data map[string]any) string {
	translator := m.translator
	if translator == nil {
		translator = i18n.MustDefault()
	}

	return translator.T(id, data)
}