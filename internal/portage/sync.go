package portage

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type RepositorySyncer struct{}

func NewRepositorySyncer() *RepositorySyncer {
	return &RepositorySyncer{}
}

func (s *RepositorySyncer) Sync(repositoryName string) error {
	repositoryName = strings.TrimSpace(repositoryName)
	if repositoryName == "" {
		return fmt.Errorf("repository name cannot be empty")
	}

	cmd := exec.Command("emaint", "sync", "-r", repositoryName)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("emaint sync failed for %s: %s", repositoryName, strings.TrimSpace(stderr.String()))
		}

		return fmt.Errorf("emaint sync failed for %s: %w", repositoryName, err)
	}

	return nil
}
