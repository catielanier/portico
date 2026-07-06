package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "portico",
	Short: "A clearer CLI/TUI entrance to Gentoo Portage",
	Long: `Portico helps inspect Gentoo packages, choose USE flags,
write per-package package.use entries, preview emerge operations,
and safely hand execution back to Portage.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(rebuildCmd)
	rootCmd.AddCommand(findCmd)
	rootCmd.AddCommand(updateCmd)
}

func requireAtom(args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("expected exactly one package atom")
	}

	return args[0], nil
}
