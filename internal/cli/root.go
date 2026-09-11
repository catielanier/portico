package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "portico",
	Short: "A clearer CLI/TUI entrance to Gentoo Portage",
	Long:  "Portico is a clearer CLI/TUI entrance to Gentoo Portage for choosing package features and safely running emerge.",
}

func Execute() error {
	if shouldPrintVersion(os.Args[1:]) {
		fmt.Println(Version())
		return nil
	}

	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(findCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(rebuildCmd)
	rootCmd.AddCommand(updateCmd)

	rootCmd.AddCommand(newRepoCommand("repo", "Manage Portage repositories"))
	rootCmd.AddCommand(newRepoCommand("overlay", "Manage Portage overlays"))
}

func shouldPrintVersion(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "--version", "-v":
			return true
		}
	}

	return false
}