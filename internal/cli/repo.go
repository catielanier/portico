package cli

import (
	"fmt"

	"github.com/catielanier/portico/internal/repo"
	"github.com/spf13/cobra"
)

func newRepoCommand(use string, short string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
	}

	cmd.AddCommand(newRepoListCommand())
	cmd.AddCommand(newRepoAddCommand())
	cmd.AddCommand(newRepoRemoveCommand())

	return cmd
}

func newRepoListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured Portage repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := repo.NewStubManager()

			repositories, err := manager.List()
			if err != nil {
				return err
			}

			fmt.Println("Portico repositories:")
			fmt.Println()

			for _, r := range repositories {
				status := "disabled"
				if r.Enabled {
					status = "enabled"
				}

				fmt.Printf("  %-16s %s\n", r.Name, status)

				if r.Description != "" {
					fmt.Printf("    %s\n", r.Description)
				}

				if r.Location != "" {
					fmt.Printf("    location: %s\n", r.Location)
				}

				if r.SyncURI != "" {
					fmt.Printf("    sync: %s\n", r.SyncURI)
				}

				fmt.Println()
			}

			return nil
		},
	}
}

func newRepoAddCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "add <name>",
		Short: "Enable a Portage repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			fmt.Printf("Portico repo add: %s\n", name)
			fmt.Println("Repository enablement is not implemented yet.")
			fmt.Println()
			fmt.Println("Future plan:")
			fmt.Printf("  eselect repository enable %s\n", name)
			fmt.Printf("  emaint sync -r %s\n", name)

			return nil
		},
	}
}

func newRepoRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Disable a Portage repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			fmt.Printf("Portico repo remove: %s\n", name)
			fmt.Println("Repository removal is not implemented yet.")
			fmt.Println()
			fmt.Println("Future plan:")
			fmt.Printf("  eselect repository disable %s\n", name)

			return nil
		},
	}
}
