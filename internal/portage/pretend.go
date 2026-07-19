package portage

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type PretendOptions struct {
	OneShot bool
}

type PretendResult struct {
	Atoms  []string
	Atom   string
	Raw    string
	Stdout string
	Stderr string
}

func EmergePretendWithConfigRoot(atom string, configRoot string) (*PretendResult, error) {
	return EmergePretendWithConfigRootForAtoms([]string{atom}, configRoot)
}

func EmergePretendWithConfigRootForAtoms(atoms []string, configRoot string) (*PretendResult, error) {
	return EmergePretendWithConfigRootForAtomsWithOptions(atoms, configRoot, PretendOptions{})
}

func EmergePretendOneshotWithConfigRootForAtoms(atoms []string, configRoot string) (*PretendResult, error) {
	return EmergePretendWithConfigRootForAtomsWithOptions(atoms, configRoot, PretendOptions{
		OneShot: true,
	})
}

func EmergePretendWithConfigRootForAtomsWithOptions(
	atoms []string,
	configRoot string,
	options PretendOptions,
) (*PretendResult, error) {
	cleanAtoms := cleanAtoms(atoms)
	if len(cleanAtoms) == 0 {
		return nil, fmt.Errorf("at least one atom is required")
	}

	args := []string{
		"--pretend",
		"--verbose",
	}

	if options.OneShot {
		args = append(args, "--oneshot")
	}

	args = append(args, cleanAtoms...)

	cmd := exec.Command("emerge", args...)

	cmd.Env = append(os.Environ(),
		"PORTAGE_CONFIGROOT="+configRoot,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &PretendResult{
		Atoms:  cleanAtoms,
		Atom:   cleanAtoms[0],
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Raw:    stdout.String() + stderr.String(),
	}

	if err != nil {
		if strings.TrimSpace(result.Raw) != "" {
			return result, fmt.Errorf("emerge --pretend failed")
		}

		return result, err
	}

	return result, nil
}

func cleanAtoms(atoms []string) []string {
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
