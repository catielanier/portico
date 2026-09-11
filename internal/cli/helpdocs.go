package cli

import (
	"strings"

	"github.com/catielanier/portico/internal/i18n"
	"github.com/spf13/cobra"
)

type commandHelpSpec struct {
	ShortKey   string
	LongKey    string
	ExampleKey string
	Data       map[string]any
}

func applyCommandHelp(root *cobra.Command, translator *i18n.Translator) {
	if root == nil {
		return
	}

	if translator == nil {
		translator = i18n.MustDefault()
	}

	helpByPath := map[string]commandHelpSpec{
		"portico": {
			ShortKey:   "root_short",
			LongKey:    "root_long",
			ExampleKey: "root_example",
		},

		"portico find": {
			ShortKey:   "find_short",
			LongKey:    "find_long",
			ExampleKey: "find_example",
		},

		"portico query": {
			ShortKey:   "query_short",
			LongKey:    "query_long",
			ExampleKey: "query_example",
		},

		"portico install": {
			ShortKey:   "install_short",
			LongKey:    "install_long",
			ExampleKey: "install_example",
		},

		"portico rebuild": {
			ShortKey:   "rebuild_short",
			LongKey:    "rebuild_long",
			ExampleKey: "rebuild_example",
		},

		"portico update": {
			ShortKey:   "update_short",
			LongKey:    "update_long",
			ExampleKey: "update_example",
		},

		"portico repo": {
			ShortKey:   "repo_short",
			LongKey:    "repo_long",
			ExampleKey: "repo_example",
			Data: map[string]any{
				"Command": "repo",
			},
		},

		"portico repo list": {
			ShortKey:   "repo_list_short",
			LongKey:    "repo_list_long",
			ExampleKey: "repo_list_example",
			Data: map[string]any{
				"Command": "repo",
			},
		},

		"portico repo add": {
			ShortKey:   "repo_add_short",
			LongKey:    "repo_add_long",
			ExampleKey: "repo_add_example",
			Data: map[string]any{
				"Command": "repo",
			},
		},

		"portico repo sync": {
			ShortKey:   "repo_sync_short",
			LongKey:    "repo_sync_long",
			ExampleKey: "repo_sync_example",
			Data: map[string]any{
				"Command": "repo",
			},
		},

		"portico repo remove": {
			ShortKey:   "repo_remove_short",
			LongKey:    "repo_remove_long",
			ExampleKey: "repo_remove_example",
			Data: map[string]any{
				"Command": "repo",
			},
		},

		"portico overlay": {
			ShortKey:   "overlay_short",
			LongKey:    "repo_long",
			ExampleKey: "repo_example",
			Data: map[string]any{
				"Command": "overlay",
			},
		},

		"portico overlay list": {
			ShortKey:   "overlay_list_short",
			LongKey:    "repo_list_long",
			ExampleKey: "repo_list_example",
			Data: map[string]any{
				"Command": "overlay",
			},
		},

		"portico overlay add": {
			ShortKey:   "overlay_add_short",
			LongKey:    "repo_add_long",
			ExampleKey: "repo_add_example",
			Data: map[string]any{
				"Command": "overlay",
			},
		},

		"portico overlay sync": {
			ShortKey:   "overlay_sync_short",
			LongKey:    "repo_sync_long",
			ExampleKey: "repo_sync_example",
			Data: map[string]any{
				"Command": "overlay",
			},
		},

		"portico overlay remove": {
			ShortKey:   "overlay_remove_short",
			LongKey:    "repo_remove_long",
			ExampleKey: "repo_remove_example",
			Data: map[string]any{
				"Command": "overlay",
			},
		},
		"portico uninstall": {
			ShortKey:   "uninstall_short",
			LongKey:    "uninstall_long",
			ExampleKey: "uninstall_example",
		},

		"portico clean": {
			ShortKey:   "clean_short",
			LongKey:    "clean_long",
			ExampleKey: "clean_example",
		},
	}

	applyCommandHelpRecursive(root, translator, helpByPath)
}

func applyCommandHelpRecursive(
	command *cobra.Command,
	translator *i18n.Translator,
	helpByPath map[string]commandHelpSpec,
) {
	if command == nil {
		return
	}

	path := normalizedCommandPath(command)
	if spec, ok := helpByPath[path]; ok {
		applyCommandHelpSpec(command, translator, spec)
	}

	for _, child := range command.Commands() {
		applyCommandHelpRecursive(child, translator, helpByPath)
	}
}

func applyCommandHelpSpec(command *cobra.Command, translator *i18n.Translator, spec commandHelpSpec) {
	data := spec.Data
	if data == nil {
		data = map[string]any{}
	}

	if spec.ShortKey != "" {
		command.Short = translator.T(spec.ShortKey, data)
	}

	if spec.LongKey != "" {
		command.Long = translator.T(spec.LongKey, data)
	}

	if spec.ExampleKey != "" {
		command.Example = translator.T(spec.ExampleKey, data)
	}
}

func normalizedCommandPath(command *cobra.Command) string {
	parts := commandPathParts(command)
	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " ")
}

func commandPathParts(command *cobra.Command) []string {
	if command == nil {
		return nil
	}

	var reversed []string

	current := command
	for current != nil {
		name := strings.TrimSpace(current.Name())
		if name != "" {
			reversed = append(reversed, name)
		}

		current = current.Parent()
	}

	parts := make([]string, 0, len(reversed))
	for i := len(reversed) - 1; i >= 0; i-- {
		parts = append(parts, reversed[i])
	}

	return parts
}