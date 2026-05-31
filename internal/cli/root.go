package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	appconfig "github.com/kespineira/r53ctl/internal/config"
	r53 "github.com/kespineira/r53ctl/internal/route53"
	"github.com/kespineira/r53ctl/internal/settings"
)

type AWSFlags struct {
	Profile     string
	Region      string
	RoleARN     string
	EndpointURL string
	Output      string
}

type ServiceFactory func(context.Context, AWSFlags) (r53.Service, error)

type app struct {
	version        string
	awsFlags       AWSFlags
	configPath     string
	out            io.Writer
	errOut         io.Writer
	serviceFactory ServiceFactory
}

func NewRootCommand(version string) *cobra.Command {
	return newRootCommand(version, os.Stdout, os.Stderr, defaultServiceFactory)
}

func newRootCommand(version string, out io.Writer, errOut io.Writer, factory ServiceFactory) *cobra.Command {
	a := &app{
		version:        version,
		out:            out,
		errOut:         errOut,
		serviceFactory: factory,
	}

	cmd := &cobra.Command{
		Use:           "r53ctl",
		Short:         "Manage Amazon Route 53 hosted zones and records",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	cmd.PersistentFlags().StringVar(&a.awsFlags.Profile, "profile", "", "AWS shared config profile")
	cmd.PersistentFlags().StringVar(&a.awsFlags.Region, "region", "", "AWS region for SDK configuration")
	cmd.PersistentFlags().StringVar(&a.awsFlags.RoleARN, "role-arn", "", "AWS role ARN to assume before calling Route 53")
	cmd.PersistentFlags().StringVar(&a.awsFlags.EndpointURL, "endpoint-url", "", "custom Route 53 endpoint URL")
	cmd.PersistentFlags().StringVarP(&a.awsFlags.Output, "output", "o", "table", "output format: table or json")
	cmd.PersistentFlags().StringVar(&a.configPath, "config", "", "path to r53ctl config file")

	cmd.PersistentPreRunE = func(c *cobra.Command, _ []string) error {
		return a.applyConfigDefaults(c)
	}

	cmd.AddCommand(newZonesCommand(a))
	cmd.AddCommand(newRecordsCommand(a))
	return cmd
}

func defaultServiceFactory(ctx context.Context, flags AWSFlags) (r53.Service, error) {
	cfg, err := appconfig.LoadAWS(ctx, appconfig.AWSOptions{
		Profile: flags.Profile,
		Region:  flags.Region,
		RoleARN: flags.RoleARN,
	})
	if err != nil {
		return nil, err
	}
	return r53.NewAWSClient(cfg, flags.EndpointURL), nil
}

func (a *app) service(ctx context.Context) (r53.Service, error) {
	if a.awsFlags.Output != "table" && a.awsFlags.Output != "json" {
		return nil, fmt.Errorf("unsupported output format %q", a.awsFlags.Output)
	}
	return a.serviceFactory(ctx, a.awsFlags)
}

func requireYes(yes bool, action string) error {
	if yes {
		return nil
	}
	return fmt.Errorf("%s requires --yes", action)
}

func (a *app) settingsPath() (string, error) {
	if a.configPath != "" {
		return a.configPath, nil
	}
	return settings.DefaultPath()
}

// applyConfigDefaults fills any flag the user did not set explicitly from the
// config file, unless the corresponding AWS environment variable is set.
// Precedence: flag > environment variable > config file > built-in default.
func (a *app) applyConfigDefaults(cmd *cobra.Command) error {
	path, err := a.settingsPath()
	if err != nil {
		return err
	}
	s, err := settings.Load(path)
	if err != nil {
		return err
	}
	flags := cmd.Flags()
	if s.Profile != "" && !flags.Changed("profile") && os.Getenv("AWS_PROFILE") == "" {
		a.awsFlags.Profile = s.Profile
	}
	if s.Region != "" && !flags.Changed("region") &&
		os.Getenv("AWS_REGION") == "" && os.Getenv("AWS_DEFAULT_REGION") == "" {
		a.awsFlags.Region = s.Region
	}
	if s.Output != "" && !flags.Changed("output") {
		a.awsFlags.Output = s.Output
	}
	return nil
}
