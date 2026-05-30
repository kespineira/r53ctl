package cli

import (
	"strings"

	"github.com/kespineira/r53ctl/internal/dns"
)

func normalizeZoneRef(ref string) string {
	trimmed := strings.TrimSpace(ref)
	withoutPrefix := strings.TrimPrefix(trimmed, "/hostedzone/")
	withoutPrefix = strings.TrimPrefix(withoutPrefix, "hostedzone/")
	if strings.HasPrefix(strings.ToUpper(withoutPrefix), "Z") && !strings.Contains(withoutPrefix, ".") {
		return trimmed
	}
	name, err := dns.NormalizeName(trimmed)
	if err != nil {
		return trimmed
	}
	return name
}
