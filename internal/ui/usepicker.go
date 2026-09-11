package ui

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/catielanier/portico/internal/i18n"
	"github.com/catielanier/portico/internal/useflags"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	defaultTerminalWidth  = 80
	defaultTerminalHeight = 24
	minFlagsPerPage       = 1
	usePickerReservedRows = 18
	descriptionMaxRows    = 4
)

type usePickerAction int

const (
	usePickerActionPrev usePickerAction = iota
	usePickerActionNext
	usePickerActionConfirm
	usePickerActionCancel
)

type UsePickerModel struct {
	Atom       string
	Selections []useflags.FlagSelection
	Translator *i18n.Translator

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
		Translator:    i18n.MustDefault(),
		Cursor:        0,
		Page:          0,
		Width:         defaultTerminalWidth,
		Height:        defaultTerminalHeight,
		FocusedAction: usePickerActionConfirm,
	}

	model.clampPageAndCursor()
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
		m.ensureFocusedActionIsAvailable()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.Cancelled = true
			return m, tea.Quit

		case "up", "k":
			m.moveCursorUp()
			return m, nil

		case "down", "j":
			m.moveCursorDown()
			return m, nil

		case "left", "h", "shift+tab":
			m.focusPreviousAction()
			return m, nil

		case "right", "l", "tab":
			m.focusNextAction()
			return m, nil

		case " ":
			m.toggleCurrentFlag()
			return m, nil

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

	b.WriteString(m.t("use_picker_title", map[string]any{
		"Atom": m.Atom,
	}))
	b.WriteString("\n\n")

	if pageCount > 1 {
		b.WriteString(m.t("use_picker_page", map[string]any{
			"Page":  m.Page + 1,
			"Pages": pageCount,
		}))
		b.WriteString("\n\n")
	}

	if m.hasInstalledColumn() {
		b.WriteString(m.t("use_picker_header_with_installed", nil))
	} else {
		b.WriteString(m.t("use_picker_header_without_installed", nil))
	}
	b.WriteString("\n")

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
	b.WriteString(m.renderHighlightedDescription())
	b.WriteString("\n")

	b.WriteString(m.t("use_picker_help_navigation", nil))
	b.WriteString("\n")
	b.WriteString(m.t("use_picker_help_buttons", nil))
	b.WriteString("\n\n")
	b.WriteString(m.renderButtons())
	b.WriteString("\n")

	return b.String()
}

func (m *UsePickerModel) moveCursorUp() {
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
		_, previousEnd := m.visibleRange()
		m.Cursor = previousEnd - 1
		m.ensureFocusedActionIsAvailable()
	}
}

func (m *UsePickerModel) moveCursorDown() {
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
		nextStart, _ := m.visibleRange()
		m.Cursor = nextStart
		m.ensureFocusedActionIsAvailable()
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
			m.ensureFocusedActionIsAvailable()
		}

		return m, nil

	case usePickerActionNext:
		if m.Page < m.pageCount()-1 {
			m.Page++
			start, _ := m.visibleRange()
			m.Cursor = start
			m.ensureFocusedActionIsAvailable()
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
		m.ensureFocusedActionIsAvailable()
		return m, nil
	}
}

func (m *UsePickerModel) focusPreviousAction() {
	actions := m.availableActions()
	if len(actions) == 0 {
		return
	}

	currentIndex := m.focusedActionIndex(actions)

	if currentIndex <= 0 {
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

	currentIndex := m.focusedActionIndex(actions)

	if currentIndex >= len(actions)-1 {
		m.FocusedAction = actions[0]
		return
	}

	m.FocusedAction = actions[currentIndex+1]
}

func (m UsePickerModel) focusedActionIndex(actions []usePickerAction) int {
	for i, action := range actions {
		if action == m.FocusedAction {
			return i
		}
	}

	return m.primaryActionIndex(actions)
}

func (m UsePickerModel) primaryActionIndex(actions []usePickerAction) int {
	primary := m.primaryAction()

	for i, action := range actions {
		if action == primary {
			return i
		}
	}

	return 0
}

func (m *UsePickerModel) ensureFocusedActionIsAvailable() {
	actions := m.availableActions()
	if len(actions) == 0 {
		m.FocusedAction = usePickerActionConfirm
		return
	}

	for _, action := range actions {
		if action == m.FocusedAction {
			return
		}
	}

	m.FocusedAction = m.primaryAction()
}

func (m *UsePickerModel) resetFocusedActionForPage() {
	m.FocusedAction = m.primaryAction()
	m.ensureFocusedActionIsAvailable()
}

func (m UsePickerModel) primaryAction() usePickerAction {
	if len(m.Selections) == 0 {
		return usePickerActionConfirm
	}

	if m.pageCount() <= 1 {
		return usePickerActionConfirm
	}

	if m.Page < m.pageCount()-1 {
		return usePickerActionNext
	}

	return usePickerActionConfirm
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
		return m.t("common_prev", nil)
	case usePickerActionNext:
		return m.t("common_next", nil)
	case usePickerActionConfirm:
		return m.t("common_confirm", nil)
	case usePickerActionCancel:
		return m.t("common_cancel", nil)
	default:
		return m.t("common_unknown", nil)
	}
}

func renderUsePickerButton(label string, focused bool) string {
	if focused {
		return "[ " + label + " ]"
	}

	return "  " + label + "  "
}

func (m UsePickerModel) renderHighlightedDescription() string {
	if len(m.Selections) == 0 {
		return ""
	}

	if m.Cursor < 0 || m.Cursor >= len(m.Selections) {
		return ""
	}

	description := strings.TrimSpace(m.Selections[m.Cursor].Description)
	if description == "" {
		description = m.t("use_picker_no_description", nil)
	}

	width := m.descriptionWidth()
	lines := wrapText(description, width)

	if len(lines) > descriptionMaxRows {
		lines = append(lines[:descriptionMaxRows-1], "…")
	}

	var b strings.Builder

	b.WriteString(m.t("use_picker_description_heading", nil))
	b.WriteString("\n")

	for _, line := range lines {
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteString("\n")
	}

	for i := len(lines); i < descriptionMaxRows; i++ {
		b.WriteString("\n")
	}

	return b.String()
}

func (m UsePickerModel) descriptionWidth() int {
	width := m.Width
	if width <= 0 {
		width = defaultTerminalWidth
	}

	width -= 2
	if width < 20 {
		return 20
	}

	return width
}

func (m UsePickerModel) visibleRange() (int, int) {
	if len(m.Selections) == 0 {
		return 0, 0
	}

	flagsPerPage := m.flagsPerPage()

	start := m.Page * flagsPerPage
	if start < 0 {
		start = 0
	}

	if start > len(m.Selections) {
		start = len(m.Selections)
	}

	end := start + flagsPerPage
	if end > len(m.Selections) {
		end = len(m.Selections)
	}

	if end < start {
		end = start
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
	if pageCount <= 0 {
		pageCount = 1
	}

	if m.Page < 0 {
		m.Page = 0
	}

	if m.Page >= pageCount {
		m.Page = pageCount - 1
	}

	start, end := m.visibleRange()

	if start == end {
		m.Cursor = start
		return
	}

	if m.Cursor < start {
		m.Cursor = start
	}

	if m.Cursor >= end {
		m.Cursor = end - 1
	}

	if m.Cursor < 0 {
		m.Cursor = 0
	}

	if m.Cursor >= len(m.Selections) {
		m.Cursor = len(m.Selections) - 1
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

func (m UsePickerModel) t(id string, data map[string]any) string {
	translator := m.Translator
	if translator == nil {
		translator = i18n.MustDefault()
	}

	return translator.T(id, data)
}

func wrapText(value string, width int) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	if width <= 0 {
		width = defaultTerminalWidth
	}

	words := strings.Fields(value)
	if len(words) == 0 {
		return nil
	}

	lines := make([]string, 0)
	current := ""

	for _, word := range words {
		if current == "" {
			current = word
			continue
		}

		if runeLen(current)+1+runeLen(word) <= width {
			current += " " + word
			continue
		}

		lines = append(lines, current)
		current = word
	}

	if current != "" {
		lines = append(lines, current)
	}

	return lines
}

func runeLen(value string) int {
	return utf8.RuneCountInString(value)
}

func RunUsePicker(atom string, selections []useflags.FlagSelection) ([]useflags.FlagSelection, bool, error) {
	if len(selections) == 0 {
		return selections, true, nil
	}

	model := NewUsePickerModel(atom, selections)
	program := tea.NewProgram(model, tea.WithAltScreen())

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
