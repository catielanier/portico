package ui

import (
	"fmt"
	"strings"

	"github.com/catielanier/portico/internal/i18n"
	"github.com/catielanier/portico/internal/plan"
)

func RenderPlan(p plan.Plan, t *i18n.Translator) string {
	return renderPlan(p, t, true)
}

func RenderPlanWithoutConfirmation(p plan.Plan, t *i18n.Translator) string {
	return renderPlan(p, t, false)
}

func renderPlan(p plan.Plan, t *i18n.Translator, includeConfirmation bool) string {
	var b strings.Builder

	title := p.TitleKey
	if title != "" {
		title = t.T(p.TitleKey, nil)
	}

	b.WriteString(title)
	b.WriteString("\n\n")

	if p.Action != "" {
		b.WriteString("Action:\n")
		b.WriteString(fmt.Sprintf("  %s\n\n", p.Action))
	}

	if len(p.Will) > 0 {
		b.WriteString(t.T("plan_will", nil))
		b.WriteString("\n\n")

		for _, item := range p.Will {
			b.WriteString("  ✓ ")
			b.WriteString(t.T(item.Key, item.Data))

			if item.Detail != "" {
				b.WriteString("\n")
				b.WriteString("    ")
				b.WriteString(item.Detail)
			}

			b.WriteString("\n")
		}

		b.WriteString("\n")
	}

	if len(p.WillNot) > 0 {
		b.WriteString(t.T("plan_will_not", nil))
		b.WriteString("\n\n")

		for _, item := range p.WillNot {
			b.WriteString("  ✗ ")
			b.WriteString(t.T(item.Key, item.Data))

			if item.Detail != "" {
				b.WriteString("\n")
				b.WriteString("    ")
				b.WriteString(item.Detail)
			}

			b.WriteString("\n")
		}

		b.WriteString("\n")
	}

	if includeConfirmation {
		b.WriteString(t.T("confirm_continue", nil))
		b.WriteString("\n")
	}

	return b.String()
}
