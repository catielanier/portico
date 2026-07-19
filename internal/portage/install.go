package portage

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type InstallProgress struct {
	CurrentPackage string
	CurrentIndex   int
	Total          int
}

type EmergeInstaller struct{}

func NewEmergeInstaller() *EmergeInstaller {
	return &EmergeInstaller{}
}

func (i *EmergeInstaller) InstallContext(
	ctx context.Context,
	atom string,
	expectedTotal int,
	onProgress func(InstallProgress),
) error {
	return i.InstallAtomsContext(ctx, []string{atom}, expectedTotal, onProgress)
}

func (i *EmergeInstaller) InstallAtomsContext(
	ctx context.Context,
	atoms []string,
	expectedTotal int,
	onProgress func(InstallProgress),
) error {
	cleanAtoms := cleanInstallAtoms(atoms)
	if len(cleanAtoms) == 0 {
		return fmt.Errorf("at least one atom is required")
	}

	args := append([]string{
		"--verbose",
		"--quiet-build=y",
	}, cleanAtoms...)

	cmd := exec.CommandContext(ctx, "emerge", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	var tail outputTail
	var scannerWait sync.WaitGroup

	scannerWait.Add(2)

	go func() {
		defer scannerWait.Done()
		scanEmergeInstallOutput(stdout, expectedTotal, &tail, onProgress)
	}()

	go func() {
		defer scannerWait.Done()
		scanEmergeInstallOutput(stderr, expectedTotal, &tail, onProgress)
	}()

	scannerWait.Wait()

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if tail.String() != "" {
			return fmt.Errorf("emerge failed: %w\n%s", err, tail.String())
		}

		return fmt.Errorf("emerge failed: %w", err)
	}

	return nil
}

type outputTail struct {
	lines []string
}

func (t *outputTail) Add(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	t.lines = append(t.lines, line)

	if len(t.lines) > 20 {
		t.lines = t.lines[len(t.lines)-20:]
	}
}

func (t *outputTail) String() string {
	return strings.Join(t.lines, "\n")
}

func scanEmergeInstallOutput(
	reader interface {
		Read([]byte) (int, error)
	},
	expectedTotal int,
	tail *outputTail,
	onProgress func(InstallProgress),
) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		tail.Add(line)

		progress, ok := parseEmergeInstallProgress(line, expectedTotal)
		if !ok {
			continue
		}

		if onProgress != nil {
			onProgress(progress)
		}
	}
}

func parseEmergeInstallProgress(line string, expectedTotal int) (InstallProgress, bool) {
	line = strings.TrimSpace(line)

	progressPattern := regexp.MustCompile(`^>>>\s+(?:Emerging|Installing)\s+\((\d+)\s+of\s+(\d+)\)\s+(.+?)\s*$`)

	matches := progressPattern.FindStringSubmatch(line)
	if matches == nil {
		return InstallProgress{}, false
	}

	currentIndex, err := strconv.Atoi(matches[1])
	if err != nil {
		return InstallProgress{}, false
	}

	total, err := strconv.Atoi(matches[2])
	if err != nil {
		total = expectedTotal
	}

	if total == 0 {
		total = expectedTotal
	}

	currentPackage := displayPackageFromEmergeToken(matches[3])

	return InstallProgress{
		CurrentPackage: currentPackage,
		CurrentIndex:   currentIndex,
		Total:          total,
	}, true
}

func displayPackageFromEmergeToken(token string) string {
	token = strings.TrimSpace(token)

	if strings.Contains(token, " ") {
		token = strings.Fields(token)[0]
	}

	if beforeRepository, _, ok := strings.Cut(token, "::"); ok {
		token = beforeRepository
	}

	return token
}

func cleanInstallAtoms(atoms []string) []string {
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
