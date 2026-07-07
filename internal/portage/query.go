// internal/portage/query.go
// SPDX-License-Identifier: GPL-3.0-or-later

package portage

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type PackageQuery struct {
	Atom    string
	Uses    []UseFlag
	RawUses string
	Found   bool
}

type UseFlag struct {
	Name            string
	Description     string
	EnabledForBuild bool
	Installed       *bool
	Raw             string
}

type PackageQuerier interface {
	Query(atom string) (*PackageQuery, error)
}

type EqueryQuerier struct{}

func NewEqueryQuerier() *EqueryQuerier {
	return &EqueryQuerier{}
}

func (q *EqueryQuerier) Query(atom string) (*PackageQuery, error) {
	atom = strings.TrimSpace(atom)
	if atom == "" {
		return nil, fmt.Errorf("package atom cannot be empty")
	}

	rawUses, err := q.equeryUses(atom)
	if err != nil {
		return nil, err
	}

	uses := ParseEqueryUses(rawUses)

	return &PackageQuery{
		Atom:    atom,
		Uses:    uses,
		RawUses: rawUses,
		Found:   len(uses) > 0,
	}, nil
}

func (q *EqueryQuerier) equeryUses(atom string) (string, error) {
	cmd := exec.Command("equery", "-C", "-N", "u", atom)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("equery uses failed: %s", strings.TrimSpace(stderr.String()))
		}

		return "", fmt.Errorf("equery uses failed: %w", err)
	}

	return stdout.String(), nil
}

func ParseEqueryUses(raw string) []UseFlag {
	lines := strings.Split(raw, "\n")

	verboseUseLinePattern := regexp.MustCompile(`^\s*([+-])\s+([+-])\s+([A-Za-z0-9_+.-]+)\s*:\s*(.*)$`)
	compactUseLinePattern := regexp.MustCompile(`^\s*([+-])([A-Za-z0-9_+.-]+)\s*$`)

	var flags []UseFlag
	var current *UseFlag

	flush := func() {
		if current == nil {
			return
		}

		current.Description = strings.Join(strings.Fields(current.Description), " ")
		flags = append(flags, *current)
		current = nil
	}

	for _, line := range lines {
		if matches := verboseUseLinePattern.FindStringSubmatch(line); matches != nil {
			flush()

			installedValue := matches[2] == "+"

			current = &UseFlag{
				Name:            matches[3],
				Description:     strings.TrimSpace(matches[4]),
				EnabledForBuild: matches[1] == "+",
				Installed:       &installedValue,
				Raw:             line,
			}

			continue
		}

		if matches := compactUseLinePattern.FindStringSubmatch(line); matches != nil {
			flush()

			flags = append(flags, UseFlag{
				Name:            matches[2],
				Description:     "",
				EnabledForBuild: matches[1] == "+",
				Installed:       nil,
				Raw:             line,
			})

			continue
		}

		if current != nil {
			trimmed := strings.TrimSpace(line)

			if trimmed != "" &&
				!strings.HasPrefix(trimmed, "[") &&
				!strings.HasPrefix(trimmed, "*") &&
				trimmed != "U I" {
				current.Description += " " + trimmed
			}
		}
	}

	flush()

	return flags
}
