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
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update [atom[@version]...]",
	Short: "Update the world set or one or more packages",
	Args:  validateZeroOrMorePackageTargetArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireRoot("update packages"); err != nil {
			return err
		}

		atoms, err := packageAtomsFromArgsOrWorld(args)
		if err != nil {
			return err
		}

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

		pretendResolution, err := resolveUpdatePretendProblemsInSandbox(atoms, sandbox, maskActions)
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

		renderUpdatePlan(
			atoms,
			maskActions,
			transaction,
			pretendResolution.Result,
			pretendResolution.Err,
			t,
		)

		if pretendResolution.Err != nil {
			fmt.Println()
			fmt.Println("Portico can currently apply update-time license changes, but not USE changes or unsupported masks.")
			fmt.Println("Resolve the issue shown above, then run update again.")
			return pretendResolution.Err
		}

		confirmed, err := confirmDefaultNo("Run this update?")
		if err != nil {
			return err
		}

		if !confirmed {
			fmt.Println("Update cancelled.")
			return nil
		}

		if len(maskActions.LicenseEntries) > 0 {
			if err := ui.RunStep("Writing Portage license configuration", func() error {
				_, err := applyInstallConfigToSystem(
					map[string][]string{},
					maskActions,
					nil,
				)
				return err
			}); err != nil {
				return err
			}
		}

		totalPackages := 0
		if transaction != nil {
			totalPackages = len(transaction.Packages)
		}

		if err := runPackageUpdate(atoms, totalPackages); err != nil {
			return err
		}

		fmt.Println()
		fmt.Println("Update complete.")

		return nil
	},
}

func packageAtomsFromArgsOrWorld(args []string) ([]string, error) {
	if len(args) == 0 {
		return nil, nil
	}

	return packageAtomsFromArgs(args)
}

func resolveUpdatePretendProblemsInSandbox(
	atoms []string,
	sandbox *portage.ConfigSandbox,
	maskActions *InstallMaskActions,
) (*PretendResolution, error) {
	const maxAttempts = 8

	resolution := &PretendResolution{}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		var pretendResult *portage.PretendResult
		var pretendErr error

		label := "Running emerge --pretend --update --deep --newuse"
		if attempt > 1 {
			label = fmt.Sprintf("Running emerge --pretend --update --deep --newuse retry %d", attempt)
		}

		if err := ui.RunStep(label, func() error {
			pretendResult, pretendErr = portage.EmergePretendUpdateWithConfigRootForAtoms(atoms, sandbox.Root)

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

		autounmaskReport := portage.ParseAutounmaskReport(pretendResult.Raw)
		if autounmaskReport != nil && len(autounmaskReport.RequiredLicenseChanges) > 0 {
			if err := applyRequiredLicenseChangesInSandbox(
				autounmaskReport.RequiredLicenseChanges,
				sandbox,
				maskActions,
			); err != nil {
				return nil, err
			}

			continue
		}

		return resolution, nil
	}

	return resolution, fmt.Errorf("emerge --pretend --update --deep --newuse did not resolve after %d attempts", maxAttempts)
}

func renderUpdatePlan(
	atoms []string,
	maskActions *InstallMaskActions,
	transaction *portage.MergeTransaction,
	pretendResult *portage.PretendResult,
	pretendErr error,
	t *i18n.Translator,
) {
	target := updateTargetLabel(atoms)

	p := plan.Plan{
		TitleKey: "plan_title",
		Action:   "Update " + target,
		Will: []plan.Item{
			{
				Key: "will_run_emerge_update_pretend",
				Data: map[string]any{
					"Target": target,
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
				Key: "will_not_apply_update_use_autounmask",
			},
			{
				Key: jokes.RandomKey(jokes.Context{
					Atom:    firstUpdateJokeAtom(atoms),
					Command: "update",
				}),
			},
		},
	}

	fmt.Println("Portico Update")
	fmt.Println()
	fmt.Println("Target:")
	fmt.Printf("  %s\n", target)
	fmt.Println()

	renderInstallMaskActions(maskActions)

	if transaction != nil {
		renderMergeTransaction(transaction)
	}

	if transaction == nil && pretendResult != nil && pretendResult.Raw != "" {
		fmt.Println("emerge --pretend output:")
		fmt.Println()
		fmt.Print(pretendResult.Raw)

		if !strings.HasSuffix(pretendResult.Raw, "\n") {
			fmt.Println()
		}
	}

	if transaction == nil && pretendResult != nil && pretendResult.Raw == "" {
		fmt.Println("emerge --pretend output:")
		fmt.Println()
		fmt.Println("  No output returned.")
	}

	if pretendErr != nil {
		fmt.Println()
		fmt.Printf("Pretend result: %v\n", pretendErr)
	}

	fmt.Println()
	fmt.Print(ui.RenderPlanWithoutConfirmation(p, t))
}

func runPackageUpdate(atoms []string, totalPackages int) error {
	installer := portage.NewEmergeInstaller()

	label := "Updating " + updateTargetLabel(atoms)

	return ui.RunInstallProgress(label, totalPackages, func(ctx context.Context, events chan<- ui.InstallProgressEvent) error {
		return installer.UpdateAtomsContext(ctx, atoms, totalPackages, func(progress portage.InstallProgress) {
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

func updateTargetLabel(atoms []string) string {
	cleanAtoms := cleanInstallArgs(atoms)
	if len(cleanAtoms) == 0 {
		return "@world"
	}

	return strings.Join(cleanAtoms, " ")
}

func firstUpdateJokeAtom(atoms []string) string {
	cleanAtoms := cleanInstallArgs(atoms)
	if len(cleanAtoms) == 0 {
		return "@world"
	}

	return cleanAtoms[0]
}
