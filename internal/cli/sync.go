package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/catielanier/portico/internal/portage"
	"github.com/catielanier/portico/internal/ui"
)

const repositoryStaleAfter = 24 * time.Hour

func syncRepositoriesForMutation() error {
	repositories, err := portage.EnabledRepositories()
	if err != nil {
		return err
	}

	repositoriesToSync := dedupeRepositories(portage.RepositoriesNeedingSync(repositories, repositoryStaleAfter))
	if len(repositoriesToSync) == 0 {
		return nil
	}

	syncer := portage.NewRepositorySyncer()

	if err := prepareRepositorySyncer(syncer); err != nil {
		return err
	}

	for _, repository := range repositoriesToSync {
		repository := repository

		if err := ui.RunStepContext("Syncing "+repository.Name, func(ctx context.Context) error {
			return syncer.SyncContext(ctx, repository.Name)
		}); err != nil {
			return err
		}
	}

	return nil
}

func prepareRepositorySyncer(syncer *portage.RepositorySyncer) error {
	if !syncer.NeedsPrivilegeEscalation() {
		return nil
	}

	fmt.Println("Portico needs sudo privileges to sync repositories.")
	fmt.Println("You may be prompted for your password.")
	fmt.Println()

	return syncer.Authenticate()
}

func dedupeRepositories(repositories []portage.RepositoryStatus) []portage.RepositoryStatus {
	seen := make(map[string]bool)
	out := make([]portage.RepositoryStatus, 0, len(repositories))

	for _, repository := range repositories {
		if repository.Name == "" {
			continue
		}

		if seen[repository.Name] {
			continue
		}

		seen[repository.Name] = true
		out = append(out, repository)
	}

	return out
}
