package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query <atom>",
	Short: "Inspect a package and its USE flags",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		atom, err := requireAtom(args)
		if err != nil {
			return err
		}

		fmt.Printf("Portico query: %s\n", atom)
		fmt.Println("Package metadata lookup is not implemented yet.")

		return nil
	},
}
