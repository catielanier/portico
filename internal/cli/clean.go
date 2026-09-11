package cli

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/catielanier/portico/internal/i18n"
	"github.com/catielanier/portico/internal/portage"
	"github.com/spf13/cobra"
)

var cleanCmd = newCleanCommand()

func newCleanCommand() *cobra.Command {
	translator := i18n.MustDefault()

	return &cobra.Command{
		Use:     "clean",
		Short:   translator.T("clean_short", nil),
		Long:    translator.T("clean_long", nil),
		Example: translator.T("clean_example", nil),
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireRoot(translator.T("clean_privilege_action", nil)); err != nil {
				return err
			}

			return runClean(cmd, translator)
		},
	}
}

func runClean(cmd *cobra.Command, translator *i18n.Translator) error {
	cleaner := portage.NewEmergeCleaner()

	fmt.Fprintln(cmd.OutOrStdout(), translator.T("clean_running", nil))

	if err := cleaner.Depclean(); err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), translator.T("clean_success", nil))

	return nil
}

func confirmYesNo(cmd *cobra.Command, translator *i18n.Translator, prompt string) (bool, error) {
	fmt.Fprint(cmd.OutOrStdout(), prompt)
	fmt.Fprint(cmd.OutOrStdout(), " ")

	reader := bufio.NewReader(cmd.InOrStdin())
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	answer = strings.TrimSpace(strings.ToLower(answer))

	switch answer {
	case "y", "yes":
		return true, nil
	case "n", "no", "":
		return false, nil
	default:
		fmt.Fprintln(cmd.OutOrStdout(), translator.T("confirm_unrecognized_no", nil))
		return false, nil
	}
}