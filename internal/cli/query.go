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
		fmt.Println()
		fmt.Println("Raw output:")
		fmt.Println(result.RawUses)
		return
	}

	fmt.Println("USE flags:")
	fmt.Println()

	for _, flag := range result.Uses {
		state := "-"
		if flag.EnabledForBuild {
			state = "+"
		}

		installed := ""
		if flag.Installed != nil {
			if *flag.Installed {
				installed = " installed:+"
			} else {
				installed = " installed:-"
			}
		}

		fmt.Printf("  %s %-18s%s\n", state, flag.Name, installed)

		if flag.Description != "" {
			fmt.Printf("    %s\n", flag.Description)
		}

		fmt.Println()
	}

	fmt.Println("Next:")
	fmt.Printf("  sudo portico install %s\n", result.Atom)
	fmt.Printf("  sudo portico rebuild %s\n", result.Atom)
}
