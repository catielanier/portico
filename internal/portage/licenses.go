package portage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const PorticoPackageLicenseFile = "90-portico"

func WritePackageLicenseEntry(configPath string, atom string, licenses []string) (string, error) {
	atom = strings.TrimSpace(atom)

	if atom == "" {
		return "", fmt.Errorf("atom cannot be empty")
	}

	cleanLicenses := cleanLicenseList(licenses)
	if len(cleanLicenses) == 0 {
		return "", fmt.Errorf("at least one license is required")
	}

	packageLicenseDir := filepath.Join(configPath, "package.license")

	if err := os.MkdirAll(packageLicenseDir, 0o755); err != nil {
		return "", err
	}

	path := filepath.Join(packageLicenseDir, PorticoPackageLicenseFile)

	line := fmt.Sprintf("%s %s\n", atom, strings.Join(cleanLicenses, " "))

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := file.WriteString(line); err != nil {
		return "", err
	}

	return path, nil
}

func cleanLicenseList(licenses []string) []string {
	var out []string
	seen := make(map[string]bool)

	for _, license := range licenses {
		license = strings.TrimSpace(license)
		license = strings.Trim(license, ",")

		if license == "" {
			continue
		}

		if seen[license] {
			continue
		}

		seen[license] = true
		out = append(out, license)
	}

	return out
}
