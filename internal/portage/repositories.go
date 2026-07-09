package portage

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	DefaultRepoRoot         = "/var/db/repos"
	DefaultPorticoCacheRoot = "/var/cache/portico"
)

type RepositoryStatus struct {
	Name        string
	Enabled     bool
	AutoSync    bool
	Location    string
	LastSync    *time.Time
	NeverSynced bool
}

func EnabledRepositories() ([]RepositoryStatus, error) {
	repositories := map[string]RepositoryStatus{}

	if err := addRepositoriesFromReposConf(repositories); err != nil {
		return nil, err
	}

	if err := addRepositoriesFromRepoRoot(repositories, DefaultRepoRoot); err != nil {
		return nil, err
	}

	out := make([]RepositoryStatus, 0, len(repositories))

	for _, repository := range repositories {
		if repository.Name == "" {
			continue
		}

		if repository.Location == "" {
			repository.Location = filepath.Join(DefaultRepoRoot, repository.Name)
		}

		repository.Enabled = true

		if !repository.AutoSync {
			// AutoSync defaults to true. A false value only comes from repos.conf.
			out = append(out, repository)
			continue
		}

		repository.NeverSynced = repositoryLooksNeverSynced(repository)
		repository.LastSync = repositoryLastSync(repository)

		out = append(out, repository)
	}

	sort.Slice(out, func(i int, j int) bool {
		return out[i].Name < out[j].Name
	})

	return out, nil
}

func NeverSyncedRepositories(repositories []RepositoryStatus) []RepositoryStatus {
	var out []RepositoryStatus

	for _, repository := range repositories {
		if !repository.Enabled || !repository.AutoSync {
			continue
		}

		if repository.NeverSynced {
			out = append(out, repository)
		}
	}

	return out
}

func RepositoriesNeedingSync(repositories []RepositoryStatus, staleAfter time.Duration) []RepositoryStatus {
	var out []RepositoryStatus

	for _, repository := range repositories {
		if !repository.Enabled || !repository.AutoSync {
			continue
		}

		if repository.NeverSynced {
			out = append(out, repository)
			continue
		}

		if repository.LastSync == nil {
			out = append(out, repository)
			continue
		}

		if time.Since(*repository.LastSync) >= staleAfter {
			out = append(out, repository)
		}
	}

	return out
}

func RecordRepositorySync(repositoryName string) error {
	repositoryName = strings.TrimSpace(repositoryName)
	if repositoryName == "" {
		return nil
	}

	stampDir := filepath.Join(DefaultPorticoCacheRoot, "repo-sync")

	if err := os.MkdirAll(stampDir, 0o755); err != nil {
		return err
	}

	stampPath := filepath.Join(stampDir, safeRepositoryStampName(repositoryName))

	now := time.Now()
	return os.WriteFile(stampPath, []byte(now.Format(time.RFC3339)+"\n"), 0o644)
}

func addRepositoriesFromReposConf(repositories map[string]RepositoryStatus) error {
	paths, err := reposConfPaths()
	if err != nil {
		return err
	}

	for _, path := range paths {
		if err := parseReposConf(path, repositories); err != nil {
			return err
		}
	}

	return nil
}

func reposConfPaths() ([]string, error) {
	var paths []string

	info, err := os.Stat("/etc/portage/repos.conf")
	if err == nil && !info.IsDir() {
		paths = append(paths, "/etc/portage/repos.conf")
	}

	matches, err := filepath.Glob("/etc/portage/repos.conf/*.conf")
	if err != nil {
		return nil, err
	}

	paths = append(paths, matches...)
	sort.Strings(paths)

	return paths, nil
}

func parseReposConf(path string, repositories map[string]RepositoryStatus) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	currentSection := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			continue
		}

		if currentSection == "" || strings.EqualFold(currentSection, "DEFAULT") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		repository := repositories[currentSection]
		repository.Name = currentSection
		repository.Enabled = true

		if !repository.AutoSync {
			// Default true unless explicitly disabled below.
			repository.AutoSync = true
		}

		switch key {
		case "location":
			repository.Location = value

		case "auto-sync":
			repository.AutoSync = !isFalseValue(value)
		}

		repositories[currentSection] = repository
	}

	return scanner.Err()
}

func addRepositoriesFromRepoRoot(repositories map[string]RepositoryStatus, repoRoot string) error {
	entries, err := os.ReadDir(repoRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		location := filepath.Join(repoRoot, entry.Name())
		name := repositoryNameFromLocation(location)
		if name == "" {
			continue
		}

		repository := repositories[name]
		repository.Name = name
		repository.Enabled = true

		if repository.Location == "" {
			repository.Location = location
		}

		if !repository.AutoSync {
			repository.AutoSync = true
		}

		repositories[name] = repository
	}

	return nil
}

func repositoryNameFromLocation(location string) string {
	repoNamePath := filepath.Join(location, "profiles", "repo_name")

	content, err := os.ReadFile(repoNamePath)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(content))
}

func repositoryLooksNeverSynced(repository RepositoryStatus) bool {
	if repository.Location == "" {
		return true
	}

	repoNamePath := filepath.Join(repository.Location, "profiles", "repo_name")

	_, err := os.Stat(repoNamePath)
	return err != nil
}

func repositoryLastSync(repository RepositoryStatus) *time.Time {
	var latest *time.Time

	candidates := []string{
		repositorySyncStampPath(repository.Name),
		filepath.Join(repository.Location, "metadata", "timestamp.chk"),
		filepath.Join(repository.Location, "metadata", "timestamp"),
		filepath.Join(repository.Location, "profiles", "repo_name"),
		repository.Location,
	}

	for _, path := range candidates {
		if path == "" {
			continue
		}

		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		modTime := info.ModTime()

		if latest == nil || modTime.After(*latest) {
			latest = &modTime
		}
	}

	return latest
}

func repositorySyncStampPath(repositoryName string) string {
	if repositoryName == "" {
		return ""
	}

	return filepath.Join(DefaultPorticoCacheRoot, "repo-sync", safeRepositoryStampName(repositoryName))
}

func safeRepositoryStampName(repositoryName string) string {
	repositoryName = strings.TrimSpace(repositoryName)
	repositoryName = strings.ReplaceAll(repositoryName, "/", "_")
	repositoryName = strings.ReplaceAll(repositoryName, string(os.PathSeparator), "_")

	return repositoryName
}

func isFalseValue(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))

	switch value {
	case "0", "false", "no", "off":
		return true
	default:
		return false
	}
}
