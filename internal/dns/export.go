package dns

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/kespineira/r53ctl/internal/domain"
)

func ExportBIND(records []domain.RecordSet) ([]byte, error) {
	records = append([]domain.RecordSet(nil), records...)
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].Name == records[j].Name {
			return records[i].Type < records[j].Type
		}
		return records[i].Name < records[j].Name
	})

	var buf bytes.Buffer
	for _, record := range records {
		if record.Alias != nil {
			fmt.Fprintf(&buf, "; omitted alias %s %s -> %s\n", record.Name, record.Type, record.Alias.DNSName)
			continue
		}
		ttl := int64(0)
		if record.TTL != nil {
			ttl = *record.TTL
		}
		for _, value := range record.Values {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			fmt.Fprintf(&buf, "%s %d IN %s %s\n", record.Name, ttl, record.Type, value)
		}
	}
	return buf.Bytes(), nil
}
