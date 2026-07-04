package cli

import (
	"fmt"

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
		atom, err := requireAtom(args)
		if err != nil {
			return err
		}

		p := plan.Plan{
			Title:  "Portico Plan",
			Action: "Install " + atom,
			Will: []plan.Item{
				{Text: "install " + atom},
				{Text: "run emerge --pretend --verbose " + atom},
				{Text: "ask before running emerge --ask --verbose " + atom},
			},
			WillNot: []plan.Item{
				{Text: "modify global USE flags in /etc/portage/make.conf"},
				{Text: "overwrite user-managed package.use entries"},
				{Text: jokes.Random(jokes.Context{Atom: atom, Command: "install"})},
			},
		}

		fmt.Print(ui.RenderPlan(p))

		return nil
	},
}
