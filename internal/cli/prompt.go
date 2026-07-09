package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func confirmDefaultNo(prompt string) (bool, error) {
	fmt.Printf("%s [y/N] ", prompt)

	reader := bufio.NewReader(os.Stdin)

	answer, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	answer = strings.ToLower(strings.TrimSpace(answer))

	return answer == "y" || answer == "yes", nil
}
