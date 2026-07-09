package portage

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type RepositorySyncer struct{}

func NewRepositorySyncer() *RepositorySyncer {
	return &RepositorySyncer{}
}

func (s *RepositorySyncer) NeedsPrivilegeEscalation() bool {
	return os.Geteuid() != 0
}

func (s *RepositorySyncer) Authenticate() error {
	if !s.NeedsPrivilegeEscalation() {
		return nil
	}

	cmd := exec.Command("sudo", "-v")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo authentication failed: %w", err)
	}

	return nil
}

func (s *RepositorySyncer) Sync(repositoryName string) error {
	return s.SyncContext(context.Background(), repositoryName)
}

func (s *RepositorySyncer) SyncContext(ctx context.Context, repositoryName string) error {
	repositoryName = strings.TrimSpace(repositoryName)
	if repositoryName == "" {
		return fmt.Errorf("repository name cannot be empty")
	}

	cmd := privilegedCommandContext(ctx, "emaint", "sync", "-r", repositoryName)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		return fmt.Errorf("emaint sync -r %s failed: %w\n%s", repositoryName, err, string(output))
	}

	if err := s.recordRepositorySyncContext(ctx, repositoryName); err != nil {
		return fmt.Errorf("failed to record sync stamp for %s: %w", repositoryName, err)
	}

	return nil
}

func (s *RepositorySyncer) recordRepositorySyncContext(ctx context.Context, repositoryName string) error {
	if !s.NeedsPrivilegeEscalation() {
		return RecordRepositorySync(repositoryName)
	}

	repositoryName = strings.TrimSpace(repositoryName)
	if repositoryName == "" {
		return nil
	}

	stampDir := filepath.Join(DefaultPorticoCacheRoot, "repo-sync")
	stampPath := repositorySyncStampPath(repositoryName)
	stampContent := time.Now().Format(time.RFC3339) + "\n"

	mkdirCmd := privilegedCommandContext(ctx, "mkdir", "-p", stampDir)
	if output, err := mkdirCmd.CombinedOutput(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		return fmt.Errorf("failed to create sync stamp directory: %w\n%s", err, string(output))
	}

	teeCmd := privilegedCommandContext(ctx, "tee", stampPath)
	teeCmd.Stdin = strings.NewReader(stampContent)

	if output, err := teeCmd.CombinedOutput(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		return fmt.Errorf("failed to write sync stamp: %w\n%s", err, string(output))
	}

	return nil
}

func privilegedCommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	if os.Geteuid() == 0 {
		return exec.CommandContext(ctx, name, args...)
	}

	sudoArgs := append([]string{"-n", name}, args...)

	return exec.CommandContext(ctx, "sudo", sudoArgs...)
}
