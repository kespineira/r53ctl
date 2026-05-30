package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/kespineira/r53ctl/internal/domain"
)

func JSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func ZonesTable(w io.Writer, zones []domain.HostedZone) error {
	rows := [][]string{{"ID", "NAME", "PRIVATE", "RECORDS", "COMMENT"}}
	for _, zone := range zones {
		rows = append(rows, []string{
			zone.ID,
			zone.Name,
			fmt.Sprintf("%t", zone.Private),
			fmt.Sprintf("%d", zone.ResourceRecordSetCount),
			zone.Comment,
		})
	}
	return table(w, rows)
}

func RecordsTable(w io.Writer, records []domain.RecordSet) error {
	rows := [][]string{{"NAME", "TYPE", "TTL", "VALUE"}}
	for _, record := range records {
		ttl := ""
		if record.TTL != nil {
			ttl = fmt.Sprintf("%d", *record.TTL)
		}
		value := strings.Join(record.Values, ", ")
		if record.Alias != nil {
			value = "alias -> " + record.Alias.DNSName
		}
		rows = append(rows, []string{record.Name, record.Type, ttl, value})
	}
	return table(w, rows)
}

func ChangesTable(w io.Writer, changes []domain.ChangeResult) error {
	rows := [][]string{{"ID", "STATUS", "SUBMITTED_AT"}}
	for _, change := range changes {
		rows = append(rows, []string{change.ID, change.Status, change.SubmittedAt})
	}
	return table(w, rows)
}

func table(w io.Writer, rows [][]string) error {
	if len(rows) == 0 {
		return nil
	}
	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				if _, err := fmt.Fprint(w, "  "); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(w, "%-*s", widths[i], cell); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	return nil
}
