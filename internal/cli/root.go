package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/catielanier/portico/internal/i18n"
	"github.com/spf13/cobra"
)

var rootCmd = newRootCommand()

func newRootCommand() *cobra.Command {
	translator := i18n.MustDefault()

	cmd := &cobra.Command{
		Use:          "portico",
		Short:        translator.T("root_short", nil),
		Long:         translator.T("root_long", nil),
		Example:      translator.T("root_example", nil),
		SilenceUsage: true,
	}

	cmd.CompletionOptions.DisableDefaultCmd = true

	return cmd
}

func Execute() error {
	args := os.Args[1:]

	if shouldPrintVersion(args) {
		fmt.Println(Version())
		return nil
	}

	if shouldPrintRoutedHelp(args) {
		return printRoutedHelp(args)
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

	applyCommandHelp(rootCmd, i18n.MustDefault())
}

func shouldPrintVersion(args []string) bool {
	if len(args) != 1 {
		return false
	}

	switch args[0] {
	case "--version", "-v":
		return true
	default:
		return false
	}
}

func shouldPrintRoutedHelp(args []string) bool {
	if len(args) == 0 {
		return false
	}

	switch args[0] {
	case "--help", "-h":
		return true
	default:
		return false
	}
}

func printRoutedHelp(args []string) error {
	commandPath := args[1:]

	if len(commandPath) == 0 {
		return rootCmd.Help()
	}

	command, err := findCommandByPath(rootCmd, commandPath)
	if err != nil {
		return err
	}

	return command.Help()
}

func findCommandByPath(root *cobra.Command, commandPath []string) (*cobra.Command, error) {
	current := root

	for _, part := range commandPath {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		next := findDirectSubcommand(current, part)
		if next == nil {
			return nil, fmt.Errorf("unknown help topic: %s", strings.Join(commandPath, " "))
		}

		current = next
	}

	return current, nil
}

func findDirectSubcommand(parent *cobra.Command, name string) *cobra.Command {
	for _, command := range parent.Commands() {
		if command.Name() == name {
			return command
		}

		for _, alias := range command.Aliases {
			if alias == name {
				return command
			}
		}
	}

	return nil
}