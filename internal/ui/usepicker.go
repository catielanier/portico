package ui

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/catielanier/portico/internal/i18n"
	"github.com/catielanier/portico/internal/portage"
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

	RequiredUseExpression *portage.RequiredUseExpression
	RequiredUseParseError error
	RequiredUseViolations []portage.RequiredUseViolation

	Done      bool
	Cancelled bool
}

func NewUsePickerModel(atom string, selections []useflags.FlagSelection, requiredUse string) UsePickerModel {
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

	if strings.TrimSpace(requiredUse) != "" {
		model.RequiredUseExpression, model.RequiredUseParseError = portage.ParseRequiredUseExpression(requiredUse)
	}

	model.refreshRequiredUseValidation()
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
	if m.hasRequiredUseNotice() {
		b.WriteString(m.renderRequiredUseNotice())
	} else {
		b.WriteString(m.renderHighlightedDescription())
	}
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

	wasBlocked := !m.canConfirm()

	m.Selections[m.Cursor].Selection = m.Selections[m.Cursor].Selection.Next()
	m.refreshRequiredUseValidation()
	m.clampPageAndCursor()

	if wasBlocked && m.canConfirm() && m.confirmIsDisplayed() {
		m.FocusedAction = usePickerActionConfirm
	}

	m.ensureFocusedActionIsAvailable()
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
		if !m.canConfirm() {
			m.ensureFocusedActionIsAvailable()
			return m, nil
		}

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

	m.FocusedAction = actions[m.primaryActionIndex(actions)]
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
	actions := m.displayActions()
	labels := make([]string, 0, len(actions))

	for _, action := range actions {
		disabled := action == usePickerActionConfirm && !m.canConfirm()
		label := m.actionLabel(action)
		if disabled {
			label = m.t("use_picker_confirm_disabled", nil)
		}

		labels = append(labels, renderUsePickerButton(label, m.FocusedAction == action, disabled))
	}

	return strings.Join(labels, "   ")
}

func (m UsePickerModel) displayActions() []usePickerAction {
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

func (m UsePickerModel) availableActions() []usePickerAction {
	display := m.displayActions()
	actions := make([]usePickerAction, 0, len(display))

	for _, action := range display {
		if action == usePickerActionConfirm && !m.canConfirm() {
			continue
		}

		actions = append(actions, action)
	}

	return actions
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

func renderUsePickerButton(label string, focused bool, disabled bool) string {
	if disabled {
		return "  " + label + "  "
	}

	if focused {
		return "[ " + label + " ]"
	}

	return "  " + label + "  "
}

func (m *UsePickerModel) refreshRequiredUseValidation() {
	m.RequiredUseViolations = nil

	if m.RequiredUseExpression == nil || m.RequiredUseParseError != nil {
		return
	}

	m.RequiredUseViolations = m.RequiredUseExpression.Violations(
		useflags.EffectiveEnabledMap(m.Selections),
	)
}

func (m UsePickerModel) canConfirm() bool {
	return len(m.RequiredUseViolations) == 0
}

func (m UsePickerModel) confirmIsDisplayed() bool {
	for _, action := range m.displayActions() {
		if action == usePickerActionConfirm {
			return true
		}
	}

	return false
}

func (m UsePickerModel) hasRequiredUseNotice() bool {
	return m.RequiredUseParseError != nil || len(m.RequiredUseViolations) > 0
}

func (m UsePickerModel) renderRequiredUseNotice() string {
	var lines []string

	if m.RequiredUseParseError != nil {
		lines = append(lines, m.t("use_picker_required_use_validation_unavailable", nil))
	} else {
		for _, violation := range m.RequiredUseViolations {
			lines = append(lines, m.renderRequiredUseViolation(violation))

			if len(violation.Context) > 0 {
				lines = append(lines, m.t("use_picker_required_use_active_condition", map[string]any{
					"Conditions": m.renderRequiredUseConditions(violation.Context),
				}))
			}
		}

		lines = append(lines, m.t("use_picker_required_use_blocked", nil))
	}

	width := m.descriptionWidth()
	wrapped := make([]string, 0, descriptionMaxRows)

	for _, line := range lines {
		wrapped = append(wrapped, wrapText(line, width)...)
	}

	if len(wrapped) > descriptionMaxRows {
		wrapped = append(wrapped[:descriptionMaxRows-1], "…")
	}

	var b strings.Builder
	b.WriteString(m.t("use_picker_required_use_heading", nil))
	b.WriteString("\n")

	for _, line := range wrapped {
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteString("\n")
	}

	for i := len(wrapped); i < descriptionMaxRows; i++ {
		b.WriteString("\n")
	}

	return b.String()
}

func (m UsePickerModel) renderRequiredUseViolation(violation portage.RequiredUseViolation) string {
	switch violation.Kind {
	case portage.RequiredUseFlag:
		key := "use_picker_required_use_flag_enabled"
		if violation.Negated {
			key = "use_picker_required_use_flag_disabled"
		}

		return m.t(key, map[string]any{
			"Flag": violation.Flag,
		})

	case portage.RequiredUseAnyOf:
		key := "use_picker_required_use_any_flags"
		if !violation.SimpleFlags {
			key = "use_picker_required_use_any_conditions"
		}

		return m.t(key, map[string]any{
			"Terms": strings.Join(violation.Terms, ", "),
		})

	case portage.RequiredUseExactlyOne:
		key := "use_picker_required_use_exactly_one_flags"
		if !violation.SimpleFlags {
			key = "use_picker_required_use_exactly_one_conditions"
		}

		return m.t(key, map[string]any{
			"Terms": strings.Join(violation.Terms, ", "),
		})

	case portage.RequiredUseAtMostOne:
		key := "use_picker_required_use_at_most_one_flags"
		if !violation.SimpleFlags {
			key = "use_picker_required_use_at_most_one_conditions"
		}

		return m.t(key, map[string]any{
			"Terms": strings.Join(violation.Terms, ", "),
		})

	default:
		return m.t("use_picker_required_use_expression_invalid", map[string]any{
			"Expression": violation.Expression,
		})
	}
}

func (m UsePickerModel) renderRequiredUseConditions(conditions []portage.RequiredUseCondition) string {
	parts := make([]string, 0, len(conditions))

	for _, condition := range conditions {
		state := m.t("use_picker_required_use_state_disabled", nil)
		if condition.Enabled {
			state = m.t("use_picker_required_use_state_enabled", nil)
		}

		parts = append(parts, m.t("use_picker_required_use_condition", map[string]any{
			"Flag":  condition.Flag,
			"State": state,
		}))
	}

	return strings.Join(parts, ", ")
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

func RunUsePicker(atom string, selections []useflags.FlagSelection, requiredUse string) ([]useflags.FlagSelection, bool, error) {
	if len(selections) == 0 {
		return selections, true, nil
	}

	model := NewUsePickerModel(atom, selections, requiredUse)
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
