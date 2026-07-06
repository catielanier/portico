package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update [atom]",
	Short: "Preview and run package or world updates",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			fmt.Println("Portico update: @world")
			fmt.Println("World update flow is not implemented yet.")
			return nil
		}

		atom := args[0]

		fmt.Printf("Portico update: %s\n", atom)
		fmt.Println("Single-package update flow is not implemented yet.")

		return nil
	},
}
