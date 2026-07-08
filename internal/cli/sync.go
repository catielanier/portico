package cli

import (
	"fmt"
	"time"

	"github.com/catielanier/portico/internal/portage"
)

const repositoryStaleAfter = 24 * time.Hour

func syncRepositoriesForMutation() error {
	repositories, err := portage.EnabledRepositories()
	if err != nil {
		return err
	}

	repositoriesToSync := portage.RepositoriesNeedingSync(repositories, repositoryStaleAfter)
	if len(repositoriesToSync) == 0 {
		return nil
	}

	fmt.Println("Portico repository sync check:")
	fmt.Println()

	for _, repository := range repositoriesToSync {
		if repository.NeverSynced {
			fmt.Printf("  %s needs sync: repository has not been synced yet\n", repository.Name)
			continue
		}

		if repository.LastSync != nil {
			fmt.Printf("  %s needs sync: last sync was %s ago\n", repository.Name, formatDuration(time.Since(*repository.LastSync)))
			continue
		}

		fmt.Printf("  %s needs sync\n", repository.Name)
	}

	fmt.Println()

	syncer := portage.NewRepositorySyncer()

	for _, repository := range repositoriesToSync {
		fmt.Printf("Syncing %s...\n", repository.Name)

		if err := syncer.Sync(repository.Name); err != nil {
			return err
		}
	}

	fmt.Println()
	return nil
}

func formatDuration(duration time.Duration) string {
	if duration < time.Minute {
		return "less than a minute"
	}

	if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute"
		}

		return fmt.Sprintf("%d minutes", minutes)
	}

	if duration < 48*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour"
		}

		return fmt.Sprintf("%d hours", hours)
	}

	days := int(duration.Hours() / 24)
	if days == 1 {
		return "1 day"
	}

	return fmt.Sprintf("%d days", days)
}
