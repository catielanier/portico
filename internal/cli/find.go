package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/catielanier/portico/internal/portage"
	"github.com/catielanier/portico/internal/ui"
	"github.com/spf13/cobra"
)

var findCmd = &cobra.Command{
	Use:   "find <query>",
	Short: "Search for packages by atom name or description",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")

		if err := maybeSyncNeverSyncedRepositories(); err != nil {
			return err
		}

		searcher := portage.NewEmergeSearcher()

		results, err := searcher.Find(query)
		if err != nil {
			return err
		}

		renderFindResults(query, results)

		return nil
	},
}

func maybeSyncNeverSyncedRepositories() error {
	repositories, err := portage.EnabledRepositories()
	if err != nil {
		return err
	}

	neverSyncedRepositories := dedupeRepositories(portage.NeverSyncedRepositories(repositories))
	if len(neverSyncedRepositories) == 0 {
		return nil
	}

	fmt.Println("Portico noticed some enabled repositories have not been synced yet:")
	fmt.Println()

	for _, repository := range neverSyncedRepositories {
		fmt.Printf("  %s\n", repository.Name)
	}

	fmt.Println()
	fmt.Println("Packages from these repositories will not appear in search results until they are synced.")
	fmt.Println()

	confirmed, err := confirmDefaultNo("Sync these repositories now?")
	if err != nil {
		return err
	}

	if !confirmed {
		return nil
	}

	syncer := portage.NewRepositorySyncer()

	if err := prepareRepositorySyncer(syncer); err != nil {
		return err
	}

	for _, repository := range neverSyncedRepositories {
		repository := repository

		if err := ui.RunStepContext("Syncing "+repository.Name, func(ctx context.Context) error {
			return syncer.SyncContext(ctx, repository.Name)
		}); err != nil {
			return err
		}
	}

	return nil
}

func renderFindResults(query string, results []portage.SearchResult) {
	fmt.Println("Portico Find")
	fmt.Println()
	fmt.Println("Search:")
	fmt.Printf("  %s\n", query)
	fmt.Println()

	if len(results) == 0 {
		fmt.Println("No packages found.")
		return
	}

	fmt.Println("Matches:")
	fmt.Println()

	for _, result := range results {
		fmt.Printf("  %s\n", result.Atom)

		if result.Description != "" {
			fmt.Printf("    %s\n", result.Description)
		}

		if len(result.Sources) > 0 {
			fmt.Println("    Available from:")

			for _, source := range result.Sources {
				status := strings.TrimSpace(source.Status)
				version := strings.TrimSpace(source.Version)

				if status == "" {
					status = "available"
				}

				if version == "" {
					fmt.Printf("      %-10s %s\n", source.Repository, status)
				} else {
					fmt.Printf("      %-10s %-12s %s\n", source.Repository, version, status)
				}
			}
		}

		fmt.Println()
	}

	fmt.Println("Next:")
	if len(results) == 1 {
		fmt.Printf("  portico query %s\n", results[0].Atom)
		fmt.Printf("  sudo portico install %s\n", results[0].Atom)
		return
	}

	fmt.Println("  portico query <atom>")
	fmt.Println("  sudo portico install <atom>")
}
