ackage portage

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

type RequiredLicenseChange struct {
	Atom       string
	Licenses   []string
	RequiredBy []string
	Raw        string
}

type AutounmaskReport struct {
	RequiredUseChanges     []RequiredUseChange
	RequiredLicenseChanges []RequiredLicenseChange
}

func ParseAutounmaskReport(raw string) *AutounmaskReport {
	report := &AutounmaskReport{
		RequiredUseChanges:     parseRequiredUseChanges(raw),
		RequiredLicenseChanges: parseRequiredLicenseChanges(raw),
	}

	if len(report.RequiredUseChanges) == 0 && len(report.RequiredLicenseChanges) == 0 {
		return nil
	}

	return report
}

func parseRequiredUseChanges(raw string) []RequiredUseChange {
	if !strings.Contains(raw, "The following USE changes are necessary to proceed:") {
		return nil
	}

	lines := strings.Split(raw, "\n")

	useChangePattern := regexp.MustCompile(`^\s*([<>=~A-Za-z0-9_+./:-]+)\s+(.+?)\s*$`)
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

		if trimmed == "" || strings.HasPrefix(trimmed, "(see ") {
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

func parseRequiredLicenseChanges(raw string) []RequiredLicenseChange {
	if !strings.Contains(raw, "The following license changes are necessary to proceed:") {
		return nil
	}

	lines := strings.Split(raw, "\n")

	licenseChangePattern := regexp.MustCompile(`^\s*([<>=~A-Za-z0-9_+./:-]+)\s+(.+?)\s*$`)
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

		if trimmed == "" || strings.HasPrefix(trimmed, "(see ") {
			continue
		}

		if matches := requiredByPattern.FindStringSubmatch(trimmed); matches != nil {
			requiredBy = append(requiredBy, strings.TrimSpace(matches[1]))
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		matches := licenseChangePattern.FindStringSubmatch(trimmed)
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
		value = strings.Trim(value, ",")

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