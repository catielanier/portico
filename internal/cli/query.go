package cli

import (
	"fmt"

	"github.com/catielanier/portico/internal/portage"
	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query <atom>",
	Short: "Inspect a package and its USE flags",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		atom := args[0]

		if err := maybeSyncNeverSyncedRepositories(); err != nil {
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
	fmt.Println("Portico Query")
	fmt.Println()
	fmt.Println("Package:")
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

	showInstalledColumn := false
	for _, flag := range result.Uses {
		if flag.Installed != nil {
			showInstalledColumn = true
			break
		}
	}

	fmt.Println("Legend:")
	fmt.Println("  U = flag setting for next build")
	if showInstalledColumn {
		fmt.Println("  I = flag setting on installed package")
	}
	fmt.Println()

	fmt.Println("USE flags:")
	fmt.Println()

	if showInstalledColumn {
		fmt.Println("  U I  Flag")
	} else {
		fmt.Println("  U  Flag")
	}

	for _, flag := range result.Uses {
		useState := "-"
		if flag.EnabledForBuild {
			useState = "+"
		}

		if showInstalledColumn {
			installedState := "?"
			if flag.Installed != nil {
				if *flag.Installed {
					installedState = "+"
				} else {
					installedState = "-"
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

	fmt.Println()
	fmt.Println("Next:")
	fmt.Printf("  sudo portico install %s\n", result.Atom)
	fmt.Printf("  sudo portico rebuild %s\n", result.Atom)
}

func renderUseFlagSummary(result *portage.PackageQuery) {
	if len(result.Uses) == 0 {
		fmt.Println("USE flags:")
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

	fmt.Println("USE flags:")
	fmt.Println()

	if showInstalledColumn {
		fmt.Println("  U I  Flag")
	} else {
		fmt.Println("  U  Flag")
	}

	for _, flag := range result.Uses {
		useState := "-"
		if flag.EnabledForBuild {
			useState = "+"
		}

		if showInstalledColumn {
			installedState := "?"
			if flag.Installed != nil {
				if *flag.Installed {
					installedState = "+"
				} else {
					installedState = "-"
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
