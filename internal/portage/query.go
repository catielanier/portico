package portage

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type PackageQuery struct {
	Atom        string
	Uses        []UseFlag
	RawUses     string
	RequiredUse string
	Found       bool
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

type EqueryQuerier struct {
	ConfigRoot string
}

func NewEqueryQuerier() *EqueryQuerier {
	return &EqueryQuerier{}
}

func NewEqueryQuerierWithConfigRoot(configRoot string) *EqueryQuerier {
	return &EqueryQuerier{
		ConfigRoot: strings.TrimSpace(configRoot),
	}
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

	// REQUIRED_USE is version-specific ebuild metadata. If metadata lookup is
	// unavailable for a particular atom, keep the normal query usable and let
	// the authoritative emerge --pretend pass catch any constraint failure.
	requiredUse, _ := q.requiredUse(atom)

	return &PackageQuery{
		Atom:        atom,
		Uses:        uses,
		RawUses:     rawUses,
		RequiredUse: strings.TrimSpace(requiredUse),
		Found:       len(uses) > 0,
	}, nil
}

func (q *EqueryQuerier) equeryUses(atom string) (string, error) {
	cmd := exec.Command("equery", "-C", "-N", "u", atom)
	cmd.Env = q.commandEnv()

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

func (q *EqueryQuerier) requiredUse(atom string) (string, error) {
	metadataTarget, err := q.bestVisibleEbuild(atom)
	if err != nil {
		return "", err
	}

	if repository := repositoryQualifier(atom); repository != "" {
		metadataTarget += "::" + repository
	}

	cmd := exec.Command(
		"portageq",
		"metadata",
		"/",
		"ebuild",
		metadataTarget,
		"REQUIRED_USE",
	)
	cmd.Env = q.commandEnv()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("portageq metadata failed: %s", strings.TrimSpace(stderr.String()))
		}

		return "", fmt.Errorf("portageq metadata failed: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func (q *EqueryQuerier) bestVisibleEbuild(atom string) (string, error) {
	cmd := exec.Command(
		"portageq",
		"best_visible",
		"/",
		"ebuild",
		atom,
	)
	cmd.Env = q.commandEnv()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("portageq best_visible failed: %s", strings.TrimSpace(stderr.String()))
		}

		return "", fmt.Errorf("portageq best_visible failed: %w", err)
	}

	best := strings.TrimSpace(stdout.String())
	if best == "" {
		return "", fmt.Errorf("portageq best_visible returned no package for %s", atom)
	}

	return best, nil
}

func repositoryQualifier(atom string) string {
	_, repository, found := strings.Cut(strings.TrimSpace(atom), "::")
	if !found {
		return ""
	}

	repository = strings.TrimSpace(repository)
	if repository == "" {
		return ""
	}

	if end := strings.IndexAny(repository, "[ \t\r\n"); end >= 0 {
		repository = repository[:end]
	}

	return repository
}

func (q *EqueryQuerier) commandEnv() []string {
	configRoot := strings.TrimSpace(q.ConfigRoot)
	if configRoot == "" {
		return os.Environ()
	}

	env := make([]string, 0, len(os.Environ())+1)
	for _, value := range os.Environ() {
		if strings.HasPrefix(value, "PORTAGE_CONFIGROOT=") {
			continue
		}

		env = append(env, value)
	}

	return append(env, "PORTAGE_CONFIGROOT="+configRoot)
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
