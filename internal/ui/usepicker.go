package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/catielanier/portico/internal/useflags"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	defaultTerminalHeight = 24
	minFlagsPerPage      = 1
	usePickerReservedRows = 14
)

type usePickerAction int

const (
	usePickerActionPrev usePickerAction = iota
	usePickerActionNext
	usePickerActionConfirm
	usePickerActionCancel
)

type UsePickerModel struct {
	Atom        string
	Selections []useflags.FlagSelection

	Cursor int
	Page   int

	Width  int
	Height int

	FocusedAction usePickerAction

	Done      bool
	Cancelled bool
}

func NewUsePickerModel(atom string, selections []useflags.FlagSelection) UsePickerModel {
	model := UsePickerModel{
		Atom:          atom,
		Selections:    selections,
		Height:        defaultTerminalHeight,
		FocusedAction: usePickerActionConfirm,
	}

	model.resetFocusedActionForPage()

	return model
}

func (m UsePickerModel) Init() tea.Cmd {
	return nil
}

func (m UsePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.clampPageAndCursor()
		m.resetFocusedActionForPage()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.Cancelled = true
			return m, tea.Quit

		case "up", "k":
			m.moveCursorUp()

		case "down", "j":
			m.moveCursorDown()

		case "left", "h", "shift+tab":
			m.focusPreviousAction()

		case "right", "l", "tab":
			m.focusNextAction()

		case " ":
			m.toggleCurrentFlag()

		case "enter":
			return m.activateFocusedAction()
		}
	}

	return m, nil
}

func (m UsePickerModel) View() string {
	if m.Done || m.Cancelled {
		return ""
	}

	if len(m.Selections) == 0 {
		return ""
	}

	var b strings.Builder

	pageCount := m.pageCount()
	flagsPerPage := m.flagsPerPage()
	start, end := m.visibleRange()

	b.WriteString(fmt.Sprintf("USE flags — %s\n\n", m.Atom))

	if pageCount > 1 {
		b.WriteString(fmt.Sprintf("Page %d of %d\n\n", m.Page+1, pageCount))
	}

	if m.hasInstalledColumn() {
		b.WriteString("  U I  Choice  Flag\n")
	} else {
		b.WriteString("  U  Choice  Flag\n")
	}

	for i := start; i < end; i++ {
		flag := m.Selections[i]

		cursor := " "
		if i == m.Cursor {
			cursor = ">"
		}

		useState := "-"
		if flag.CurrentEnabled {
			useState = "+"
		}

		choice := "[ ]"
		switch flag.Selection {
		case useflags.SelectionEnabled:
			choice = "[+]"
		case useflags.SelectionDisabled:
			choice = "[-]"
		}

		if m.hasInstalledColumn() {
			installedState := "?"
			if flag.Installed != nil {
				if *flag.Installed {
					installedState = "+"
				} else {
					installedState = "-"
				}
			}

			b.WriteString(fmt.Sprintf("%s %s %s  %-6s  %s\n", cursor, useState, installedState, choice, flag.Name))
		} else {
			b.WriteString(fmt.Sprintf("%s %s  %-6s  %s\n", cursor, useState, choice, flag.Name))
		}
	}

	renderedFlags := end - start
	for i := renderedFlags; i < flagsPerPage; i++ {
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString("↑/↓ or k/j Navigate   Space Toggle\n")
	b.WriteString("←/→ or h/l Move buttons   Enter Select\n\n")
	b.WriteString(m.renderButtons())
	b.WriteString("\n")

	return b.String()
}

func (m UsePickerModel) moveCursorUp() {
	if len(m.Selections) == 0 {
		return
	}

	start, _ := m.visibleRange()

	if m.Cursor > start {
		m.Cursor--
		return
	}

	if m.Page > 0 {
		m.Page--
		_, end := m.visibleRange()
		m.Cursor = end - 1
		m.resetFocusedActionForPage()
	}
}

func (m UsePickerModel) moveCursorDown() {
	if len(m.Selections) == 0 {
		return
	}

	_, end := m.visibleRange()

	if m.Cursor < end-1 {
		m.Cursor++
		return
	}

	if m.Page < m.pageCount()-1 {
		m.Page++
		start, _ := m.visibleRange()
		m.Cursor = start
		m.resetFocusedActionForPage()
	}
}

func (m *UsePickerModel) toggleCurrentFlag() {
	if len(m.Selections) == 0 {
		return
	}

	if m.Cursor < 0 || m.Cursor >= len(m.Selections) {
		return
	}

	m.Selections[m.Cursor].Selection = m.Selections[m.Cursor].Selection.Next()
}

func (m UsePickerModel) activateFocusedAction() (tea.Model, tea.Cmd) {
	switch m.FocusedAction {
	case usePickerActionPrev:
		if m.Page > 0 {
			m.Page--
			start, _ := m.visibleRange()
			m.Cursor = start
			m.resetFocusedActionForPage()
		}

		return m, nil

	case usePickerActionNext:
		if m.Page < m.pageCount()-1 {
			m.Page++
			start, _ := m.visibleRange()
			m.Cursor = start
			m.resetFocusedActionForPage()
			return m, nil
		}

		m.Done = true
		return m, tea.Quit

	case usePickerActionConfirm:
		m.Done = true
		return m, tea.Quit

	case usePickerActionCancel:
		m.Cancelled = true
		return m, tea.Quit

	default:
		return m, nil
	}
}

func (m *UsePickerModel) focusPreviousAction() {
	actions := m.availableActions()
	if len(actions) == 0 {
		return
	}

	currentIndex := 0
	for i, action := range actions {
		if action == m.FocusedAction {
			currentIndex = i
			break
		}
	}

	if currentIndex == 0 {
		m.FocusedAction = actions[len(actions)-1]
		return
	}

	m.FocusedAction = actions[currentIndex-1]
}

func (m *UsePickerModel) focusNextAction() {
	actions := m.availableActions()
	if len(actions) == 0 {
		return
	}

	currentIndex := 0
	for i, action := range actions {
		if action == m.FocusedAction {
			currentIndex = i
			break
		}
	}

	if currentIndex == len(actions)-1 {
		m.FocusedAction = actions[0]
		return
	}

	m.FocusedAction = actions[currentIndex+1]
}

func (m *UsePickerModel) resetFocusedActionForPage() {
	if len(m.Selections) == 0 {
		m.FocusedAction = usePickerActionConfirm
		return
	}

	if m.pageCount() <= 1 {
		m.FocusedAction = usePickerActionConfirm
		return
	}

	if m.Page < m.pageCount()-1 {
		m.FocusedAction = usePickerActionNext
		return
	}

	m.FocusedAction = usePickerActionConfirm
}

func (m UsePickerModel) renderButtons() string {
	actions := m.availableActions()
	labels := make([]string, 0, len(actions))

	for _, action := range actions {
		labels = append(labels, renderUsePickerButton(m.actionLabel(action), m.FocusedAction == action))
	}

	return strings.Join(labels, "   ")
}

func (m UsePickerModel) availableActions() []usePickerAction {
	if m.pageCount() <= 1 {
		return []usePickerAction{
			usePickerActionConfirm,
			usePickerActionCancel,
		}
	}

	if m.Page == 0 {
		return []usePickerAction{
			usePickerActionNext,
			usePickerActionCancel,
		}
	}

	if m.Page == m.pageCount()-1 {
		return []usePickerAction{
			usePickerActionPrev,
			usePickerActionConfirm,
			usePickerActionCancel,
		}
	}

	return []usePickerAction{
		usePickerActionPrev,
		usePickerActionNext,
		usePickerActionCancel,
	}
}

func (m UsePickerModel) actionLabel(action usePickerAction) string {
	switch action {
	case usePickerActionPrev:
		return "Prev"
	case usePickerActionNext:
		return "Next"
	case usePickerActionConfirm:
		return "Confirm"
	case usePickerActionCancel:
		return "Cancel"
	default:
		return "Unknown"
	}
}

func renderUsePickerButton(label string, focused bool) string {
	if focused {
		return "[ " + label + " ]"
	}

	return "  " + label + "  "
}

func (m UsePickerModel) visibleRange() (int, int) {
	flagsPerPage := m.flagsPerPage()

	start := m.Page * flagsPerPage
	if start > len(m.Selections) {
		start = len(m.Selections)
	}

	end := start + flagsPerPage
	if end > len(m.Selections) {
		end = len(m.Selections)
	}

	return start, end
}

func (m UsePickerModel) flagsPerPage() int {
	height := m.Height
	if height <= 0 {
		height = defaultTerminalHeight
	}

	available := height - usePickerReservedRows
	if available < minFlagsPerPage {
		return minFlagsPerPage
	}

	return available
}

func (m UsePickerModel) pageCount() int {
	if len(m.Selections) == 0 {
		return 1
	}

	flagsPerPage := m.flagsPerPage()
	if flagsPerPage <= 0 {
		flagsPerPage = minFlagsPerPage
	}

	return int(math.Ceil(float64(len(m.Selections)) / float64(flagsPerPage)))
}

func (m *UsePickerModel) clampPageAndCursor() {
	if len(m.Selections) == 0 {
		m.Page = 0
		m.Cursor = 0
		return
	}

	pageCount := m.pageCount()
	if m.Page >= pageCount {
		m.Page = pageCount - 1
	}

	if m.Page < 0 {
		m.Page = 0
	}

	start, end := m.visibleRange()

	if m.Cursor < start {
		m.Cursor = start
	}

	if m.Cursor >= end {
		m.Cursor = end - 1
	}

	if m.Cursor < 0 {
		m.Cursor = 0
	}
}

func (m UsePickerModel) hasInstalledColumn() bool {
	for _, flag := range m.Selections {
		if flag.Installed != nil {
			return true
		}
	}

	return false
}

func RunUsePicker(atom string, selections []useflags.FlagSelection) ([]useflags.FlagSelection, bool, error) {
	if len(selections) == 0 {
		return selections, true, nil
	}

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