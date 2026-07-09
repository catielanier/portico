package portage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const PorticoPackageUseFile = "90-portico"

func WritePackageUseEntry(configPath string, atom string, flags []string) (string, error) {
	atom = strings.TrimSpace(atom)
	if atom == "" {
		return "", fmt.Errorf("atom cannot be empty")
	}

	packageUseDir := filepath.Join(configPath, "package.use")

	if err := os.MkdirAll(packageUseDir, 0o755); err != nil {
		return "", err
	}

	path := filepath.Join(packageUseDir, PorticoPackageUseFile)

	if len(flags) == 0 {
		return path, nil
	}

	line := fmt.Sprintf("%s %s\n", atom, strings.Join(flags, " "))

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
