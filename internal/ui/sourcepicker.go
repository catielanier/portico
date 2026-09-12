package ui

import (
	"fmt"
	"strings"

	"github.com/catielanier/portico/internal/i18n"
	"github.com/catielanier/portico/internal/portage"
	tea "github.com/charmbracelet/bubbletea"
)

type PackageSourceSelection struct {
	Candidate portage.PackageSourceCandidate
	Cancelled bool
}

type packageSourcePickerModel struct {
	translator *i18n.Translator
	atom       string
	candidates []portage.PackageSourceCandidate
	cursor     int
	offset     int
	height     int
	selected   *portage.PackageSourceCandidate
	cancelled  bool
}

func PickPackageSource(
	atom string,
	candidates []portage.PackageSourceCandidate,
) (*PackageSourceSelection, error) {
	model := packageSourcePickerModel{
		translator: i18n.MustDefault(),
		atom:       atom,
		candidates: candidates,
		height:     10,
	}

	program := tea.NewProgram(model)

	finalModel, err := program.Run()
	if err != nil {
		return nil, err
	}

	result, ok := finalModel.(packageSourcePickerModel)
	if !ok {
		return nil, fmt.Errorf("unexpected package source picker model type")
	}

	if result.cancelled || result.selected == nil {
		return &PackageSourceSelection{
			Cancelled: true,
		}, nil
	}

	return &PackageSourceSelection{
		Candidate: *result.selected,
		Cancelled: false,
	}, nil
}

func (m packageSourcePickerModel) Init() tea.Cmd {
	return nil
}

func (m packageSourcePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height - 10
		if m.height < 5 {
			m.height = 5
		}

		if m.height > 15 {
			m.height = 15
		}

		m.ensureCursorVisible()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.cancelled = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			m.ensureCursorVisible()

		case "down", "j":
			if m.cursor < len(m.candidates)-1 {
				m.cursor++
			}
			m.ensureCursorVisible()

		case "pgup", "b":
			m.cursor -= m.height
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.ensureCursorVisible()

		case "pgdown", "f":
			m.cursor += m.height
			if m.cursor >= len(m.candidates) {
				m.cursor = len(m.candidates) - 1
			}
			m.ensureCursorVisible()

		case "home", "g":
			m.cursor = 0
			m.ensureCursorVisible()

		case "end", "G":
			m.cursor = len(m.candidates) - 1
			m.ensureCursorVisible()

		case "enter":
			if len(m.candidates) == 0 {
				m.cancelled = true
				return m, tea.Quit
			}

			selected := m.candidates[m.cursor]
			m.selected = &selected

			return m, tea.Quit
		}
	}

	return m, nil
}

func (m packageSourcePickerModel) View() string {
	var b strings.Builder

	b.WriteString(m.t("source_picker_title", map[string]any{
		"Atom": m.atom,
	}))
	b.WriteString("\n\n")

	if len(m.candidates) == 0 {
		b.WriteString(m.t("source_picker_empty", nil))
		b.WriteString("\n")
		return b.String()
	}

	b.WriteString(m.t("source_picker_header", nil))
	b.WriteString("\n")

	start := m.offset
	end := start + m.height

	if end > len(m.candidates) {
		end = len(m.candidates)
	}

	for i := start; i < end; i++ {
		candidate := m.candidates[i]

		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}

		b.WriteString(fmt.Sprintf(
			"%s %-18s %-16s %-12s %s\n",
			cursor,
			truncateSourcePickerText(candidate.Repository, 18),
			truncateSourcePickerText(candidate.Version, 16),
			truncateSourcePickerText(candidateStatus(candidate), 12),
			candidate.InstallTarget,
		))
	}

	if len(m.candidates) > m.height {
		b.WriteString("\n")
		b.WriteString(m.t("source_picker_scroll_position", map[string]any{
			"Current": m.cursor + 1,
			"Total":   len(m.candidates),
		}))
		b.WriteString("\n")
	}

	selected := m.candidates[m.cursor]

	b.WriteString("\n")
	b.WriteString(m.t("source_picker_selected_target", map[string]any{
		"Target": selected.InstallTarget,
	}))
	b.WriteString("\n")

	if selected.Masked && selected.MaskReason != "" {
		b.WriteString(m.t("source_picker_selected_masked", map[string]any{
			"Reason": selected.MaskReason,
		}))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.t("source_picker_help", nil))
	b.WriteString("\n")

	return b.String()
}

func (m *packageSourcePickerModel) ensureCursorVisible() {
	if m.cursor < 0 {
		m.cursor = 0
	}

	if m.cursor >= len(m.candidates) {
		m.cursor = len(m.candidates) - 1
	}

	if m.cursor < 0 {
		m.cursor = 0
	}

	if m.offset > m.cursor {
		m.offset = m.cursor
	}

	if m.cursor >= m.offset+m.height {
		m.offset = m.cursor - m.height + 1
	}

	if m.offset < 0 {
		m.offset = 0
	}
}

func candidateStatus(candidate portage.PackageSourceCandidate) string {
	switch {
	case candidate.Live:
		return "live"
	case candidate.Masked:
		return "masked"
	default:
		return "available"
	}
}

func truncateSourcePickerText(value string, width int) string {
	value = strings.TrimSpace(value)

	if width <= 0 {
		return ""
	}

	if len([]rune(value)) <= width {
		return value
	}

	runes := []rune(value)
	if width == 1 {
		return string(runes[:1])
	}

	return string(runes[:width-1]) + "…"
}

func (m packageSourcePickerModel) t(id string, data map[string]any) string {
	translator := m.translator
	if translator == nil {
		translator = i18n.MustDefault()
	}

	return translator.T(id, data)
}
