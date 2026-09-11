package portage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const PorticoAcceptKeywordsFile = "90-portico"

func WriteAcceptKeywordEntry(configPath string, atom string, keyword string) (string, error) {
	atom = strings.TrimSpace(atom)
	keyword = strings.TrimSpace(keyword)

	if atom == "" {
		return "", fmt.Errorf("atom cannot be empty")
	}

	if keyword == "" {
		return "", fmt.Errorf("keyword cannot be empty")
	}

	acceptKeywordsDir := filepath.Join(configPath, "package.accept_keywords")

	if err := os.MkdirAll(acceptKeywordsDir, 0o755); err != nil {
		return "", err
	}

	path := filepath.Join(acceptKeywordsDir, PorticoAcceptKeywordsFile)

	line := fmt.Sprintf("%s %s\n", atom, keyword)

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
