package cli

import (
	"fmt"
	"regexp"
	"strings"
)

var packageVersionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]*$`)

type packageTarget struct {
	Atom    string
	Version string
}

func parsePackageTarget(value string) (packageTarget, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return packageTarget{}, fmt.Errorf("package target cannot be empty")
	}

	parts := strings.Split(value, "@")
	if len(parts) > 2 {
		return packageTarget{}, fmt.Errorf("invalid package target %q: expected atom or atom@version", value)
	}

	atom := strings.TrimSpace(parts[0])
	if err := validateAtomShape(atom); err != nil {
		return packageTarget{}, err
	}

	target := packageTarget{
		Atom: atom,
	}

	if len(parts) == 1 {
		return target, nil
	}

	version := strings.TrimSpace(parts[1])
	if err := validatePackageVersion(version); err != nil {
		return packageTarget{}, err
	}

	target.Version = version

	return target, nil
}

func parsePackageTargets(args []string) ([]packageTarget, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("at least one package target is required")
	}

	targets := make([]packageTarget, 0, len(args))

	for _, arg := range args {
		target, err := parsePackageTarget(arg)
		if err != nil {
			return nil, err
		}

		targets = append(targets, target)
	}

	return targets, nil
}

func packageAtomsFromArgs(args []string) ([]string, error) {
	targets, err := parsePackageTargets(args)
	if err != nil {
		return nil, err
	}

	atoms := make([]string, 0, len(targets))
	seen := make(map[string]bool)

	for _, target := range targets {
		atom := target.PortageAtom()

		if seen[atom] {
			continue
		}

		seen[atom] = true
		atoms = append(atoms, atom)
	}

	return atoms, nil
}

func packageAtomFromArg(arg string) (string, error) {
	target, err := parsePackageTarget(arg)
	if err != nil {
		return "", err
	}

	return target.PortageAtom(), nil
}

func validatePackageVersion(packageVersion string) error {
	packageVersion = strings.TrimSpace(packageVersion)

	if packageVersion == "" {
		return fmt.Errorf("package version cannot be empty")
	}

	if packageVersion == "9999" {
		return fmt.Errorf("live ebuild version 9999 is not supported by atom@version yet")
	}

	if strings.HasPrefix(packageVersion, "-") {
		return fmt.Errorf("package version cannot start with '-'")
	}

	if strings.Contains(packageVersion, "@") {
		return fmt.Errorf("package version cannot contain '@'")
	}

	if strings.Contains(packageVersion, "/") {
		return fmt.Errorf("package version cannot contain '/'")
	}

	if !packageVersionPattern.MatchString(packageVersion) {
		return fmt.Errorf(
			"invalid package version %q: expected a Gentoo-style version such as 1.4.5 or 0.62.2-r1",
			packageVersion,
		)
	}

	return nil
}

func (t packageTarget) PortageAtom() string {
	if t.Version == "" {
		return t.Atom
	}

	return exactVersionAtom(t.Atom, t.Version)
}

func exactVersionAtom(atom string, packageVersion string) string {
	return "=" + strings.TrimSpace(atom) + "-" + strings.TrimSpace(packageVersion)
}
