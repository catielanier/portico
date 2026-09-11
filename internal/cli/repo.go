package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/catielanier/portico/internal/i18n"
	"github.com/catielanier/portico/internal/repo"
	"github.com/spf13/cobra"
)

func newRepoCommand(commandName string, short string) *cobra.Command {
	manager := repo.NewManager()
	translator := i18n.MustDefault()

	cmd := &cobra.Command{
		Use:   commandName,
		Short: short,
	}

	cmd.AddCommand(newRepoListCommand(commandName, manager, translator))
	cmd.AddCommand(newRepoAddCommand(commandName, manager, translator))
	cmd.AddCommand(newRepoSyncCommand(commandName, manager, translator))
	cmd.AddCommand(newRepoRemoveCommand(commandName, manager, translator))

	return cmd
}

func newRepoListCommand(commandName string, manager *repo.Manager, translator *i18n.Translator) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: translator.T("repo_list_short", nil),
		RunE: func(cmd *cobra.Command, args []string) error {
			repositories, err := manager.ListEnabled()
			if err != nil {
				return err
			}

			if len(repositories) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), translator.T("repo_list_empty", nil))
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), translator.T("repo_list_heading", nil))

			for _, repository := range repositories {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", repository.Name)
			}

			return nil
		},
	}
}

func newRepoAddCommand(commandName string, manager *repo.Manager, translator *i18n.Translator) *cobra.Command {
	return &cobra.Command{
		Use:   "add <name>",
		Short: translator.T("repo_add_short", nil),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])

			if err := requireRoot(translator.T("repo_add_privilege_action", map[string]any{
				"Repository": name,
			})); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), translator.T("repo_add_enabling", map[string]any{
				"Repository": name,
			}))

			if err := manager.Add(name); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), translator.T("repo_add_success", map[string]any{
				"Repository": name,
			}))

			return nil
		},
	}
}

func newRepoSyncCommand(commandName string, manager *repo.Manager, translator *i18n.Translator) *cobra.Command {
	return &cobra.Command{
		Use:   "sync <name>",
		Short: translator.T("repo_sync_short", nil),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])

			if err := requireRoot(translator.T("repo_sync_privilege_action", map[string]any{
				"Repository": name,
			})); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), translator.T("repo_sync_start", map[string]any{
				"Repository": name,
			}))

			if err := manager.Sync(name); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), translator.T("repo_sync_success", map[string]any{
				"Repository": name,
			}))

			return nil
		},
	}
}

func newRepoRemoveCommand(commandName string, manager *repo.Manager, translator *i18n.Translator) *cobra.Command {
	var force bool

	removeCmd := &cobra.Command{
		Use:   "remove <name>",
		Short: translator.T("repo_remove_short", nil),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])

			if err := requireRoot(translator.T("repo_remove_privilege_action", map[string]any{
				"Repository": name,
			})); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), translator.T("repo_remove_checking", map[string]any{
				"Repository": name,
			}))

			if err := manager.Remove(name, force); err != nil {
				var inUseErr *repo.RepositoryInUseError
				if errors.As(err, &inUseErr) {
					renderRepositoryInUseError(cmd, translator, inUseErr, commandName)
					return err
				}

				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), translator.T("repo_remove_success", map[string]any{
				"Repository": name,
			}))

			return nil
		},
	}

	removeCmd.Flags().BoolVar(&force, "force", false, translator.T("repo_remove_force_help", nil))

	return removeCmd
}

func renderRepositoryInUseError(
	cmd *cobra.Command,
	translator *i18n.Translator,
	err *repo.RepositoryInUseError,
	commandName string,
) {
	fmt.Fprintln(cmd.ErrOrStderr(), translator.T("repo_remove_blocked", map[string]any{
		"Repository": err.Repository,
	}))

	limit := 12
	for index, installedPackage := range err.Packages {
		if index >= limit {
			break
		}

		fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", installedPackage.Atom)
	}

	if len(err.Packages) > limit {
		fmt.Fprintln(cmd.ErrOrStderr(), translator.T("repo_remove_blocked_more", map[string]any{
			"Count": len(err.Packages) - limit,
		}))
	}

	fmt.Fprintln(cmd.ErrOrStderr())
	fmt.Fprintln(cmd.ErrOrStderr(), translator.T("repo_remove_force_hint", map[string]any{
		"Command": commandName,
	}))
}