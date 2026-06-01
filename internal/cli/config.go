package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kespineira/r53ctl/internal/output"
	"github.com/kespineira/r53ctl/internal/settings"
)

type configView struct {
	Path    string `json:"path"`
	Profile string `json:"profile"`
	Region  string `json:"region"`
	Output  string `json:"output"`
}

func newConfigCommand(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage r53ctl configuration defaults",
	}
	cmd.AddCommand(newConfigViewCommand(a))
	cmd.AddCommand(newConfigGetCommand(a))
	cmd.AddCommand(newConfigSetCommand(a))
	cmd.AddCommand(newConfigUnsetCommand(a))
	cmd.AddCommand(newConfigPathCommand(a))
	return cmd
}

func newConfigViewCommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "view",
		Short: "Show current configuration and file path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := a.settingsPath()
			if err != nil {
				return err
			}
			s, err := settings.Load(path)
			if err != nil {
				return err
			}
			if !settings.ValidOutput(a.awsFlags.Output) {
				return fmt.Errorf("unsupported output format %q", a.awsFlags.Output)
			}
			if a.awsFlags.Output == "json" {
				return output.JSON(a.out, configView{
					Path:    path,
					Profile: s.Profile,
					Region:  s.Region,
					Output:  s.Output,
				})
			}
			if _, err := fmt.Fprintf(a.out, "config: %s\n", path); err != nil {
				return err
			}
			for _, key := range settings.Keys {
				value, _ := s.Get(key)
				if _, err := fmt.Fprintf(a.out, "%-8s %s\n", key, value); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func newConfigGetCommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Print a single configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := a.settingsPath()
			if err != nil {
				return err
			}
			s, err := settings.Load(path)
			if err != nil {
				return err
			}
			value, err := s.Get(args[0])
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(a.out, value)
			return err
		},
	}
}

func newConfigSetCommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := a.settingsPath()
			if err != nil {
				return err
			}
			s, err := settings.Load(path)
			if err != nil {
				return err
			}
			if err := s.Set(args[0], args[1]); err != nil {
				return err
			}
			if err := settings.Save(path, s); err != nil {
				return err
			}
			_, err = fmt.Fprintf(a.out, "set %s to %q\n", args[0], args[1])
			return err
		},
	}
}

func newConfigUnsetCommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "unset <key>",
		Short: "Clear a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := a.settingsPath()
			if err != nil {
				return err
			}
			s, err := settings.Load(path)
			if err != nil {
				return err
			}
			if err := s.Unset(args[0]); err != nil {
				return err
			}
			if err := settings.Save(path, s); err != nil {
				return err
			}
			_, err = fmt.Fprintf(a.out, "unset %s\n", args[0])
			return err
		},
	}
}

func newConfigPathCommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the resolved config file path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := a.settingsPath()
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(a.out, path)
			return err
		},
	}
}
