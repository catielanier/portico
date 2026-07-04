package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rebuildCmd = &cobra.Command{
	Use:   "rebuild <atom>",
	Short: "Reconfigure USE flags and rebuild a package",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		atom, err := requireAtom(args)
		if err != nil {
			return err
		}

		fmt.Printf("Portico rebuild: %s\n", atom)
		fmt.Println("Rebuild flow is not implemented yet.")

		return nil
	},
}
