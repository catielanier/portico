package portage

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type EmergeCleaner struct{}

func NewEmergeCleaner() *EmergeCleaner {
	return &EmergeCleaner{}
}

func (c *EmergeCleaner) Unmerge(atoms []string) error {
	return c.UnmergeContext(context.Background(), atoms)
}

func (c *EmergeCleaner) UnmergeContext(ctx context.Context, atoms []string) error {
	cleanAtoms := cleanPackageAtoms(atoms)
	if len(cleanAtoms) == 0 {
		return fmt.Errorf("at least one atom is required")
	}

	args := []string{
		"--verbose",
		"--unmerge",
	}
	args = append(args, cleanAtoms...)

	return runInteractiveEmerge(ctx, args)
}

func (c *EmergeCleaner) Depclean() error {
	return c.DepcleanContext(context.Background())
}

func (c *EmergeCleaner) DepcleanContext(ctx context.Context) error {
	args := []string{
		"--verbose",
		"--depclean",
	}

	return runInteractiveEmerge(ctx, args)
}

func runInteractiveEmerge(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, "emerge", args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		return fmt.Errorf("emerge %s failed: %w", strings.Join(args, " "), err)
	}

	return nil
}

func cleanPackageAtoms(atoms []string) []string {
	var out []string
	seen := make(map[string]bool)

	for _, atom := range atoms {
		atom = strings.TrimSpace(atom)
		if atom == "" {
			continue
		}

		if seen[atom] {
			continue
		}

		seen[atom] = true
		out = append(out, atom)
	}

	return out
}