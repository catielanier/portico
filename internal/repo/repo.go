package repo

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	installedPackageDatabasePath = "/var/db/pkg"
	syncStampDirectory           = "/var/cache/portico/repo-sync"
	defaultSyncStaleAfter        = 24 * time.Hour
)

var ErrRepositoryNotEnabled = errors.New("repository is not enabled")

type Repository struct {
	Name string
	Raw  string
}

type InstalledPackage struct {
	Atom       string
	Repository string
	Path       string
}

type SyncDecision struct {
	Repository string
	ShouldSync bool
	Reason     SyncReason
	LastSynced  *time.Time
}

type SyncReason string

const (
	SyncReasonManual      SyncReason = "manual"
	SyncReasonNotNeeded   SyncReason = "not-needed"
	SyncReasonNeverSynced SyncReason = "never-synced"
	SyncReasonStale       SyncReason = "stale"
)

type RemoveResult struct {
	Repository        string
	Forced            bool
	InstalledPackages []InstalledPackage
}

type ProtectedRepositoryError struct {
	Repository string
}

func (e *ProtectedRepositoryError) Error() string {
	return fmt.Sprintf("repository %s is protected and cannot be removed", e.Repository)
}

type RepositoryInUseError struct {
	Repository string
	Packages   []InstalledPackage
}

func (e *RepositoryInUseError) Error() string {
	return fmt.Sprintf("repository %s has installed packages", e.Repository)
}

type Manager struct{}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) ListEnabled() ([]Repository, error) {
	output, err := runCommand("eselect", "repository", "list", "-i")
	if err != nil {
		return nil, err
	}

	repositories := parseEselectRepositoryList(output)

	sort.Slice(repositories, func(i int, j int) bool {
		return repositories[i].Name < repositories[j].Name
	})

	return repositories, nil
}

func (m *Manager) IsEnabled(name string) (bool, error) {
	name, err := normalizeRepositoryName(name)
	if err != nil {
		return false, err
	}

	repositories, err := m.ListEnabled()
	if err != nil {
		return false, err
	}

	for _, repository := range repositories {
		if repository.Name == name {
			return true, nil
		}
	}

	return false, nil
}

func (m *Manager) Add(name string) error {
	name, err := normalizeRepositoryName(name)
	if err != nil {
		return err
	}

	enabled, err := m.IsEnabled(name)
	if err != nil {
		return err
	}

	if !enabled {
		if _, err := runCommand("eselect", "repository", "enable", name); err != nil {
			return err
		}

		enabled, err = m.IsEnabled(name)
		if err != nil {
			return err
		}

		if !enabled {
			return fmt.Errorf("repository %s was enabled but does not appear in enabled repository list", name)
		}
	}

	return m.Sync(name)
}

func (m *Manager) Sync(name string) error {
	name, err := normalizeRepositoryName(name)
	if err != nil {
		return err
	}

	enabled, err := m.IsEnabled(name)
	if err != nil {
		return err
	}

	if !enabled {
		return fmt.Errorf("%w: %s", ErrRepositoryNotEnabled, name)
	}

	if _, err := runCommand("emaint", "sync", "-r", name); err != nil {
		return err
	}

	return writeSyncStamp(name, time.Now())
}

func (m *Manager) SyncEnabled() ([]SyncDecision, error) {
	repositories, err := m.ListEnabled()
	if err != nil {
		return nil, err
	}

	decisions := make([]SyncDecision, 0, len(repositories))

	for _, repository := range repositories {
		if err := m.Sync(repository.Name); err != nil {
			return decisions, err
		}

		now := time.Now()
		decisions = append(decisions, SyncDecision{
			Repository: repository.Name,
			ShouldSync: true,
			Reason:     SyncReasonManual,
			LastSynced:  &now,
		})
	}

	return decisions, nil
}

func (m *Manager) SyncIfNeeded(name string, staleAfter time.Duration) (SyncDecision, error) {
	name, err := normalizeRepositoryName(name)
	if err != nil {
		return SyncDecision{}, err
	}

	decision, err := m.SyncDecision(name, staleAfter)
	if err != nil {
		return SyncDecision{}, err
	}

	if !decision.ShouldSync {
		return decision, nil
	}

	if err := m.Sync(name); err != nil {
		return decision, err
	}

	now := time.Now()
	decision.LastSynced = &now

	return decision, nil
}

func (m *Manager) SyncDecision(name string, staleAfter time.Duration) (SyncDecision, error) {
	name, err := normalizeRepositoryName(name)
	if err != nil {
		return SyncDecision{}, err
	}

	if staleAfter <= 0 {
		staleAfter = defaultSyncStaleAfter
	}

	enabled, err := m.IsEnabled(name)
	if err != nil {
		return SyncDecision{}, err
	}

	if !enabled {
		return SyncDecision{}, fmt.Errorf("%w: %s", ErrRepositoryNotEnabled, name)
	}

	lastSynced, ok, err := readSyncStamp(name)
	if err != nil {
		return SyncDecision{}, err
	}

	if !ok {
		return SyncDecision{
			Repository: name,
			ShouldSync: true,
			Reason:     SyncReasonNeverSynced,
			LastSynced:  nil,
		}, nil
	}

	if time.Since(lastSynced) >= staleAfter {
		return SyncDecision{
			Repository: name,
			ShouldSync: true,
			Reason:     SyncReasonStale,
			LastSynced:  &lastSynced,
		}, nil
	}

	return SyncDecision{
		Repository: name,
		ShouldSync: false,
		Reason:     SyncReasonNotNeeded,
		LastSynced:  &lastSynced,
	}, nil
}

func (m *Manager) SyncEnabledIfNeeded(staleAfter time.Duration) ([]SyncDecision, error) {
	if staleAfter <= 0 {
		staleAfter = defaultSyncStaleAfter
	}

	repositories, err := m.ListEnabled()
	if err != nil {
		return nil, err
	}

	decisions := make([]SyncDecision, 0, len(repositories))

	for _, repository := range repositories {
		decision, err := m.SyncIfNeeded(repository.Name, staleAfter)
		if err != nil {
			return decisions, err
		}

		decisions = append(decisions, decision)
	}

	return decisions, nil
}

func (m *Manager) Remove(name string, force bool) (*RemoveResult, error) {
	name, err := normalizeRepositoryName(name)
	if err != nil {
		return nil, err
	}

	if isProtectedRepository(name) {
		return nil, &ProtectedRepositoryError{
			Repository: name,
		}
	}

	enabled, err := m.IsEnabled(name)
	if err != nil {
		return nil, err
	}

	if !enabled {
		return nil, fmt.Errorf("%w: %s", ErrRepositoryNotEnabled, name)
	}

	installedPackages, err := InstalledPackagesFromRepository(name)
	if err != nil {
		return nil, err
	}

	if len(installedPackages) > 0 && !force {
		return nil, &RepositoryInUseError{
			Repository: name,
			Packages:   installedPackages,
		}
	}

	if _, err := runCommand("eselect", "repository", "disable", name); err != nil {
		return nil, err
	}

	stillEnabled, err := m.IsEnabled(name)
	if err != nil {
		return nil, err
	}

	if stillEnabled {
		return nil, fmt.Errorf("repository %s was disabled but still appears in enabled repository list", name)
	}

	if err := deleteSyncStamp(name); err != nil {
		return nil, err
	}

	return &RemoveResult{
		Repository:        name,
		Forced:            force && len(installedPackages) > 0,
		InstalledPackages: installedPackages,
	}, nil
}

func InstalledPackagesFromRepository(repository string) ([]InstalledPackage, error) {
	repository, err := normalizeRepositoryName(repository)
	if err != nil {
		return nil, err
	}

	var packages []InstalledPackage

	err = filepath.WalkDir(installedPackageDatabasePath, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() {
			return nil
		}

		if entry.Name() != "repository" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		sourceRepository := strings.TrimSpace(string(data))
		if sourceRepository != repository {
			return nil
		}

		packagePath := filepath.Dir(path)
		packageAtom, err := installedPackageAtomFromPath(packagePath)
		if err != nil {
			return err
		}

		packages = append(packages, InstalledPackage{
			Atom:       packageAtom,
			Repository: sourceRepository,
			Path:       packagePath,
		})

		return nil
	})

	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	sort.Slice(packages, func(i int, j int) bool {
		return packages[i].Atom < packages[j].Atom
	})

	return packages, nil
}

func installedPackageAtomFromPath(packagePath string) (string, error) {
	relativePath, err := filepath.Rel(installedPackageDatabasePath, packagePath)
	if err != nil {
		return "", err
	}

	parts := strings.Split(relativePath, string(os.PathSeparator))
	if len(parts) != 2 {
		return "", fmt.Errorf("unexpected installed package path: %s", packagePath)
	}

	return parts[0] + "/" + parts[1], nil
}

func parseEselectRepositoryList(output string) []Repository {
	lines := strings.Split(output, "\n")

	var repositories []Repository
	seen := make(map[string]bool)

	for _, line := range lines {
		repository, ok := parseEselectRepositoryListLine(line)
		if !ok {
			continue
		}

		if seen[repository.Name] {
			continue
		}

		seen[repository.Name] = true
		repositories = append(repositories, repository)
	}

	return repositories
}

func parseEselectRepositoryListLine(line string) (Repository, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return Repository{}, false
	}

	if !strings.Contains(trimmed, "]") {
		return Repository{}, false
	}

	afterBracket := trimmed[strings.Index(trimmed, "]")+1:]
	afterBracket = strings.TrimSpace(afterBracket)
	if afterBracket == "" {
		return Repository{}, false
	}

	fields := strings.Fields(afterBracket)
	if len(fields) == 0 {
		return Repository{}, false
	}

	name := strings.TrimSpace(fields[0])
	name = strings.TrimSuffix(name, "*")
	name = strings.TrimSpace(name)

	if name == "" {
		return Repository{}, false
	}

	if strings.HasPrefix(name, "[") {
		return Repository{}, false
	}

	return Repository{
		Name: name,
		Raw:  trimmed,
	}, true
}

func normalizeRepositoryName(name string) (string, error) {
	name = strings.TrimSpace(name)

	if name == "" {
		return "", fmt.Errorf("repository name is required")
	}

	if strings.ContainsRune(name, os.PathSeparator) {
		return "", fmt.Errorf("repository name must not contain path separators: %s", name)
	}

	if strings.Contains(name, "\x00") {
		return "", fmt.Errorf("repository name must not contain null bytes")
	}

	if name == "." || name == ".." {
		return "", fmt.Errorf("invalid repository name: %s", name)
	}

	return name, nil
}

func isProtectedRepository(name string) bool {
	switch name {
	case "gentoo":
		return true
	default:
		return false
	}
}

func syncStampPath(repository string) (string, error) {
	repository, err := normalizeRepositoryName(repository)
	if err != nil {
		return "", err
	}

	return filepath.Join(syncStampDirectory, repository), nil
}

func writeSyncStamp(repository string, syncedAt time.Time) error {
	path, err := syncStampPath(repository)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	value := syncedAt.UTC().Format(time.RFC3339) + "\n"
	tempPath := path + ".tmp"

	if err := os.WriteFile(tempPath, []byte(value), 0o644); err != nil {
		return err
	}

	return os.Rename(tempPath, path)
}

func readSyncStamp(repository string) (time.Time, bool, error) {
	path, err := syncStampPath(repository)
	if err != nil {
		return time.Time{}, false, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return time.Time{}, false, nil
	}

	if err != nil {
		return time.Time{}, false, err
	}

	value := strings.TrimSpace(string(data))
	if value == "" {
		return time.Time{}, false, nil
	}

	syncedAt, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("invalid sync stamp for repository %s: %w", repository, err)
	}

	return syncedAt, true, nil
}

func deleteSyncStamp(repository string) error {
	path, err := syncStampPath(repository)
	if err != nil {
		return err
	}

	if err := os.Remove(path); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}

	return nil
}

func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := strings.TrimSpace(stdout.String())
	errorOutput := strings.TrimSpace(stderr.String())

	if err != nil {
		if errorOutput != "" {
			return output, fmt.Errorf("%s %s failed: %w\n%s", name, strings.Join(args, " "), err, errorOutput)
		}

		return output, fmt.Errorf("%s %s failed: %w", name, strings.Join(args, " "), err)
	}

	return output, nil
}