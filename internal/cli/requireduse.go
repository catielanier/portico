package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/catielanier/portico/internal/i18n"
	"github.com/catielanier/portico/internal/portage"
	"github.com/catielanier/portico/internal/ui"
	"github.com/catielanier/portico/internal/useflags"
)

var errRequiredUseResolutionCancelled = errors.New("REQUIRED_USE resolution cancelled")

func resolveRequiredUseFailureInSandbox(
	raw string,
	sandbox *portage.ConfigSandbox,
	requestedAtoms []string,
	requiredUseChanges *[]portage.RequiredUseChange,
) (bool, error) {
	failure := portage.ParseRequiredUseFailure(raw)
	if failure == nil {
		return false, nil
	}

	translator := i18n.MustDefault()
	expression := strings.TrimSpace(failure.Expression())

	if expression == "" {
		return true, fmt.Errorf("%s", translator.T("required_use_fallback_missing_expression", map[string]any{
			"Atom": failure.Package,
		}))
	}

	if _, err := portage.ParseRequiredUseExpression(expression); err != nil {
		return true, fmt.Errorf("%s", translator.T("required_use_fallback_unparseable", map[string]any{
			"Atom":       failure.Package,
			"Expression": expression,
		}))
	}

	displayAtom := strings.TrimSpace(failure.Package)
	if displayAtom == "" {
		displayAtom = strings.TrimSpace(failure.PackageSpec)
	}
	if displayAtom == "" {
		displayAtom = "unknown package"
	}

	fmt.Println()
	fmt.Println(ui.Warning(translator.T("required_use_fallback_detected", map[string]any{
		"Atom": displayAtom,
	})))
	fmt.Println()
	fmt.Printf("  %s\n", ui.Warning(expression))
	fmt.Println()
	fmt.Println(ui.Info(translator.T("required_use_fallback_reopen_picker", nil)))
	fmt.Println()

	queryAtom := strings.TrimSpace(failure.ExactAtom)
	if queryAtom == "" {
		queryAtom = strings.TrimSpace(failure.Package)
	}
	if queryAtom == "" {
		return true, fmt.Errorf("%s", translator.T("required_use_fallback_package_unknown", nil))
	}

	querier := portage.NewEqueryQuerierWithConfigRoot(sandbox.Root)
	queryResult, err := querier.Query(queryAtom)
	if err != nil && failure.Package != "" && queryAtom != failure.Package {
		queryAtom = failure.Package
		queryResult, err = querier.Query(queryAtom)
	}
	if err != nil {
		return true, err
	}

	if len(queryResult.Uses) == 0 {
		return true, fmt.Errorf("%s", translator.T("required_use_fallback_no_use_flags", map[string]any{
			"Atom": displayAtom,
		}))
	}

	selections := useflags.FromQuery(queryResult)
	selected, ok, err := ui.RunUsePicker(displayAtom, selections, expression)
	if err != nil {
		return true, err
	}
	if !ok {
		fmt.Println(ui.Warning(translator.T("required_use_fallback_cancelled", nil)))
		return true, errRequiredUseResolutionCancelled
	}

	selectedFlags := useflags.SelectedFlags(selected)
	if len(selectedFlags) == 0 {
		return true, fmt.Errorf("%s", translator.T("required_use_fallback_no_changes", map[string]any{
			"Atom": displayAtom,
		}))
	}

	configAtom := preferredRequiredUseConfigAtom(failure, requestedAtoms)
	if configAtom == "" {
		configAtom = queryAtom
	}

	if _, err := portage.WritePackageUseEntry(
		sandbox.PortageConfigPath,
		configAtom,
		selectedFlags,
	); err != nil {
		return true, err
	}

	if requiredUseChanges != nil {
		*requiredUseChanges = appendRequiredUseChangeUnique(*requiredUseChanges, portage.RequiredUseChange{
			Atom:       configAtom,
			Flags:      selectedFlags,
			RequiredBy: append([]string(nil), failure.RequiredBy...),
			Raw:        strings.TrimSpace(failure.UnsatisfiedExpression),
		})
	}

	fmt.Println()
	fmt.Println(ui.Success(translator.T("required_use_fallback_applied", map[string]any{
		"Atom": configAtom,
	})))

	return true, nil
}

func preferredRequiredUseConfigAtom(
	failure *portage.RequiredUseFailure,
	requestedAtoms []string,
) string {
	if failure == nil {
		return ""
	}

	packageName := strings.TrimSpace(failure.Package)
	if packageName != "" {
		for _, atom := range requestedAtoms {
			atom = strings.TrimSpace(atom)
			if atom == "" {
				continue
			}

			if portage.PackageNameFromSpec(atom) == packageName {
				return atom
			}
		}
	}

	if strings.TrimSpace(failure.ExactAtom) != "" {
		return strings.TrimSpace(failure.ExactAtom)
	}

	return packageName
}

func isRequiredUseResolutionCancelled(err error) bool {
	return errors.Is(err, errRequiredUseResolutionCancelled)
}
