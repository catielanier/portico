package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/catielanier/portico/internal/i18n"
	"github.com/catielanier/portico/internal/jokes"
	"github.com/catielanier/portico/internal/plan"
	"github.com/catielanier/portico/internal/portage"
	"github.com/catielanier/portico/internal/ui"
	"github.com/catielanier/portico/internal/useflags"
	"github.com/spf13/cobra"
)

var rebuildCmd = &cobra.Command{
	Use:   "rebuild <atom...>",
	Short: "Revise USE flags and rebuild one or more packages",
	Args:  validateOneOrMoreAtomArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireRoot("rebuild packages"); err != nil {
			return err
		}

		atoms := cleanInstallArgs(args)

		if err := syncRepositoriesForMutation(); err != nil {
			return err
		}

		var sandbox *portage.ConfigSandbox

		if err := ui.RunStep("Creating temporary Portage config sandbox", func() error {
			var err error
			sandbox, err = portage.NewConfigSandbox()
			return err
		}); err != nil {
			return err
		}
		defer sandbox.Cleanup()

		maskActions := NewInstallMaskActions()

		if err := resolveInitialRebuildMasksInSandbox(atoms, sandbox, maskActions); err != nil {
			return err
		}

		queryResults := make([]*portage.PackageQuery, 0, len(atoms))
		selectedFlagsByAtom := make(map[string][]string)
		selectionsByAtom := make(map[string][]useflags.FlagSelection)
		packageUsePaths := make([]string, 0, len(atoms))

		querier := portage.NewEqueryQuerier()

		for _, atom := range atoms {
			queryResult, err := querier.Query(atom)
			if err != nil {
				return err
			}

			queryResults = append(queryResults, queryResult)

			selections := useflags.FromQuery(queryResult)

			selected, ok, err := ui.RunUsePicker(atom, selections)
			if err != nil {
				return err
			}

			if !ok {
				fmt.Println("Rebuild cancelled.")
				return nil
			}

			selectedFlags := useflags.SelectedFlags(selected)

			packageUsePath, err := portage.WritePackageUseEntry(
				sandbox.PortageConfigPath,
				atom,
				selectedFlags,
			)
			if err != nil {
				return err
			}

			selectedFlagsByAtom[atom] = selectedFlags
			selectionsByAtom[atom] = selected
			packageUsePaths = append(packageUsePaths, packageUsePath)
		}

		pretendResolution, err := resolveRebuildPretendProblemsInSandbox(atoms, sandbox, maskActions)
		if err != nil {
			return err
		}

		t, err := i18n.New("en")
		if err != nil {
			return err
		}

		transaction := (*portage.MergeTransaction)(nil)
		if pretendResolution.Result != nil {
			transaction = portage.ParseMergeTransaction(pretendResolution.Result.Raw)
		}

		renderRebuildPrototype(
			atoms,
			queryResults,
			selectionsByAtom,
			selectedFlagsByAtom,
			packageUsePaths,
			maskActions,
			pretendResolution.RequiredUseChanges,
			transaction,
			pretendResolution.Result,
			pretendResolution.Err,
			t,
		)

		if pretendResolution.Err != nil {
			return pretendResolution.Err
		}

		confirmed, err := confirmDefaultNo("Apply configuration and rebuild these packages?")
		if err != nil {
			return err
		}

		if !confirmed {
			fmt.Println("Rebuild cancelled.")
			return nil
		}

		if err := ui.RunStep("Writing Portage configuration", func() error {
			_, err := applyInstallConfigToSystem(
				selectedFlagsByAtom,
				maskActions,
				pretendResolution.RequiredUseChanges,
			)
			return err
		}); err != nil {
			return err
		}

		totalPackages := 0
		if transaction != nil {
			totalPackages = len(transaction.Packages)
		}

		if err := runPackageRebuild(atoms, totalPackages); err != nil {
			return err
		}

		fmt.Println()
		fmt.Println("Rebuild complete.")

		return nil
	},
}

func resolveInitialRebuildMasksInSandbox(
	atoms []string,
	sandbox *portage.ConfigSandbox,
	maskActions *InstallMaskActions,
) error {
	const maxAttempts = 8

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		var initialPretendResult *portage.PretendResult
		var initialPretendErr error

		label := "Checking package availability"
		if attempt > 1 {
			label = fmt.Sprintf("Checking package availability retry %d", attempt)
		}

		if err := ui.RunStep(label, func() error {
			initialPretendResult, initialPretendErr = portage.EmergePretendOneshotWithConfigRootForAtoms(atoms, sandbox.Root)

			if initialPretendErr != nil && initialPretendResult == nil {
				return initialPretendErr
			}

			return nil
		}); err != nil {
			return err
		}

		if initialPretendErr == nil {
			return nil
		}

		if initialPretendResult == nil {
			return initialPretendErr
		}

		maskReport := portage.ParseMaskedPackageReport("", initialPretendResult.Raw)
		if maskReport == nil {
			return nil
		}

		if err := applyMaskedPackageReportInSandbox(maskReport, sandbox, maskActions, initialPretendErr); err != nil {
			return err
		}
	}

	return fmt.Errorf("package availability did not resolve after %d attempts", maxAttempts)
}

func resolveRebuildPretendProblemsInSandbox(
	atoms []string,
	sandbox *portage.ConfigSandbox,
	maskActions *InstallMaskActions,
) (*PretendResolution, error) {
	const maxAttempts = 8

	resolution := &PretendResolution{}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		var pretendResult *portage.PretendResult
		var pretendErr error

		label := "Running emerge --pretend --oneshot"
		if attempt > 1 {
			label = fmt.Sprintf("Running emerge --pretend --oneshot retry %d", attempt)
		}

		if err := ui.RunStep(label, func() error {
			pretendResult, pretendErr = portage.EmergePretendOneshotWithConfigRootForAtoms(atoms, sandbox.Root)

			if pretendErr != nil && pretendResult == nil {
				return pretendErr
			}

			return nil
		}); err != nil {
			return nil, err
		}

		resolution.Result = pretendResult
		resolution.Err = pretendErr

		if pretendErr == nil {
			return resolution, nil
		}

		if pretendResult == nil {
			return resolution, nil
		}

		useReport := portage.ParseAutounmaskReport(pretendResult.Raw)
		if useReport != nil && len(useReport.RequiredUseChanges) > 0 {
			fmt.Println()
			fmt.Println("Portage requires additional USE changes to proceed:")
			fmt.Println()

			for _, change := range useReport.RequiredUseChanges {
				fmt.Printf("  %s %s\n", change.Atom, strings.Join(change.Flags, " "))

				for _, requiredBy := range change.RequiredBy {
					fmt.Printf("    required by: %s\n", requiredBy)
				}
			}

			fmt.Println()
			fmt.Println("Portico will apply these changes to the temporary sandbox and retry.")

			for _, change := range useReport.RequiredUseChanges {
				if _, err := portage.WritePackageUseEntry(
					sandbox.PortageConfigPath,
					change.Atom,
					change.Flags,
				); err != nil {
					return nil, err
				}

				resolution.RequiredUseChanges = append(resolution.RequiredUseChanges, change)
			}

			continue
		}

		maskReport := portage.ParseMaskedPackageReport("", pretendResult.Raw)
		if maskReport != nil {
			if err := applyMaskedPackageReportInSandbox(maskReport, sandbox, maskActions, pretendErr); err != nil {
				return nil, err
			}

			continue
		}

		return resolution, nil
	}

	return resolution, fmt.Errorf("emerge --pretend --oneshot did not resolve after %d attempts", maxAttempts)
}

func runPackageRebuild(atoms []string, totalPackages int) error {
	installer := portage.NewEmergeInstaller()

	label := "Rebuilding " + strings.Join(atoms, " ")

	return ui.RunInstallProgress(label, totalPackages, func(ctx context.Context, events chan<- ui.InstallProgressEvent) error {
		return installer.RebuildAtomsContext(ctx, atoms, totalPackages, func(progress portage.InstallProgress) {
			event := ui.InstallProgressEvent{
				CurrentPackage: progress.CurrentPackage,
				CurrentIndex:   progress.CurrentIndex,
				Total:          progress.Total,
			}

			select {
			case events <- event:
			case <-ctx.Done():
			}
		})
	})
}

func renderRebuildPrototype(
	atoms []string,
	queryResults []*portage.PackageQuery,
	selectionsByAtom map[string][]useflags.FlagSelection,
	selectedFlagsByAtom map[string][]string,
	packageUsePaths []string,
	maskActions *InstallMaskActions,
	requiredUseChanges []portage.RequiredUseChange,
	transaction *portage.MergeTransaction,
	pretendResult *portage.PretendResult,
	pretendErr error,
	t *i18n.Translator,
) {
	action := "Rebuild " + strings.Join(atoms, " ")

	p := plan.Plan{
		TitleKey: "plan_title",
		Action:   action,
		Will: []plan.Item{
			{
				Key: "will_inspect_use_flags",
				Data: map[string]any{
					"Atom": strings.Join(atoms, " "),
				},
			},
			{
				Key: "will_create_config_sandbox",
			},
			{
				Key: "will_write_sandbox_package_use",
			},
			{
				Key: "will_run_emerge_pretend",
				Data: map[string]any{
					"Atom": "--oneshot " + strings.Join(atoms, " "),
				},
			},
			{
				Key: "will_show_portage_transaction",
			},
		},
		WillNot: []plan.Item{
			{
				Key: "will_not_modify_global_use",
				Data: map[string]any{
					"Path": "/etc/portage/make.conf",
				},
			},
			{
				Key: "will_not_overwrite_package_use",
			},
			{
				Key: "will_not_add_to_world",
			},
			{
				Key: jokes.RandomKey(jokes.Context{
					Atom:    atoms[0],
					Command: "rebuild",
				}),
			},
		},
	}

	fmt.Println("Portico Rebuild")
	fmt.Println()
	fmt.Println("Packages:")

	for _, atom := range atoms {
		fmt.Printf("  %s\n", atom)
	}

	for _, queryResult := range queryResults {
		fmt.Println()
		renderUseFlagSummary(queryResult)
	}

	renderInstallMaskActions(maskActions)
	renderRequiredUseChanges(requiredUseChanges)

	fmt.Println()
	fmt.Println("Selected package.use entries:")

	hasSelectedFlags := false
	for _, atom := range atoms {
		selectedFlags := selectedFlagsByAtom[atom]
		if len(selectedFlags) == 0 {
			continue
		}

		hasSelectedFlags = true
		fmt.Printf("  %s %s\n", atom, strings.Join(selectedFlags, " "))
	}

	if !hasSelectedFlags {
		fmt.Println("  No explicit USE flag changes selected.")
	}

	if len(packageUsePaths) > 0 {
		fmt.Println()
		fmt.Println("Sandbox package.use paths:")

		for _, path := range dedupeStrings(packageUsePaths) {
			fmt.Printf("  %s\n", path)
		}
	}

	fmt.Println()

	if transaction != nil {
		renderMergeTransaction(transaction)
	}

	if transaction == nil && pretendResult != nil && pretendResult.Raw != "" {
		fmt.Println("emerge --pretend --oneshot output:")
		fmt.Println()
		fmt.Print(pretendResult.Raw)

		if !strings.HasSuffix(pretendResult.Raw, "\n") {
			fmt.Println()
		}
	}

	if transaction == nil && pretendResult != nil && pretendResult.Raw == "" {
		fmt.Println("emerge --pretend --oneshot output:")
		fmt.Println()
		fmt.Println("  No output returned.")
	}

	if pretendErr != nil {
		fmt.Println()
		fmt.Printf("Pretend result: %v\n", pretendErr)
	}

	fmt.Println()
	fmt.Print(ui.RenderPlanWithoutConfirmation(p, t))

	_ = selectionsByAtom
}
