// internal/cli/install.go
// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"fmt"
	"strings"

	"github.com/catielanier/portico/internal/i18n"
	"github.com/catielanier/portico/internal/jokes"
	"github.com/catielanier/portico/internal/plan"
	"github.com/catielanier/portico/internal/portage"
	"github.com/catielanier/portico/internal/ui"
	"github.com/catielanier/portico/internal/useflags"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install <atom>",
	Short: "Configure USE flags and install a package",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireRoot("install packages"); err != nil {
			return err
		}

		atom := args[0]

		if err := syncRepositoriesForMutation(); err != nil {
			return err
		}

		querier := portage.NewEqueryQuerier()

		queryResult, err := querier.Query(atom)
		if err != nil {
			return err
		}

		selections := useflags.FromQuery(queryResult)

		selected, ok, err := ui.RunUsePicker(atom, selections)
		if err != nil {
			return err
		}

		if !ok {
			fmt.Println("Install cancelled.")
			return nil
		}

		t, err := i18n.New("en")
		if err != nil {
			return err
		}

		renderInstallPrototype(queryResult, selected, t)

		return nil
	},
}

func renderInstallPrototype(queryResult *portage.PackageQuery, selections []useflags.FlagSelection, t *i18n.Translator) {
	atom := queryResult.Atom

	p := plan.Plan{
		TitleKey: "plan_title",
		Action:   "Install " + atom,
		Will: []plan.Item{
			{
				Key: "will_inspect_use_flags",
				Data: map[string]any{
					"Atom": atom,
				},
			},
			{
				Key: "will_write_package_use",
				Data: map[string]any{
					"Path": "/etc/portage/package.use/portico",
				},
			},
			{
				Key: "will_run_emerge_pretend",
				Data: map[string]any{
					"Atom": atom,
				},
			},
			{
				Key: "will_ask_before_emerge",
				Data: map[string]any{
					"Atom": atom,
				},
			},
		},
		WillNot: []plan.Item{
			{
				Key: "will_not_modify_global_use",
				Data: map[string]any{
					"Path": "/etc/portage/make.conf",
				},
			},
			{
				Key: "will_not_overwrite_package_use",
			},
			{
				Key: "will_not_install_in_prototype",
			},
			{
				Key: jokes.RandomKey(jokes.Context{
					Atom:    atom,
					Command: "install",
				}),
			},
		},
	}

	fmt.Println("Portico Install")
	fmt.Println()
	fmt.Println("Package:")
	fmt.Printf("  %s\n", atom)
	fmt.Println()

	renderUseFlagSummary(queryResult)

	selectedFlags := useflags.SelectedFlags(selections)

	fmt.Println()

	if len(selectedFlags) > 0 {
		fmt.Println("Selected package.use entry:")
		fmt.Printf("  %s %s\n", atom, strings.Join(selectedFlags, " "))
	} else {
		fmt.Println("Selected package.use entry:")
		fmt.Println("  No explicit USE flag changes selected.")
	}

	fmt.Println()
	fmt.Print(ui.RenderPlan(p, t))
}
