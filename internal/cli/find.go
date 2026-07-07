package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/catielanier/portico/internal/portage"
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

	neverSynced := portage.NeverSyncedRepositories(repositories)
	if len(neverSynced) == 0 {
		return nil
	}

	fmt.Println("Portico noticed enabled repositories that do not appear to have been synced yet:")
	fmt.Println()

	for _, repository := range neverSynced {
		fmt.Printf("  %s\n", repository.Name)
	}

	fmt.Println()
	fmt.Println("Packages from these repositories may not appear in search results until they are synced.")
	fmt.Print("Sync them now? [Y/n] ")

	reader := bufio.NewReader(os.Stdin)

	answer, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	answer = strings.ToLower(strings.TrimSpace(answer))

	if answer == "n" || answer == "no" {
		fmt.Println()
		fmt.Println("Continuing with currently available repository metadata.")
		fmt.Println()
		return nil
	}

	syncer := portage.NewRepositorySyncer()

	for _, repository := range neverSynced {
		fmt.Printf("Syncing %s...\n", repository.Name)

		if err := syncer.Sync(repository.Name); err != nil {
			return err
		}
	}

	fmt.Println()
	return nil
}

func renderFindResults(query string, results []portage.SearchResult) {
	fmt.Println("Portico Find")
	fmt.Println()
	fmt.Println("Search:")
	fmt.Printf("  %s\n", query)
	fmt.Println()

	if len(results) == 0 {
		fmt.Println("No matching packages found.")
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
				repository := source.Repository
				if repository == "" {
					repository = "unknown"
				}

				version := source.Version
				if version == "" {
					version = "unknown"
				}

				status := source.Status
				if status == "" {
					status = "available"
				}

				fmt.Printf("      %-12s %-10s %s\n", repository, version, status)
			}
		}

		if result.Installed {
			fmt.Println("    installed")
		}

		fmt.Println()
	}

	fmt.Println("Next:")
	fmt.Printf("  portico query %s\n", results[0].Atom)
	fmt.Printf("  sudo portico install %s\n", results[0].Atom)
}
