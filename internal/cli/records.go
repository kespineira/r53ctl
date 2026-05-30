package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kespineira/r53ctl/internal/dns"
	"github.com/kespineira/r53ctl/internal/domain"
	"github.com/kespineira/r53ctl/internal/output"
	r53 "github.com/kespineira/r53ctl/internal/route53"
)

func newRecordsCommand(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "records",
		Short: "Manage resource record sets",
	}
	cmd.AddCommand(newRecordsListCommand(a))
	cmd.AddCommand(newRecordsUpsertCommand(a))
	cmd.AddCommand(newRecordsDeleteCommand(a))
	cmd.AddCommand(newRecordsExportCommand(a))
	return cmd
}

func newRecordsListCommand(a *app) *cobra.Command {
	var name string
	var recordType string
	cmd := &cobra.Command{
		Use:   "list <zone-id-or-name>",
		Short: "List records in a hosted zone",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filters, err := normalizeFilters(name, recordType)
			if err != nil {
				return err
			}
			svc, err := a.service(cmd.Context())
			if err != nil {
				return err
			}
			records, err := svc.ListRecords(cmd.Context(), normalizeZoneRef(args[0]), filters)
			if err != nil {
				return err
			}
			if a.awsFlags.Output == "json" {
				return output.JSON(a.out, records)
			}
			return output.RecordsTable(a.out, records)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "filter by fully qualified record name")
	cmd.Flags().StringVar(&recordType, "type", "", "filter by record type")
	return cmd
}

func newRecordsUpsertCommand(a *app) *cobra.Command {
	var name string
	var recordType string
	var ttl int64
	var values []string

	cmd := &cobra.Command{
		Use:   "upsert <zone-id-or-name>",
		Short: "Create or update a basic record set",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			recordName, err := dns.NormalizeName(name)
			if err != nil {
				return err
			}
			recordType, err := dns.NormalizeType(recordType)
			if err != nil {
				return err
			}
			ttlPtr, err := dns.NormalizeTTL(ttl)
			if err != nil {
				return err
			}
			values, err := dns.NormalizeRecordValues(recordType, values)
			if err != nil {
				return err
			}

			record := domain.RecordSet{
				Name:   recordName,
				Type:   recordType,
				TTL:    ttlPtr,
				Values: values,
			}

			svc, err := a.service(cmd.Context())
			if err != nil {
				return err
			}
			change, err := svc.UpsertRecord(cmd.Context(), normalizeZoneRef(args[0]), record)
			if err != nil {
				return err
			}
			if a.awsFlags.Output == "json" {
				return output.JSON(a.out, change)
			}
			return output.ChangesTable(a.out, []domain.ChangeResult{change})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "record name")
	cmd.Flags().StringVar(&recordType, "type", "", "record type")
	cmd.Flags().Int64Var(&ttl, "ttl", 300, "record TTL in seconds")
	cmd.Flags().StringArrayVar(&values, "value", nil, "record value; repeat for multiple values")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("type")
	_ = cmd.MarkFlagRequired("value")
	return cmd
}

func newRecordsDeleteCommand(a *app) *cobra.Command {
	var name string
	var recordType string
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <zone-id-or-name>",
		Short: "Delete a record set",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireYes(yes, "record deletion"); err != nil {
				return err
			}
			recordName, err := dns.NormalizeName(name)
			if err != nil {
				return err
			}
			recordType, err := dns.NormalizeType(recordType)
			if err != nil {
				return err
			}
			svc, err := a.service(cmd.Context())
			if err != nil {
				return err
			}
			change, err := svc.DeleteRecord(cmd.Context(), normalizeZoneRef(args[0]), recordName, recordType)
			if err != nil {
				return err
			}
			if a.awsFlags.Output == "json" {
				return output.JSON(a.out, change)
			}
			return output.ChangesTable(a.out, []domain.ChangeResult{change})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "record name")
	cmd.Flags().StringVar(&recordType, "type", "", "record type")
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm deletion")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("type")
	return cmd
}

func newRecordsExportCommand(a *app) *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "export <zone-id-or-name>",
		Short: "Export records from a hosted zone",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := a.service(cmd.Context())
			if err != nil {
				return err
			}
			records, err := svc.ListRecords(cmd.Context(), normalizeZoneRef(args[0]), r53.RecordFilters{})
			if err != nil {
				return err
			}
			switch strings.ToLower(format) {
			case "json":
				return output.JSON(a.out, records)
			case "bind":
				data, err := dns.ExportBIND(records)
				if err != nil {
					return err
				}
				_, err = a.out.Write(data)
				return err
			default:
				return fmt.Errorf("unsupported export format %q", format)
			}
		},
	}
	cmd.Flags().StringVar(&format, "format", "bind", "export format: bind or json")
	return cmd
}

func normalizeFilters(name string, recordType string) (r53.RecordFilters, error) {
	filters := r53.RecordFilters{}
	if strings.TrimSpace(name) != "" {
		normalizedName, err := dns.NormalizeName(name)
		if err != nil {
			return filters, err
		}
		filters.Name = normalizedName
	}
	if strings.TrimSpace(recordType) != "" {
		normalizedType, err := dns.NormalizeFilterType(recordType)
		if err != nil {
			return filters, err
		}
		filters.Type = normalizedType
	}
	return filters, nil
}
