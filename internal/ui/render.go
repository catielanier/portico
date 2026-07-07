package ui

import (
	"strings"

	"github.com/catielanier/portico/internal/i18n"
	"github.com/catielanier/portico/internal/plan"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	okStyle    = lipgloss.NewStyle()
	noStyle    = lipgloss.NewStyle()
)

func RenderPlan(p plan.Plan, t *i18n.Translator) string {
	var b strings.Builder

	title := p.TitleKey
	if title == "" {
		title = "plan_title"
	}

	b.WriteString(titleStyle.Render(t.T(title, nil)))
	b.WriteString("\n\n")

	if p.Action != "" {
		b.WriteString("Action:\n")
		b.WriteString("  " + p.Action + "\n\n")
	}

	b.WriteString(t.T("plan_will", nil))
	b.WriteString("\n\n")

	for _, item := range p.Will {
		b.WriteString("  " + okStyle.Render("✓") + " " + t.T(item.Key, item.Data) + "\n")
		if item.Detail != "" {
			b.WriteString("    " + item.Detail + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(t.T("plan_will_not", nil))
	b.WriteString("\n\n")

	for _, item := range p.WillNot {
		b.WriteString("  " + noStyle.Render("✗") + " " + t.T(item.Key, item.Data) + "\n")
		if item.Detail != "" {
			b.WriteString("    " + item.Detail + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(t.T("confirm_continue", nil))
	b.WriteString("\n")

	return b.String()
}
