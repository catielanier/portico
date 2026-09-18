package cli

import (
	"fmt"

	"github.com/catielanier/portico/internal/portage"
	"github.com/catielanier/portico/internal/ui"
	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query <atom[@version]>",
	Short: "Show package USE flags",
	Args:  validateSinglePackageTargetArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := maybeSyncNeverSyncedRepositories(); err != nil {
			return err
		}

		atom, err := packageAtomFromArg(args[0])
		if err != nil {
			return err
		}

		querier := portage.NewEqueryQuerier()

		result, err := querier.Query(atom)
		if err != nil {
			return err
		}

		renderQueryResult(result)

		return nil
	},
}

func renderQueryResult(result *portage.PackageQuery) {
	fmt.Println(ui.Selected("Portico Query"))
	fmt.Println()
	fmt.Println(ui.Accent("Package:"))
	fmt.Printf("  %s\n", result.Atom)
	fmt.Println()

	if len(result.Uses) == 0 {
		fmt.Println("No USE flags found.")

		if result.RawUses != "" {
			fmt.Println()
			fmt.Println("Raw output:")
			fmt.Println(result.RawUses)
		}

		return
	}

	renderUseFlagSummary(result)

	fmt.Println()
	fmt.Println(ui.Accent("Next:"))
	fmt.Printf("  sudo portico install %s\n", result.Atom)
	fmt.Printf("  sudo portico rebuild %s\n", result.Atom)
}

func renderUseFlagSummary(result *portage.PackageQuery) {
	if result == nil {
		return
	}

	if len(result.Uses) == 0 {
		fmt.Println(ui.Accent("USE flags:"))
		fmt.Println()
		fmt.Println("  No USE flags found.")
		return
	}

	showInstalledColumn := false
	for _, flag := range result.Uses {
		if flag.Installed != nil {
			showInstalledColumn = true
			break
		}
	}

	fmt.Println(ui.Accent("USE flags for:"))
	fmt.Printf("  %s\n", result.Atom)
	fmt.Println()

	fmt.Println(ui.Accent("Legend:"))
	fmt.Println("  U = flag setting for next build")
	if showInstalledColumn {
		fmt.Println("  I = flag setting on installed package")
	}
	fmt.Println()

	if showInstalledColumn {
		fmt.Println("  U I  Flag")
	} else {
		fmt.Println("  U  Flag")
	}

	for _, flag := range result.Uses {
		useState := ui.Error("-")
		if flag.EnabledForBuild {
			useState = ui.Success("+")
		}

		if showInstalledColumn {
			installedState := ui.Muted("?")
			if flag.Installed != nil {
				if *flag.Installed {
					installedState = ui.Muted("+")
				} else {
					installedState = ui.Muted("-")
				}
			}

			fmt.Printf("  %s %s  %-34s\n", useState, installedState, flag.Name)
		} else {
			fmt.Printf("  %s  %-34s\n", useState, flag.Name)
		}

		if flag.Description != "" {
			fmt.Printf("       %s\n", flag.Description)
		}
	}
}
