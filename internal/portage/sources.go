package portage

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type PackageSourceCandidate struct {
	RequestedAtom   string
	Package         string
	Version         string
	Repository      string
	InstallTarget   string
	Masked          bool
	MaskReason      string
	Live            bool
	Raw             string
	DiscoveryOutput string
}

type PackageSourceReport struct {
	RequestedAtom string
	Candidates    []PackageSourceCandidate
	Raw           string
}

func FindPackageSourceCandidates(atom string) (*PackageSourceReport, error) {
	atom = strings.TrimSpace(atom)
	if atom == "" {
		return nil, fmt.Errorf("atom cannot be empty")
	}

	raw, err := runPackageSourceDiscovery(atom)
	if err != nil {
		return nil, err
	}

	report := ParsePackageSourceReport(atom, raw)
	if report == nil {
		return &PackageSourceReport{
			RequestedAtom: atom,
			Raw:           raw,
		}, nil
	}

	return report, nil
}

func runPackageSourceDiscovery(atom string) (string, error) {
	cmd := exec.Command("emerge", "-pvO", atom)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := strings.TrimSpace(strings.Join([]string{
		stdout.String(),
		stderr.String(),
	}, "\n"))

	if err != nil {
		// `emerge -pvO` can exit non-zero when all available candidates are masked,
		// but that output is still useful for source discovery.
		if output != "" && strings.Contains(output, "All ebuilds that could satisfy") {
			return output, nil
		}

		return output, fmt.Errorf("emerge -pvO %s failed: %w", atom, err)
	}

	return output, nil
}

func ParsePackageSourceReport(requestedAtom string, raw string) *PackageSourceReport {
	requestedAtom = strings.TrimSpace(requestedAtom)
	raw = strings.TrimSpace(raw)

	if requestedAtom == "" || raw == "" {
		return nil
	}

	candidates := parsePackageSourceCandidates(requestedAtom, raw)
	if len(candidates) == 0 {
		return nil
	}

	return &PackageSourceReport{
		RequestedAtom: requestedAtom,
		Candidates:    latestPackageSourceCandidatePerRepository(candidates),
		Raw:           raw,
	}
}

func parsePackageSourceCandidates(requestedAtom string, raw string) []PackageSourceCandidate {
	lines := strings.Split(raw, "\n")

	var candidates []PackageSourceCandidate

	for _, line := range lines {
		candidate, ok := parsePackageSourceCandidateLine(requestedAtom, raw, line)
		if !ok {
			continue
		}

		candidates = append(candidates, candidate)
	}

	return candidates
}

func parsePackageSourceCandidateLine(
	requestedAtom string,
	raw string,
	line string,
) (PackageSourceCandidate, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return PackageSourceCandidate{}, false
	}

	if candidate, ok := parseMaskedPackageSourceCandidateLine(requestedAtom, raw, line); ok {
		return candidate, true
	}

	if candidate, ok := parseMergePackageSourceCandidateLine(requestedAtom, raw, line); ok {
		return candidate, true
	}

	return PackageSourceCandidate{}, false
}

func parseMaskedPackageSourceCandidateLine(
	requestedAtom string,
	raw string,
	line string,
) (PackageSourceCandidate, bool) {
	pattern := regexp.MustCompile(`^\s*-\s+(.+?)::([^ ]+)\s+\(masked by:\s+(.+?)\)\s*$`)
	matches := pattern.FindStringSubmatch(line)
	if matches == nil {
		return PackageSourceCandidate{}, false
	}

	versionedPackage := strings.TrimSpace(matches[1])
	repository := strings.TrimSpace(matches[2])
	maskReason := strings.TrimSpace(matches[3])

	return buildPackageSourceCandidate(
		requestedAtom,
		raw,
		line,
		versionedPackage,
		repository,
		true,
		maskReason,
	), true
}

func parseMergePackageSourceCandidateLine(
	requestedAtom string,
	raw string,
	line string,
) (PackageSourceCandidate, bool) {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`^\s*\[[^\]]+\]\s+(.+?)::([^ ]+).*$`),
		regexp.MustCompile(`^\s*\[[^\]]+\]\s+(.+?)\s+\[.+?\]\s+(.+?)::([^ ]+).*$`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		if len(matches) == 3 {
			versionedPackage := strings.TrimSpace(matches[1])
			repository := strings.TrimSpace(matches[2])

			return buildPackageSourceCandidate(
				requestedAtom,
				raw,
				line,
				versionedPackage,
				repository,
				false,
				"",
			), true
		}

		if len(matches) == 4 {
			versionedPackage := strings.TrimSpace(matches[2])
			repository := strings.TrimSpace(matches[3])

			return buildPackageSourceCandidate(
				requestedAtom,
				raw,
				line,
				versionedPackage,
				repository,
				false,
				"",
			), true
		}
	}

	return PackageSourceCandidate{}, false
}

func buildPackageSourceCandidate(
	requestedAtom string,
	raw string,
	line string,
	versionedPackage string,
	repository string,
	masked bool,
	maskReason string,
) PackageSourceCandidate {
	pkg := packageNameFromVersionedPackage(requestedAtom, versionedPackage)
	version := versionFromPackageSource(requestedAtom, pkg, versionedPackage)

	installTarget := versionedPackage + "::" + repository
	if version != "" {
		installTarget = "=" + installTarget
	}

	return PackageSourceCandidate{
		RequestedAtom:   requestedAtom,
		Package:         pkg,
		Version:         version,
		Repository:      repository,
		InstallTarget:   installTarget,
		Masked:          masked,
		MaskReason:      maskReason,
		Live:            version == "9999" || strings.HasSuffix(versionedPackage, "-9999"),
		Raw:             strings.TrimSpace(line),
		DiscoveryOutput: raw,
	}
}

func latestPackageSourceCandidatePerRepository(
	candidates []PackageSourceCandidate,
) []PackageSourceCandidate {
	seen := make(map[string]bool)
	latestByPortageOrder := make([]PackageSourceCandidate, 0, len(candidates))

	// Trust Portage's output ordering for version priority. Keep only the first
	// candidate seen for each repository, because `emerge -pvO` reports newer
	// candidates before older candidates for the same source.
	for _, candidate := range candidates {
		if candidate.Repository == "" {
			continue
		}

		if seen[candidate.Repository] {
			continue
		}

		seen[candidate.Repository] = true
		latestByPortageOrder = append(latestByPortageOrder, candidate)
	}

	return preferGentooCandidateFirst(latestByPortageOrder)
}

func preferGentooCandidateFirst(candidates []PackageSourceCandidate) []PackageSourceCandidate {
	if len(candidates) <= 1 {
		return candidates
	}

	gentooIndex := -1
	for i, candidate := range candidates {
		if candidate.Repository == "gentoo" {
			gentooIndex = i
			break
		}
	}

	if gentooIndex <= 0 {
		return candidates
	}

	out := make([]PackageSourceCandidate, 0, len(candidates))
	out = append(out, candidates[gentooIndex])
	out = append(out, candidates[:gentooIndex]...)
	out = append(out, candidates[gentooIndex+1:]...)

	return out
}

func packageNameFromVersionedPackage(requestedAtom string, versionedPackage string) string {
	requestedAtom = stripPackageSourceAtomDecoration(requestedAtom)
	versionedPackage = stripPackageSourceAtomDecoration(versionedPackage)

	if requestedAtom != "" && strings.HasPrefix(versionedPackage, requestedAtom+"-") {
		return requestedAtom
	}

	slashIndex := strings.Index(versionedPackage, "/")
	if slashIndex < 0 {
		return requestedAtom
	}

	category := versionedPackage[:slashIndex]
	nameAndVersion := versionedPackage[slashIndex+1:]

	versionStart := findVersionStart(nameAndVersion)
	if versionStart < 0 {
		return requestedAtom
	}

	return category + "/" + nameAndVersion[:versionStart-1]
}

func versionFromPackageSource(requestedAtom string, pkg string, versionedPackage string) string {
	requestedAtom = stripPackageSourceAtomDecoration(requestedAtom)
	pkg = stripPackageSourceAtomDecoration(pkg)
	versionedPackage = stripPackageSourceAtomDecoration(versionedPackage)

	if pkg != "" && strings.HasPrefix(versionedPackage, pkg+"-") {
		return strings.TrimPrefix(versionedPackage, pkg+"-")
	}

	if requestedAtom != "" && strings.HasPrefix(versionedPackage, requestedAtom+"-") {
		return strings.TrimPrefix(versionedPackage, requestedAtom+"-")
	}

	slashIndex := strings.Index(versionedPackage, "/")
	if slashIndex < 0 {
		return ""
	}

	nameAndVersion := versionedPackage[slashIndex+1:]
	versionStart := findVersionStart(nameAndVersion)
	if versionStart < 0 {
		return ""
	}

	return nameAndVersion[versionStart:]
}

func findVersionStart(nameAndVersion string) int {
	for i := 1; i < len(nameAndVersion)-1; i++ {
		if nameAndVersion[i] != '-' {
			continue
		}

		next := nameAndVersion[i+1]
		if next >= '0' && next <= '9' {
			return i + 1
		}
	}

	return -1
}

func stripPackageSourceAtomDecoration(atom string) string {
	atom = strings.TrimSpace(atom)
	atom = strings.TrimPrefix(atom, "=")

	if beforeRepo, _, ok := strings.Cut(atom, "::"); ok {
		atom = beforeRepo
	}

	return atom
}

func NeedsPackageSourceSelection(report *PackageSourceReport) bool {
	if report == nil {
		return false
	}

	repositories := make(map[string]bool)

	for _, candidate := range report.Candidates {
		if candidate.Repository == "" {
			continue
		}

		repositories[candidate.Repository] = true
	}

	return len(repositories) > 1
}
