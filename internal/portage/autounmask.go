package portage

import (
	"regexp"
	"strings"
)

type RequiredUseChange struct {
	Atom       string
	Flags      []string
	RequiredBy []string
	Raw        string
}

type AutounmaskReport struct {
	RequiredUseChanges []RequiredUseChange
}

func ParseAutounmaskReport(raw string) *AutounmaskReport {
	if !strings.Contains(raw, "The following USE changes are necessary to proceed:") {
		return nil
	}

	lines := strings.Split(raw, "\n")

	useChangePattern := regexp.MustCompile(`^\s*([<>=~A-Za-z0-9_+./:-]+)\s+(.+?)\s*$`)
	requiredByPattern := regexp.MustCompile(`^\s*#\s+required by\s+(.+?)\s*$`)

	report := &AutounmaskReport{}
	var requiredBy []string
	inUseChanges := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "The following USE changes are necessary to proceed:") {
			inUseChanges = true
			requiredBy = nil
			continue
		}

		if !inUseChanges {
			continue
		}

		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "* ") ||
			strings.HasPrefix(trimmed, "NOTE:") ||
			strings.HasPrefix(trimmed, "!!!") {
			break
		}

		if strings.HasPrefix(trimmed, "(see ") {
			continue
		}

		if matches := requiredByPattern.FindStringSubmatch(trimmed); matches != nil {
			requiredBy = append(requiredBy, strings.TrimSpace(matches[1]))
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		matches := useChangePattern.FindStringSubmatch(trimmed)
		if matches == nil {
			continue
		}

		atom := strings.TrimSpace(matches[1])
		flags := strings.Fields(strings.TrimSpace(matches[2]))

		if atom == "" || len(flags) == 0 {
			continue
		}

		change := RequiredUseChange{
			Atom:       atom,
			Flags:      flags,
			RequiredBy: append([]string(nil), requiredBy...),
			Raw:        trimmed,
		}

		report.RequiredUseChanges = append(report.RequiredUseChanges, change)
		requiredBy = nil
	}

	if len(report.RequiredUseChanges) == 0 {
		return nil
	}

	return report
}
