package portage

import (
	"regexp"
	"strings"
)

type MergeAction string

const (
	MergeActionNew       MergeAction = "new"
	MergeActionUpdate    MergeAction = "update"
	MergeActionReinstall MergeAction = "reinstall"
	MergeActionNewSlot   MergeAction = "new-slot"
	MergeActionUnknown   MergeAction = "unknown"
)

type MergePackage struct {
	Atom       string
	Repository string
	Action     MergeAction
	Binary     bool
	Raw        string
}

type MergeTransaction struct {
	Packages     []MergePackage
	TotalLine    string
	DownloadSize string
}

func ParseMergeTransaction(raw string) *MergeTransaction {
	lines := strings.Split(raw, "\n")

	// Examples:
	// [binary  N g   ] x11-base/xorg-proto-2025.1-1::gentoo  USE="-test" 280 KiB
	// [ebuild  N    ~] media-video/obs-studio-32.1.2::gentoo  USE="..." 333,933 KiB
	// [binary  NSg   ] dev-lang/lua-5.4.8-6:5.4::gentoo [5.1.5-r200:5.1::gentoo] USE="deprecated readline" 260 KiB
	packagePattern := regexp.MustCompile(`^\[(binary|ebuild)\s+([A-Z]+)[^\]]*\]\s+([A-Za-z0-9_+./-]+)-[^:\s]+(?::[^:\s]+)?::([A-Za-z0-9_+.-]+)`)
	totalPattern := regexp.MustCompile(`^Total:\s+(.+)$`)
	sizePattern := regexp.MustCompile(`Size of downloads:\s+([^,]+(?:,\d{3})*\s+KiB)`)

	transaction := &MergeTransaction{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if matches := packagePattern.FindStringSubmatch(trimmed); matches != nil {
			sourceType := matches[1]
			actionCode := matches[2]
			atom := matches[3]
			repository := matches[4]

			transaction.Packages = append(transaction.Packages, MergePackage{
				Atom:       atom,
				Repository: repository,
				Action:     mergeActionFromCode(actionCode),
				Binary:     sourceType == "binary",
				Raw:        trimmed,
			})

			continue
		}

		if matches := totalPattern.FindStringSubmatch(trimmed); matches != nil {
			transaction.TotalLine = strings.TrimSpace(matches[1])

			if sizeMatches := sizePattern.FindStringSubmatch(trimmed); sizeMatches != nil {
				transaction.DownloadSize = strings.TrimSpace(sizeMatches[1])
			}
		}
	}

	if len(transaction.Packages) == 0 && transaction.TotalLine == "" {
		return nil
	}

	return transaction
}

func mergeActionFromCode(code string) MergeAction {
	switch {
	case strings.Contains(code, "NS"):
		return MergeActionNewSlot
	case strings.Contains(code, "N"):
		return MergeActionNew
	case strings.Contains(code, "U"):
		return MergeActionUpdate
	case strings.Contains(code, "R"):
		return MergeActionReinstall
	default:
		return MergeActionUnknown
	}
}

func (t *MergeTransaction) PackageAtoms() []string {
	if t == nil {
		return nil
	}

	atoms := make([]string, 0, len(t.Packages))

	for _, pkg := range t.Packages {
		atoms = append(atoms, pkg.Atom)
	}

	return atoms
}
