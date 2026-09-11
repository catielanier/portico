package cli

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"github.com/catielanier/portico/internal/i18n"
	"github.com/catielanier/portico/internal/portage"
	"github.com/catielanier/portico/internal/ui"
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

			return runCleanWorkflow(cmd, translator)
		},
	}
}

func runCleanWorkflow(cmd *cobra.Command, translator *i18n.Translator) error {
	cleaner := portage.NewEmergeCleaner()

	fmt.Fprintln(cmd.OutOrStdout(), translator.T("clean_preview_heading", nil))

	preview, err := cleaner.PretendDepclean()
	if preview != "" {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), preview)
	}

	if err != nil {
		return err
	}

	confirmed, err := confirmYesNo(cmd, translator, translator.T("clean_confirm_prompt", nil))
	if err != nil {
		return err
	}

	if !confirmed {
		fmt.Fprintln(cmd.OutOrStdout(), translator.T("clean_cancelled", nil))
		return nil
	}

	total := portage.ParseCleanupTotal(preview)

	return ui.RunCleanupProgress(
		translator.T("clean_progress_label", nil),
		portage.CleanupModeDepclean,
		total,
		func(ctx context.Context, events chan<- ui.CleanupProgressEvent) error {
			return cleaner.DepcleanContext(ctx, total, func(progress portage.CleanupProgress) {
				events <- ui.CleanupProgressEvent{
					Mode:           progress.Mode,
					CurrentPackage: progress.CurrentPackage,
					CurrentIndex:   progress.CurrentIndex,
					Total:          progress.Total,
					Waiting:        progress.Waiting,
					Message:        progress.Message,
				}
			})
		},
	)
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
