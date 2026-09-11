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
)

const installedPackageDatabasePath = "/var/db/pkg"

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
	name = strings.TrimSpace(name)
	if name == "" {
		return false, fmt.Errorf("repository name is required")
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
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("repository name is required")
	}

	enabled, err := m.IsEnabled(name)
	if err != nil {
		return err
	}

	if !enabled {
		if _, err := runCommand("eselect", "repository", "enable", name); err != nil {
			return err
		}
	}

	return m.Sync(name)
}

func (m *Manager) Sync(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("repository name is required")
	}

	enabled, err := m.IsEnabled(name)
	if err != nil {
		return err
	}

	if !enabled {
		return fmt.Errorf("%w: %s", ErrRepositoryNotEnabled, name)
	}

	_, err = runCommand("emaint", "sync", "-r", name)
	return err
}

func (m *Manager) Remove(name string, force bool) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("repository name is required")
	}

	enabled, err := m.IsEnabled(name)
	if err != nil {
		return err
	}

	if !enabled {
		return fmt.Errorf("%w: %s", ErrRepositoryNotEnabled, name)
	}

	installedPackages, err := InstalledPackagesFromRepository(name)
	if err != nil {
		return err
	}

	if len(installedPackages) > 0 && !force {
		return &RepositoryInUseError{
			Repository: name,
			Packages:   installedPackages,
		}
	}

	_, err = runCommand("eselect", "repository", "disable", name)
	return err
}

func InstalledPackagesFromRepository(repository string) ([]InstalledPackage, error) {
	repository = strings.TrimSpace(repository)
	if repository == "" {
		return nil, fmt.Errorf("repository name is required")
	}

	var packages []InstalledPackage

	err := filepath.WalkDir(installedPackageDatabasePath, func(path string, entry os.DirEntry, walkErr error) error {
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