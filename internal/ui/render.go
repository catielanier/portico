package ui

import (
	"strings"

	"github.com/catielanier/portico/internal/plan"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	okStyle    = lipgloss.NewStyle()
	noStyle    = lipgloss.NewStyle()
)

func RenderPlan(p plan.Plan) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(p.Title))
	b.WriteString("\n\n")

	if p.Action != "" {
		b.WriteString("Action:\n")
		b.WriteString("  " + p.Action + "\n\n")
	}

	b.WriteString("Portico will:\n\n")
	for _, item := range p.Will {
		b.WriteString("  " + okStyle.Render("✓") + " " + item.Text + "\n")
		if item.Detail != "" {
			b.WriteString("    " + item.Detail + "\n")
		}
	}

	b.WriteString("\nPortico will not:\n\n")
	for _, item := range p.WillNot {
		b.WriteString("  " + noStyle.Render("✗") + " " + item.Text + "\n")
		if item.Detail != "" {
			b.WriteString("    " + item.Detail + "\n")
		}
	}

	b.WriteString("\nContinue? [y/N]\n")

	return b.String()
}
