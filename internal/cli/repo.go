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
		Use:     commandName,
		Short:   short,
		Long:    repoCommandLong(commandName, translator),
		Example: repoCommandExample(commandName, translator),
	}

	cmd.AddCommand(newRepoListCommand(commandName, manager, translator))
	cmd.AddCommand(newRepoAddCommand(commandName, manager, translator))
	cmd.AddCommand(newRepoSyncCommand(commandName, manager, translator))
	cmd.AddCommand(newRepoRemoveCommand(commandName, manager, translator))

	return cmd
}

func repoCommandLong(commandName string, translator *i18n.Translator) string {
	return translator.T("repo_long", map[string]any{
		"Command": commandName,
	})
}

func repoCommandExample(commandName string, translator *i18n.Translator) string {
	return translator.T("repo_example", map[string]any{
		"Command": commandName,
	})
}

func newRepoListCommand(commandName string, manager *repo.Manager, translator *i18n.Translator) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   translator.T("repo_list_short", nil),
		Long:    translator.T("repo_list_long", map[string]any{"Command": commandName}),
		Example: translator.T("repo_list_example", map[string]any{"Command": commandName}),
		Args:    cobra.NoArgs,
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
		Use:     "add <name>",
		Short:   translator.T("repo_add_short", nil),
		Long:    translator.T("repo_add_long", map[string]any{"Command": commandName}),
		Example: translator.T("repo_add_example", map[string]any{"Command": commandName}),
		Args:    cobra.ExactArgs(1),
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
	var preflight bool

	syncCmd := &cobra.Command{
		Use:     "sync [name]",
		Short:   translator.T("repo_sync_short", nil),
		Long:    translator.T("repo_sync_long", map[string]any{"Command": commandName}),
		Example: translator.T("repo_sync_example", map[string]any{"Command": commandName}),
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				if preflight {
					return runRepoSyncAllPreflight(cmd, manager, translator)
				}

				return runRepoSyncAll(cmd, manager, translator)
			}

			name := strings.TrimSpace(args[0])
			if preflight {
				return runRepoSyncOnePreflight(cmd, manager, translator, name)
			}

			return runRepoSyncOne(cmd, manager, translator, name)
		},
	}

	syncCmd.Flags().BoolVarP(
		&preflight,
		"preflight",
		"p",
		false,
		translator.T("repo_sync_preflight_help", nil),
	)

	return syncCmd
}

func runRepoSyncOne(cmd *cobra.Command, manager *repo.Manager, translator *i18n.Translator, name string) error {
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
}

func runRepoSyncAll(cmd *cobra.Command, manager *repo.Manager, translator *i18n.Translator) error {
	if err := requireRoot(translator.T("repo_sync_all_privilege_action", nil)); err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), translator.T("repo_sync_all_start", nil))

	decisions, err := manager.SyncEnabled()
	if err != nil {
		return err
	}

	if len(decisions) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), translator.T("repo_sync_all_empty", nil))
		return nil
	}

	for _, decision := range decisions {
		fmt.Fprintln(cmd.OutOrStdout(), translator.T("repo_sync_all_synced", map[string]any{
			"Repository": decision.Repository,
		}))
	}

	fmt.Fprintln(cmd.OutOrStdout(), translator.T("repo_sync_all_success", map[string]any{
		"Synced": len(decisions),
	}))

	return nil
}

func runRepoSyncOnePreflight(cmd *cobra.Command, manager *repo.Manager, translator *i18n.Translator, name string) error {
	if err := requireRoot(translator.T("repo_sync_preflight_privilege_action", map[string]any{
		"Repository": name,
	})); err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), translator.T("repo_sync_preflight_start", map[string]any{
		"Repository": name,
	}))

	decision, err := manager.SyncIfNeeded(name, 0)
	if err != nil {
		return err
	}

	renderSyncDecision(cmd, translator, decision)

	return nil
}

func runRepoSyncAllPreflight(cmd *cobra.Command, manager *repo.Manager, translator *i18n.Translator) error {
	if err := requireRoot(translator.T("repo_sync_all_preflight_privilege_action", nil)); err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), translator.T("repo_sync_all_preflight_start", nil))

	decisions, err := manager.SyncEnabledIfNeeded(0)
	if err != nil {
		return err
	}

	if len(decisions) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), translator.T("repo_sync_all_empty", nil))
		return nil
	}

	syncedCount := 0
	skippedCount := 0

	for _, decision := range decisions {
		if decision.ShouldSync {
			syncedCount++
		} else {
			skippedCount++
		}

		renderSyncDecision(cmd, translator, decision)
	}

	fmt.Fprintln(cmd.OutOrStdout(), translator.T("repo_sync_all_preflight_success", map[string]any{
		"Synced":  syncedCount,
		"Skipped": skippedCount,
	}))

	return nil
}

func renderSyncDecision(cmd *cobra.Command, translator *i18n.Translator, decision repo.SyncDecision) {
	if decision.ShouldSync {
		fmt.Fprintln(cmd.OutOrStdout(), translator.T("repo_sync_preflight_synced", map[string]any{
			"Repository": decision.Repository,
			"Reason":     string(decision.Reason),
		}))
		return
	}

	fmt.Fprintln(cmd.OutOrStdout(), translator.T("repo_sync_preflight_skipped", map[string]any{
		"Repository": decision.Repository,
	}))
}

func newRepoRemoveCommand(commandName string, manager *repo.Manager, translator *i18n.Translator) *cobra.Command {
	var force bool

	removeCmd := &cobra.Command{
		Use:     "remove <name>",
		Short:   translator.T("repo_remove_short", nil),
		Long:    translator.T("repo_remove_long", map[string]any{"Command": commandName}),
		Example: translator.T("repo_remove_example", map[string]any{"Command": commandName}),
		Args:    cobra.ExactArgs(1),
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

			result, err := manager.Remove(name, force)
			if err != nil {
				var protectedErr *repo.ProtectedRepositoryError
				if errors.As(err, &protectedErr) {
					renderProtectedRepositoryError(cmd, translator, protectedErr)
					return err
				}

				var inUseErr *repo.RepositoryInUseError
				if errors.As(err, &inUseErr) {
					renderRepositoryInUseError(cmd, translator, inUseErr, commandName)
					return err
				}

				return err
			}

			if result != nil && result.Forced {
				renderForcedRepositoryRemoveWarning(cmd, translator, result)
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

func renderProtectedRepositoryError(
	cmd *cobra.Command,
	translator *i18n.Translator,
	err *repo.ProtectedRepositoryError,
) {
	fmt.Fprintln(cmd.ErrOrStderr(), translator.T("repo_remove_protected", map[string]any{
		"Repository": err.Repository,
	}))
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

	renderInstalledPackageSample(cmd, translator, err.Packages)

	fmt.Fprintln(cmd.ErrOrStderr())
	fmt.Fprintln(cmd.ErrOrStderr(), translator.T("repo_remove_reinstall_hint", nil))
	fmt.Fprintln(cmd.ErrOrStderr(), translator.T("repo_remove_force_hint", map[string]any{
		"Command": commandName,
	}))
}

func renderForcedRepositoryRemoveWarning(
	cmd *cobra.Command,
	translator *i18n.Translator,
	result *repo.RemoveResult,
) {
	fmt.Fprintln(cmd.ErrOrStderr(), translator.T("repo_remove_forced_warning", map[string]any{
		"Repository": result.Repository,
		"Count":      len(result.InstalledPackages),
	}))

	renderInstalledPackageSample(cmd, translator, result.InstalledPackages)

	fmt.Fprintln(cmd.ErrOrStderr())
}

func renderInstalledPackageSample(
	cmd *cobra.Command,
	translator *i18n.Translator,
	packages []repo.InstalledPackage,
) {
	limit := 12

	for index, installedPackage := range packages {
		if index >= limit {
			break
		}

		fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", installedPackage.Atom)
	}

	if len(packages) > limit {
		fmt.Fprintln(cmd.ErrOrStderr(), translator.T("repo_remove_blocked_more", map[string]any{
			"Count": len(packages) - limit,
		}))
	}
}