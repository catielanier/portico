package portage

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

type RepositoryStatus struct {
	Name        string
	Enabled     bool
	Location    string
	LastSync    *time.Time
	NeverSynced bool
}

func EnabledRepositories() ([]RepositoryStatus, error) {
	const reposConfDir = "/etc/portage/repos.conf"

	entries, err := os.ReadDir(reposConfDir)
	if err != nil {
		// If repos.conf does not exist, do not block commands.
		return nil, nil
	}

	var repositories []RepositoryStatus

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".conf") {
			continue
		}

		path := filepath.Join(reposConfDir, entry.Name())

		found, err := parseReposConf(path)
		if err != nil {
			continue
		}

		repositories = append(repositories, found...)
	}

	return repositories, nil
}

func NeverSyncedRepositories(repositories []RepositoryStatus) []RepositoryStatus {
	var out []RepositoryStatus

	for _, repository := range repositories {
		if repository.Enabled && repository.NeverSynced {
			out = append(out, repository)
		}
	}

	return out
}

func RepositoriesNeedingSync(repositories []RepositoryStatus, staleAfter time.Duration) []RepositoryStatus {
	var out []RepositoryStatus

	for _, repository := range repositories {
		if !repository.Enabled {
			continue
		}

		if repository.NeverSynced {
			out = append(out, repository)
			continue
		}

		if repository.LastSync == nil {
			continue
		}

		if time.Since(*repository.LastSync) >= staleAfter {
			out = append(out, repository)
		}
	}

	return out
}

func parseReposConf(path string) ([]RepositoryStatus, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")

	var repositories []RepositoryStatus
	var current *RepositoryStatus

	flush := func() {
		if current == nil {
			return
		}

		if current.Name != "" {
			current.Enabled = true
			current.LastSync = repositoryLastSync(*current)
			current.NeverSynced = repositoryLooksNeverSynced(*current)
			repositories = append(repositories, *current)
		}

		current = nil
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			flush()

			name := strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			current = &RepositoryStatus{Name: name}
			continue
		}

		if current == nil {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch key {
		case "location":
			current.Location = value
		}
	}

	flush()

	return repositories, nil
}

func repositoryLooksNeverSynced(repository RepositoryStatus) bool {
	if repository.Location == "" {
		return false
	}

	infoPath := filepath.Join(repository.Location, "profiles", "repo_name")

	if _, err := os.Stat(infoPath); err == nil {
		return false
	}

	return true
}

func repositoryLastSync(repository RepositoryStatus) *time.Time {
	if repository.Location == "" {
		return nil
	}

	// Gentoo repositories commonly have metadata/timestamp after sync.
	candidates := []string{
		filepath.Join(repository.Location, "metadata", "timestamp"),
		filepath.Join(repository.Location, "profiles", "repo_name"),
		filepath.Join(repository.Location),
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}

		modTime := info.ModTime()
		return &modTime
	}

	return nil
}
