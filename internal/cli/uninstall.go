package cli

import (
	"fmt"
	"strings"

	"github.com/catielanier/portico/internal/i18n"
	"github.com/catielanier/portico/internal/portage"
	"github.com/spf13/cobra"
)

var uninstallCmd = newUninstallCommand()

func newUninstallCommand() *cobra.Command {
	translator := i18n.MustDefault()

	return &cobra.Command{
		Use:     "uninstall <atom...>",
		Aliases: []string{"remove-package"},
		Short:   translator.T("uninstall_short", nil),
		Long:    translator.T("uninstall_long", nil),
		Example: translator.T("uninstall_example", nil),
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireRoot(translator.T("uninstall_privilege_action", nil)); err != nil {
				return err
			}

			return runUninstall(cmd, translator, args)
		},
	}
}

func runUninstall(cmd *cobra.Command, translator *i18n.Translator, atoms []string) error {
	cleaner := portage.NewEmergeCleaner()

	fmt.Fprintln(cmd.OutOrStdout(), translator.T("uninstall_running", map[string]any{
		"Atoms": strings.Join(atoms, " "),
	}))

	if err := cleaner.Unmerge(atoms); err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), translator.T("uninstall_success", map[string]any{
		"Atoms": strings.Join(atoms, " "),
	}))

	runDepclean, err := confirmYesNo(
		cmd,
		translator,
		translator.T("uninstall_depclean_prompt", nil),
	)
	if err != nil {
		return err
	}

	if !runDepclean {
		fmt.Fprintln(cmd.OutOrStdout(), translator.T("uninstall_depclean_skipped", nil))
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout())

	return runClean(cmd, translator)
}