package portage

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type CleanupMode string

const (
	CleanupModeUnmerge  CleanupMode = "unmerge"
	CleanupModeDepclean CleanupMode = "depclean"
)

type CleanupProgress struct {
	Mode           CleanupMode
	CurrentPackage string
	CurrentIndex   int
	Total          int
	Waiting        bool
	Message        string
}

type EmergeCleaner struct{}

func NewEmergeCleaner() *EmergeCleaner {
	return &EmergeCleaner{}
}

func (c *EmergeCleaner) PretendUnmerge(atoms []string) (string, error) {
	cleanAtoms := cleanCleanupAtoms(atoms)
	if len(cleanAtoms) == 0 {
		return "", fmt.Errorf("at least one atom is required")
	}

	args := []string{
		"--verbose",
		"--pretend",
		"--unmerge",
	}
	args = append(args, cleanAtoms...)

	return runCapturedEmerge(args)
}

func (c *EmergeCleaner) PretendDepclean() (string, error) {
	return runCapturedEmerge([]string{
		"--verbose",
		"--pretend",
		"--depclean",
	})
}

func (c *EmergeCleaner) UnmergeContext(
	ctx context.Context,
	atoms []string,
	expectedTotal int,
	onProgress func(CleanupProgress),
) error {
	cleanAtoms := cleanCleanupAtoms(atoms)
	if len(cleanAtoms) == 0 {
		return fmt.Errorf("at least one atom is required")
	}

	args := []string{
		"--verbose",
		"--unmerge",
	}
	args = append(args, cleanAtoms...)

	return runCleanupContext(ctx, CleanupModeUnmerge, args, expectedTotal, onProgress)
}

func (c *EmergeCleaner) DepcleanContext(
	ctx context.Context,
	expectedTotal int,
	onProgress func(CleanupProgress),
) error {
	return runCleanupContext(ctx, CleanupModeDepclean, []string{
		"--verbose",
		"--depclean",
	}, expectedTotal, onProgress)
}

func runCapturedEmerge(args []string) (string, error) {
	cmd := exec.Command("emerge", args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := strings.TrimSpace(strings.Join([]string{
		stdout.String(),
		stderr.String(),
	}, "\n"))

	if err != nil {
		if output != "" {
			return output, fmt.Errorf("emerge %s failed: %w", strings.Join(args, " "), err)
		}

		return output, fmt.Errorf("emerge %s failed: %w", strings.Join(args, " "), err)
	}

	return output, nil
}

func runCleanupContext(
	ctx context.Context,
	mode CleanupMode,
	args []string,
	expectedTotal int,
	onProgress func(CleanupProgress),
) error {
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

	var tail cleanupOutputTail
	var scannerWait sync.WaitGroup

	scannerWait.Add(2)

	go func() {
		defer scannerWait.Done()
		scanCleanupOutput(stdout, mode, expectedTotal, &tail, onProgress)
	}()

	go func() {
		defer scannerWait.Done()
		scanCleanupOutput(stderr, mode, expectedTotal, &tail, onProgress)
	}()

	scannerWait.Wait()

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if tail.String() != "" {
			return fmt.Errorf("emerge %s failed: %w\n%s", strings.Join(args, " "), err, tail.String())
		}

		return fmt.Errorf("emerge %s failed: %w", strings.Join(args, " "), err)
	}

	return nil
}

func scanCleanupOutput(
	reader io.Reader,
	mode CleanupMode,
	expectedTotal int,
	tail *cleanupOutputTail,
	onProgress func(CleanupProgress),
) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		tail.Add(line)

		progress, ok := parseCleanupProgressLine(mode, line, expectedTotal)
		if !ok {
			continue
		}

		if onProgress != nil {
			onProgress(progress)
		}
	}
}

func parseCleanupProgressLine(mode CleanupMode, line string, expectedTotal int) (CleanupProgress, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return CleanupProgress{}, false
	}

	if isCleanupCountdownLine(line) {
		return CleanupProgress{
			Mode:    mode,
			Total:   expectedTotal,
			Waiting: true,
			Message: line,
		}, true
	}

	progressPattern := regexp.MustCompile(`^>>>\s+Unmerging\s+\((\d+)\s+of\s+(\d+)\)\s+(.+?)\s*$`)
	matches := progressPattern.FindStringSubmatch(line)
	if matches == nil {
		return CleanupProgress{}, false
	}

	currentIndex, err := strconv.Atoi(matches[1])
	if err != nil {
		return CleanupProgress{}, false
	}

	total, err := strconv.Atoi(matches[2])
	if err != nil {
		total = expectedTotal
	}

	if total == 0 {
		total = expectedTotal
	}

	return CleanupProgress{
		Mode:           mode,
		CurrentPackage: displayCleanupPackage(matches[3]),
		CurrentIndex:   currentIndex,
		Total:          total,
		Waiting:        false,
		Message:        line,
	}, true
}

func isCleanupCountdownLine(line string) bool {
	lower := strings.ToLower(line)

	if strings.Contains(lower, "waiting") && strings.Contains(lower, "seconds") {
		return true
	}

	if strings.Contains(lower, "press ctrl-c") || strings.Contains(lower, "press ctrl+c") {
		return true
	}

	return false
}

func displayCleanupPackage(token string) string {
	token = strings.TrimSpace(token)

	if strings.Contains(token, " ") {
		token = strings.Fields(token)[0]
	}

	if beforeRepository, _, ok := strings.Cut(token, "::"); ok {
		token = beforeRepository
	}

	return token
}

func ParseCleanupTotal(raw string) int {
	lines := strings.Split(raw, "\n")

	totalPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)total:\s+(\d+)\s+package`),
		regexp.MustCompile(`(?i)selected:\s+(\d+)\s+package`),
		regexp.MustCompile(`(?i)packages selected for removal:\s+(\d+)`),
	}

	for _, line := range lines {
		for _, pattern := range totalPatterns {
			matches := pattern.FindStringSubmatch(line)
			if matches == nil {
				continue
			}

			total, err := strconv.Atoi(matches[1])
			if err != nil {
				continue
			}

			return total
		}
	}

	return 0
}

func cleanCleanupAtoms(atoms []string) []string {
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

type cleanupOutputTail struct {
	mu    sync.Mutex
	lines []string
}

func (t *cleanupOutputTail) Add(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.lines = append(t.lines, line)

	if len(t.lines) > 20 {
		t.lines = t.lines[len(t.lines)-20:]
	}
}

func (t *cleanupOutputTail) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	return strings.Join(t.lines, "\n")
}