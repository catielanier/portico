package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var findCmd = &cobra.Command{
	Use:   "find <query>",
	Short: "Search for packages by atom name or description",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")

		fmt.Printf("Portico find: %s\n", query)
		fmt.Println("Package search is not implemented yet.")

		return nil
	},
}
