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

type RequiredKeywordChange struct {
	Atom       string
	Keywords  []string
	RequiredBy []string
	Raw        string
}

type RequiredLicenseChange struct {
	Atom       string
	Licenses   []string
	RequiredBy []string
	Raw        string
}

type AutounmaskReport struct {
	RequiredUseChanges     []RequiredUseChange
	RequiredKeywordChanges []RequiredKeywordChange
	RequiredLicenseChanges []RequiredLicenseChange
}

func ParseAutounmaskReport(raw string) *AutounmaskReport {
	report := &AutounmaskReport{
		RequiredUseChanges:     parseRequiredUseChanges(raw),
		RequiredKeywordChanges: parseRequiredKeywordChanges(raw),
		RequiredLicenseChanges: parseRequiredLicenseChanges(raw),
	}

	if len(report.RequiredUseChanges) == 0 &&
		len(report.RequiredKeywordChanges) == 0 &&
		len(report.RequiredLicenseChanges) == 0 {
		return nil
	}

	return report
}

func parseRequiredUseChanges(raw string) []RequiredUseChange {
	if !strings.Contains(raw, "The following USE changes are necessary to proceed:") {
		return nil
	}

	lines := strings.Split(raw, "\n")

	changePattern := regexp.MustCompile(`^\s*(\S+)\s+(.+?)\s*$`)
	requiredByPattern := regexp.MustCompile(`^\s*#\s+required by\s+(.+?)\s*$`)

	var changes []RequiredUseChange
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

		if isAutounmaskSectionBoundary(trimmed) {
			break
		}

		if shouldSkipAutounmaskLine(trimmed) {
			continue
		}

		if matches := requiredByPattern.FindStringSubmatch(trimmed); matches != nil {
			requiredBy = append(requiredBy, strings.TrimSpace(matches[1]))
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		matches := changePattern.FindStringSubmatch(trimmed)
		if matches == nil {
			continue
		}

		atom := strings.TrimSpace(matches[1])
		flags := cleanAutounmaskValues(strings.Fields(strings.TrimSpace(matches[2])))

		if atom == "" || len(flags) == 0 {
			continue
		}

		changes = append(changes, RequiredUseChange{
			Atom:       atom,
			Flags:      flags,
			RequiredBy: append([]string(nil), requiredBy...),
			Raw:        trimmed,
		})

		requiredBy = nil
	}

	return changes
}

func parseRequiredKeywordChanges(raw string) []RequiredKeywordChange {
	if !strings.Contains(raw, "The following keyword changes are necessary to proceed:") {
		return nil
	}

	lines := strings.Split(raw, "\n")

	changePattern := regexp.MustCompile(`^\s*(\S+)\s+(.+?)\s*$`)
	requiredByPattern := regexp.MustCompile(`^\s*#\s+required by\s+(.+?)\s*$`)

	var changes []RequiredKeywordChange
	var requiredBy []string
	inKeywordChanges := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "The following keyword changes are necessary to proceed:") {
			inKeywordChanges = true
			requiredBy = nil
			continue
		}

		if !inKeywordChanges {
			continue
		}

		if isAutounmaskSectionBoundary(trimmed) {
			break
		}

		if shouldSkipAutounmaskLine(trimmed) {
			continue
		}

		if matches := requiredByPattern.FindStringSubmatch(trimmed); matches != nil {
			requiredBy = append(requiredBy, strings.TrimSpace(matches[1]))
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		matches := changePattern.FindStringSubmatch(trimmed)
		if matches == nil {
			continue
		}

		atom := strings.TrimSpace(matches[1])
		keywords := cleanAutounmaskValues(strings.Fields(strings.TrimSpace(matches[2])))

		if atom == "" || len(keywords) == 0 {
			continue
		}

		changes = append(changes, RequiredKeywordChange{
			Atom:       atom,
			Keywords:  keywords,
			RequiredBy: append([]string(nil), requiredBy...),
			Raw:        trimmed,
		})

		requiredBy = nil
	}

	return changes
}

func parseRequiredLicenseChanges(raw string) []RequiredLicenseChange {
	if !strings.Contains(raw, "The following license changes are necessary to proceed:") {
		return nil
	}

	lines := strings.Split(raw, "\n")

	changePattern := regexp.MustCompile(`^\s*(\S+)\s+(.+?)\s*$`)
	requiredByPattern := regexp.MustCompile(`^\s*#\s+required by\s+(.+?)\s*$`)

	var changes []RequiredLicenseChange
	var requiredBy []string
	inLicenseChanges := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "The following license changes are necessary to proceed:") {
			inLicenseChanges = true
			requiredBy = nil
			continue
		}

		if !inLicenseChanges {
			continue
		}

		if isAutounmaskSectionBoundary(trimmed) {
			break
		}

		if shouldSkipAutounmaskLine(trimmed) {
			continue
		}

		if matches := requiredByPattern.FindStringSubmatch(trimmed); matches != nil {
			requiredBy = append(requiredBy, strings.TrimSpace(matches[1]))
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		matches := changePattern.FindStringSubmatch(trimmed)
		if matches == nil {
			continue
		}

		atom := strings.TrimSpace(matches[1])
		licenses := cleanAutounmaskValues(strings.Fields(strings.TrimSpace(matches[2])))

		if atom == "" || len(licenses) == 0 {
			continue
		}

		changes = append(changes, RequiredLicenseChange{
			Atom:       atom,
			Licenses:   licenses,
			RequiredBy: append([]string(nil), requiredBy...),
			Raw:        trimmed,
		})

		requiredBy = nil
	}

	return changes
}

func shouldSkipAutounmaskLine(trimmed string) bool {
	if trimmed == "" {
		return true
	}

	if strings.HasPrefix(trimmed, "(see ") {
		return true
	}

	if strings.HasPrefix(trimmed, "For more information") {
		return true
	}

	return false
}

func isAutounmaskSectionBoundary(trimmed string) bool {
	if trimmed == "" {
		return false
	}

	if strings.HasPrefix(trimmed, "* ") ||
		strings.HasPrefix(trimmed, "NOTE:") ||
		strings.HasPrefix(trimmed, "!!!") ||
		strings.HasPrefix(trimmed, "Would you like to add") ||
		strings.HasPrefix(trimmed, "Autounmask") {
		return true
	}

	if strings.Contains(trimmed, "The following ") &&
		strings.Contains(trimmed, " changes are necessary to proceed:") {
		return true
	}

	return false
}

func cleanAutounmaskValues(values []string) []string {
	var out []string
	seen := make(map[string]bool)

	for _, value := range values {
		value = strings.TrimSpace(value)
		value = strings.TrimSuffix(value, ",")

		if value == "" {
			continue
		}

		if strings.HasPrefix(value, "#") {
			continue
		}

		if seen[value] {
			continue
		}

		seen[value] = true
		out = append(out, value)
	}

	return out
}