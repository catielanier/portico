package cli

import (
	"fmt"
	"regexp"

	"github.com/spf13/cobra"
)

var simpleAtomPattern = regexp.MustCompile(`^[A-Za-z0-9+_.-]+/[A-Za-z0-9+_.-]+$`)

func validateSingleAtomArg(cmd *cobra.Command, args []string) error {
	if err := cobra.ExactArgs(1)(cmd, args); err != nil {
		return err
	}

	return validateAtomShape(args[0])
}

func validateOneOrMoreAtomArgs(cmd *cobra.Command, args []string) error {
	if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
		return err
	}

	for _, arg := range args {
		if err := validateAtomShape(arg); err != nil {
			return err
		}
	}

	return nil
}

func validateAtomShape(atom string) error {
	if simpleAtomPattern.MatchString(atom) {
		return nil
	}

	return fmt.Errorf(
		"invalid atom %q: expected category/package, for example net-irc/irssi",
		atom,
	)
}
