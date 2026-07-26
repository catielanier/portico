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
	Use:   "update [atom...]",
	Short: "Update the world set or one or more packages",
	Args:  validateZeroOrMoreAtomArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireRoot("update packages"); err != nil {
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

		var pretendResult *portage.PretendResult
		var pretendErr error

		if err := ui.RunStep("Running emerge --pretend --update --deep --newuse", func() error {
			pretendResult, pretendErr = portage.EmergePretendUpdateWithConfigRootForAtoms(atoms, sandbox.Root)

			if pretendErr != nil && pretendResult == nil {
				return pretendErr
			}

			return nil
		}); err != nil {
			return err
		}

		t, err := i18n.New("en")
		if err != nil {
			return err
		}

		transaction := (*portage.MergeTransaction)(nil)
		if pretendResult != nil {
			transaction = portage.ParseMergeTransaction(pretendResult.Raw)
		}

		renderUpdatePlan(
			atoms,
			transaction,
			pretendResult,
			pretendErr,
			t,
		)

		if pretendErr != nil {
			fmt.Println()
			fmt.Println("Portico does not apply update-time Portage config changes yet.")
			fmt.Println("Resolve the issue shown above, then run update again.")
			return pretendErr
		}

		confirmed, err := confirmDefaultNo("Run this update?")
		if err != nil {
			return err
		}

		if !confirmed {
			fmt.Println("Update cancelled.")
			return nil
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

func renderUpdatePlan(
	atoms []string,
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
				Key: "will_not_apply_update_autounmask",
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
