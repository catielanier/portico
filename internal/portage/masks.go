package portage

import (
	"regexp"
	"strings"
)

type MaskReason string

const (
	MaskReasonTestingKeyword MaskReason = "testing-keyword"
	MaskReasonMissingKeyword MaskReason = "missing-keyword"
	MaskReasonLicense        MaskReason = "license"
	MaskReasonUnknown        MaskReason = "unknown"
)

type MaskedCandidate struct {
	Atom             string
	Version          string
	Repository       string
	Reasons          []MaskReason
	RawReason        string
	RequiredKeyword  string
	RequiredLicenses []string
	Live             bool
	Raw              string
}

type MaskedPackageReport struct {
	RequestedAtom string
	Candidates    []MaskedCandidate
}

func ParseMaskedPackageReport(requestedAtom string, raw string) *MaskedPackageReport {
	if !strings.Contains(raw, "All ebuilds that could satisfy") {
		return nil
	}

	requestedAtom = strings.TrimSpace(requestedAtom)
	if requestedAtom == "" {
		requestedAtom = extractRequestedAtomFromMaskedReport(raw)
	}

	lines := strings.Split(raw, "\n")

	// Examples:
	// !!! All ebuilds that could satisfy "media-video/obs-studio" have been masked.
	// - media-video/obs-studio-32.1.2::gentoo (masked by: ~amd64 keyword)
	// - app-example/foo-1.0::gentoo (masked by: GPL-3 license)
	// - app-example/foo-1.0::gentoo (masked by: ~amd64 keyword, GPL-3 license)
	candidatePattern := regexp.MustCompile(`^\s*-\s+(.+?)::([^ ]+)\s+\(masked by:\s+(.+?)\)\s*$`)

	report := &MaskedPackageReport{
		RequestedAtom: requestedAtom,
	}

	for _, line := range lines {
		matches := candidatePattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		fullAtomWithVersion := strings.TrimSpace(matches[1])
		repository := strings.TrimSpace(matches[2])
		rawReason := strings.TrimSpace(matches[3])
		version := versionFromVersionedAtom(requestedAtom, fullAtomWithVersion)

		candidate := MaskedCandidate{
			Atom:       fullAtomWithVersion,
			Version:    version,
			Repository: repository,
			RawReason:  rawReason,
			Live:       strings.Contains(version, "9999") || strings.HasSuffix(fullAtomWithVersion, "-9999"),
			Raw:        line,
		}

		candidate.Reasons = classifyMaskReasons(rawReason)
		candidate.RequiredKeyword = extractRequiredKeyword(rawReason)
		candidate.RequiredLicenses = extractRequiredLicenses(rawReason)

		report.Candidates = append(report.Candidates, candidate)
	}

	if len(report.Candidates) == 0 {
		return nil
	}

	return report
}

func extractRequestedAtomFromMaskedReport(raw string) string {
	pattern := regexp.MustCompile(`All ebuilds that could satisfy\s+"([^"]+)"\s+have been masked`)
	matches := pattern.FindStringSubmatch(raw)
	if matches == nil {
		return ""
	}

	return strings.TrimSpace(matches[1])
}

func classifyMaskReasons(reason string) []MaskReason {
	parts := splitMaskReasonParts(reason)

	reasons := make([]MaskReason, 0, len(parts))

	for _, part := range parts {
		switch {
		case strings.Contains(part, "~") && strings.Contains(part, "keyword"):
			reasons = append(reasons, MaskReasonTestingKeyword)

		case strings.Contains(part, "missing keyword"):
			reasons = append(reasons, MaskReasonMissingKeyword)

		case strings.Contains(part, "license"):
			reasons = append(reasons, MaskReasonLicense)

		default:
			reasons = append(reasons, MaskReasonUnknown)
		}
	}

	if len(reasons) == 0 {
		return []MaskReason{MaskReasonUnknown}
	}

	return reasons
}

func extractRequiredKeyword(reason string) string {
	parts := splitMaskReasonParts(reason)

	for _, part := range parts {
		fields := strings.Fields(part)

		for _, field := range fields {
			if strings.HasPrefix(field, "~") {
				return field
			}
		}
	}

	return ""
}

func extractRequiredLicenses(reason string) []string {
	parts := splitMaskReasonParts(reason)

	var licenses []string

	for _, part := range parts {
		if !strings.Contains(part, "license") {
			continue
		}

		part = strings.TrimSpace(part)
		part = strings.TrimSuffix(part, "license")
		part = strings.TrimSuffix(part, "licenses")
		part = strings.TrimSuffix(part, "license(s)")
		part = strings.TrimSpace(part)

		if part == "" {
			continue
		}

		for _, token := range strings.Fields(part) {
			token = strings.TrimSpace(token)
			token = strings.Trim(token, ",")

			if token == "" {
				continue
			}

			licenses = append(licenses, token)
		}
	}

	return licenses
}

func splitMaskReasonParts(reason string) []string {
	rawParts := strings.Split(reason, ",")

	parts := make([]string, 0, len(rawParts))

	for _, part := range rawParts {
		part = strings.ToLower(strings.TrimSpace(part))
		if part == "" {
			continue
		}

		parts = append(parts, part)
	}

	return parts
}

func versionFromVersionedAtom(requestedAtom string, versionedAtom string) string {
	prefix := requestedAtom + "-"
	if strings.HasPrefix(versionedAtom, prefix) {
		return strings.TrimPrefix(versionedAtom, prefix)
	}

	return ""
}

func BestMaskedCandidate(report *MaskedPackageReport) *MaskedCandidate {
	if report == nil || len(report.Candidates) == 0 {
		return nil
	}

	// Prefer latest non-live ebuild from the list Portage gives us.
	// Portage usually lists newest first.
	for i := range report.Candidates {
		if !report.Candidates[i].Live {
			return &report.Candidates[i]
		}
	}

	return &report.Candidates[0]
}

func (c MaskedCandidate) HasReason(reason MaskReason) bool {
	for _, candidateReason := range c.Reasons {
		if candidateReason == reason {
			return true
		}
	}

	return false
}

func (c MaskedCandidate) HasUnsupportedReasons() bool {
	for _, reason := range c.Reasons {
		switch reason {
		case MaskReasonTestingKeyword, MaskReasonLicense:
			continue
		default:
			return true
		}
	}

	return false
}
