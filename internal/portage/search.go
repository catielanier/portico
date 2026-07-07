package portage

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type SearchResult struct {
	Atom        string
	Category    string
	Name        string
	Description string
	Installed   bool
	Sources     []PackageSource
	Raw         string
}

type PackageSource struct {
	Repository string
	Version    string
	Status     string
	Masked     bool
	Installed  bool
}

type Searcher interface {
	Find(query string) ([]SearchResult, error)
}

type EmergeSearcher struct{}

func NewEmergeSearcher() *EmergeSearcher {
	return &EmergeSearcher{}
}

func (s *EmergeSearcher) Find(query string) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	cmd := exec.Command("emerge", "--search", query)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("emerge --search failed: %s", strings.TrimSpace(stderr.String()))
		}

		return nil, fmt.Errorf("emerge --search failed: %w", err)
	}

	return ParseEmergeSearch(stdout.String()), nil
}

func ParseEmergeSearch(raw string) []SearchResult {
	lines := strings.Split(raw, "\n")

	atomPattern := regexp.MustCompile(`^\*\s+([A-Za-z0-9+_.-]+/[A-Za-z0-9+_.-]+)`)
	descPattern := regexp.MustCompile(`^\s+Description:\s+(.+)$`)
	installedPattern := regexp.MustCompile(`^\s+Installed:\s+(.+)$`)

	var results []SearchResult
	var current *SearchResult
	var rawLines []string

	flush := func() {
		if current == nil {
			return
		}

		current.Raw = strings.Join(rawLines, "\n")
		results = append(results, *current)

		current = nil
		rawLines = nil
	}

	for _, line := range lines {
		if matches := atomPattern.FindStringSubmatch(line); matches != nil {
			flush()

			atom := matches[1]
			category, name := splitAtom(atom)

			current = &SearchResult{
				Atom:     atom,
				Category: category,
				Name:     name,
			}

			rawLines = append(rawLines, line)
			continue
		}

		if current == nil {
			continue
		}

		rawLines = append(rawLines, line)

		if matches := descPattern.FindStringSubmatch(line); matches != nil {
			current.Description = strings.TrimSpace(matches[1])
			continue
		}

		if matches := installedPattern.FindStringSubmatch(line); matches != nil {
			installedText := strings.TrimSpace(matches[1])
			current.Installed = installedText != "" && installedText != "[ Not Installed ]"
			continue
		}
	}

	flush()

	return results
}

func splitAtom(atom string) (string, string) {
	parts := strings.SplitN(atom, "/", 2)
	if len(parts) != 2 {
		return "", atom
	}

	return parts[0], parts[1]
}
