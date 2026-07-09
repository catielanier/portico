package portage

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type PretendResult struct {
	Atom   string
	Raw    string
	Stdout string
	Stderr string
}

func EmergePretendWithConfigRoot(atom string, configRoot string) (*PretendResult, error) {
	atom = strings.TrimSpace(atom)
	if atom == "" {
		return nil, fmt.Errorf("atom cannot be empty")
	}

	cmd := exec.Command("emerge", "--pretend", "--verbose", atom)

	cmd.Env = append(os.Environ(),
		"PORTAGE_CONFIGROOT="+configRoot,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &PretendResult{
		Atom:   atom,
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
