package cli

import (
	"strings"

	"github.com/catielanier/portico/internal/ui"
)

func renderUSEChangeTokens(flags []string) string {
	styled := make([]string, 0, len(flags))

	for _, flag := range flags {
		flag = strings.TrimSpace(flag)
		if flag == "" {
			continue
		}

		if strings.HasPrefix(flag, "-") {
			styled = append(styled, ui.Error(flag))
			continue
		}

		styled = append(styled, ui.Success(flag))
	}

	return strings.Join(styled, " ")
}

func renderInfoTokens(tokens []string) string {
	styled := make([]string, 0, len(tokens))

	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}

		styled = append(styled, ui.Info(token))
	}

	return strings.Join(styled, " ")
}
