package cli

import (
	"github.com/spf13/cobra"

	"github.com/kespineira/r53ctl/internal/dns"
	"github.com/kespineira/r53ctl/internal/domain"
	"github.com/kespineira/r53ctl/internal/output"
)

func newZonesCommand(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zones",
		Short: "Manage hosted zones",
	}
	cmd.AddCommand(newZonesListCommand(a))
	cmd.AddCommand(newZonesCreateCommand(a))
	cmd.AddCommand(newZonesDeleteCommand(a))
	return cmd
}

func newZonesListCommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List hosted zones",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := a.service(cmd.Context())
			if err != nil {
				return err
			}
			zones, err := svc.ListZones(cmd.Context())
			if err != nil {
				return err
			}
			if a.awsFlags.Output == "json" {
				return output.JSON(a.out, zones)
			}
			return output.ZonesTable(a.out, zones)
		},
	}
}

func newZonesCreateCommand(a *app) *cobra.Command {
	var comment string
	cmd := &cobra.Command{
		Use:   "create <domain>",
		Short: "Create a public hosted zone",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := dns.NormalizeName(args[0])
			if err != nil {
				return err
			}
			svc, err := a.service(cmd.Context())
			if err != nil {
				return err
			}
			zone, change, err := svc.CreateZone(cmd.Context(), name, comment)
			if err != nil {
				return err
			}
			if a.awsFlags.Output == "json" {
				return output.JSON(a.out, struct {
					Zone   domain.HostedZone   `json:"zone"`
					Change domain.ChangeResult `json:"change"`
				}{Zone: zone, Change: change})
			}
			if err := output.ZonesTable(a.out, []domain.HostedZone{zone}); err != nil {
				return err
			}
			return output.ChangesTable(a.out, []domain.ChangeResult{change})
		},
	}
	cmd.Flags().StringVar(&comment, "comment", "", "hosted zone comment")
	return cmd
}

func newZonesDeleteCommand(a *app) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <zone-id-or-name>",
		Short: "Delete an empty hosted zone",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireYes(yes, "zone deletion"); err != nil {
				return err
			}
			zoneRef := normalizeZoneRef(args[0])
			svc, err := a.service(cmd.Context())
			if err != nil {
				return err
			}
			change, err := svc.DeleteZone(cmd.Context(), zoneRef)
			if err != nil {
				return err
			}
			if a.awsFlags.Output == "json" {
				return output.JSON(a.out, change)
			}
			return output.ChangesTable(a.out, []domain.ChangeResult{change})
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm deletion")
	return cmd
}
