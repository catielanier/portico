package ui

import (
	"os"
	"strings"
	"testing"
)

func TestColorDisabledWhenOutputIsNotTerminal(t *testing.T) {
	withColorEnvironment(t, func() {
		if colorEnabledForTerminal(false) {
			t.Fatal("expected color to be disabled for non-terminal output")
		}
	})
}

func TestColorDisabledByNoColor(t *testing.T) {
	withColorEnvironment(t, func() {
		if err := os.Setenv("NO_COLOR", "1"); err != nil {
			t.Fatal(err)
		}

		if colorEnabledForTerminal(true) {
			t.Fatal("expected NO_COLOR to disable color")
		}
	})
}

func TestColorDisabledForDumbTerminal(t *testing.T) {
	withColorEnvironment(t, func() {
		if err := os.Setenv("TERM", "dumb"); err != nil {
			t.Fatal(err)
		}

		if colorEnabledForTerminal(true) {
			t.Fatal("expected TERM=dumb to disable color")
		}
	})
}

func TestColorDisabledByCliColorZero(t *testing.T) {
	withColorEnvironment(t, func() {
		if err := os.Setenv("CLICOLOR", "0"); err != nil {
			t.Fatal(err)
		}

		if colorEnabledForTerminal(true) {
			t.Fatal("expected CLICOLOR=0 to disable color")
		}
	})
}

func TestColorEnabledForNormalTerminal(t *testing.T) {
	withColorEnvironment(t, func() {
		if err := os.Setenv("TERM", "xterm-256color"); err != nil {
			t.Fatal(err)
		}

		if !colorEnabledForTerminal(true) {
			t.Fatal("expected color to be enabled for a normal terminal")
		}
	})
}

func TestSemanticStylesRenderPlainTextWithoutColor(t *testing.T) {
	withColorEnvironment(t, func() {
		if err := os.Setenv("NO_COLOR", "1"); err != nil {
			t.Fatal(err)
		}

		for name, value := range map[string]string{
			"success":  Success("success"),
			"error":    Error("error"),
			"warning":  Warning("warning"),
			"info":     Info("info"),
			"accent":   Accent("accent"),
			"muted":    Muted("muted"),
			"selected": Selected("selected"),
			"disabled": Disabled("disabled"),
		} {
			if strings.Contains(value, "\x1b[") {
				t.Fatalf("%s style emitted ANSI escapes with color disabled: %q", name, value)
			}
		}
	})
}

func withColorEnvironment(t *testing.T, run func()) {
	t.Helper()

	keys := []string{"NO_COLOR", "TERM", "CLICOLOR"}
	type savedValue struct {
		value string
		set   bool
	}

	saved := make(map[string]savedValue, len(keys))
	for _, key := range keys {
		value, set := os.LookupEnv(key)
		saved[key] = savedValue{value: value, set: set}
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}

	defer func() {
		for _, key := range keys {
			value := saved[key]
			if value.set {
				_ = os.Setenv(key, value.value)
			} else {
				_ = os.Unsetenv(key)
			}
		}
	}()

	run()
}
