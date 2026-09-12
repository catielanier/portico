package cli

import (
	"fmt"
	"strings"

	"github.com/catielanier/portico/internal/i18n"
	"github.com/catielanier/portico/internal/portage"
	"github.com/catielanier/portico/internal/ui"
	"github.com/spf13/cobra"
)

func resolvePackageSourcesForInstall(
	cmd *cobra.Command,
	translator *i18n.Translator,
	atoms []string,
) ([]string, error) {
	resolved := make([]string, 0, len(atoms))

	for _, atom := range atoms {
		atom = strings.TrimSpace(atom)
		if atom == "" {
			continue
		}

		if hasExplicitRepositoryQualifier(atom) {
			resolved = append(resolved, atom)
			continue
		}

		report, err := portage.FindPackageSourceCandidates(atom)
		if err != nil {
			return nil, err
		}

		if !portage.NeedsPackageSourceSelection(report) {
			resolved = append(resolved, atom)
			continue
		}

		fmt.Fprintln(cmd.OutOrStdout(), translator.T("source_picker_found_multiple", map[string]any{
			"Atom": atom,
		}))

		selection, err := ui.PickPackageSource(atom, report.Candidates)
		if err != nil {
			return nil, err
		}

		if selection.Cancelled {
			return nil, fmt.Errorf("%s", translator.T("source_picker_cancelled", map[string]any{
				"Atom": atom,
			}))
		}

		resolved = append(resolved, selection.Candidate.InstallTarget)
	}

	return resolved, nil
}

func hasExplicitRepositoryQualifier(atom string) bool {
	atom = strings.TrimSpace(atom)
	if atom == "" {
		return false
	}

	beforeRepo, repository, found := strings.Cut(atom, "::")
	if !found {
		return false
	}

	return strings.TrimSpace(beforeRepo) != "" && strings.TrimSpace(repository) != ""
}
