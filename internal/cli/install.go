package cli

import (
	"fmt"

	"github.com/catielanier/portico/internal/i18n"
	"github.com/catielanier/portico/internal/jokes"
	"github.com/catielanier/portico/internal/plan"
	"github.com/catielanier/portico/internal/ui"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install <atom>",
	Short: "Configure USE flags and install a package",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		atom := args[0]

		t, err := i18n.New("en")
		if err != nil {
			return err
		}

		p := plan.Plan{
			TitleKey: "plan_title",
			Action:   "Install " + atom,
			Will: []plan.Item{
				{
					Key: "will_install_atom",
					Data: map[string]any{
						"Atom": atom,
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
					Key: jokes.RandomKey(jokes.Context{
						Atom:    atom,
						Command: "install",
					}),
				},
			},
		}

		fmt.Print(ui.RenderPlan(p, t))

		return nil
	},
}
