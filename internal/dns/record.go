package dns

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

var supportedTypes = map[string]struct{}{
	"A":     {},
	"AAAA":  {},
	"CAA":   {},
	"CNAME": {},
	"MX":    {},
	"NS":    {},
	"SRV":   {},
	"TXT":   {},
}

func NormalizeType(recordType string) (string, error) {
	recordType = strings.ToUpper(strings.TrimSpace(recordType))
	if recordType == "" {
		return "", fmt.Errorf("record type is required")
	}
	if _, ok := supportedTypes[recordType]; !ok {
		return "", fmt.Errorf("record type %q is not supported in this MVP", recordType)
	}
	return recordType, nil
}

func NormalizeFilterType(recordType string) (string, error) {
	recordType = strings.ToUpper(strings.TrimSpace(recordType))
	if recordType == "" {
		return "", fmt.Errorf("record type is required")
	}
	for _, r := range recordType {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		return "", fmt.Errorf("record type %q contains unsupported character %q", recordType, r)
	}
	return recordType, nil
}

func NormalizeTTL(ttl int64) (*int64, error) {
	if ttl <= 0 {
		return nil, fmt.Errorf("ttl must be greater than zero")
	}
	return &ttl, nil
}

func NormalizeRecordValues(recordType string, values []string) ([]string, error) {
	recordType, err := NormalizeType(recordType)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("at least one --value is required")
	}

	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("record values cannot be empty")
		}

		switch recordType {
		case "A":
			ip := net.ParseIP(value)
			if ip == nil || ip.To4() == nil {
				return nil, fmt.Errorf("A record value %q is not an IPv4 address", value)
			}
			normalized = append(normalized, value)
		case "AAAA":
			ip := net.ParseIP(value)
			if ip == nil || ip.To4() != nil {
				return nil, fmt.Errorf("AAAA record value %q is not an IPv6 address", value)
			}
			normalized = append(normalized, value)
		case "CNAME", "NS":
			name, err := NormalizeName(value)
			if err != nil {
				return nil, fmt.Errorf("%s record value %q is invalid: %w", recordType, value, err)
			}
			normalized = append(normalized, name)
		case "MX":
			value, err := normalizeMX(value)
			if err != nil {
				return nil, err
			}
			normalized = append(normalized, value)
		case "SRV":
			value, err := normalizeSRV(value)
			if err != nil {
				return nil, err
			}
			normalized = append(normalized, value)
		case "CAA":
			normalizedValue, err := normalizeCAA(value)
			if err != nil {
				return nil, err
			}
			normalized = append(normalized, normalizedValue)
		case "TXT":
			normalized = append(normalized, quoteTXT(value))
		}
	}

	return normalized, nil
}

func normalizeMX(value string) (string, error) {
	parts := strings.Fields(value)
	if len(parts) != 2 {
		return "", fmt.Errorf("MX record value %q must be '<priority> <exchange>'", value)
	}
	if _, err := strconv.Atoi(parts[0]); err != nil {
		return "", fmt.Errorf("MX priority %q is not an integer", parts[0])
	}
	exchange, err := NormalizeName(parts[1])
	if err != nil {
		return "", fmt.Errorf("MX exchange %q is invalid: %w", parts[1], err)
	}
	return parts[0] + " " + exchange, nil
}

func normalizeSRV(value string) (string, error) {
	parts := strings.Fields(value)
	if len(parts) != 4 {
		return "", fmt.Errorf("SRV record value %q must be '<priority> <weight> <port> <target>'", value)
	}
	for i, label := range []string{"priority", "weight", "port"} {
		if _, err := strconv.Atoi(parts[i]); err != nil {
			return "", fmt.Errorf("SRV %s %q is not an integer", label, parts[i])
		}
	}
	target, err := NormalizeName(parts[3])
	if err != nil {
		return "", fmt.Errorf("SRV target %q is invalid: %w", parts[3], err)
	}
	return strings.Join([]string{parts[0], parts[1], parts[2], target}, " "), nil
}

func normalizeCAA(value string) (string, error) {
	parts := strings.Fields(value)
	if len(parts) < 3 {
		return "", fmt.Errorf("CAA record value %q must be '<flag> <tag> <value>'", value)
	}
	if flag, err := strconv.Atoi(parts[0]); err != nil || flag < 0 || flag > 255 {
		return "", fmt.Errorf("CAA flag %q must be an integer between 0 and 255", parts[0])
	}
	caaValue := strings.Join(parts[2:], " ")
	return parts[0] + " " + parts[1] + " " + quoteString(caaValue), nil
}

// quoteString wraps a single value in double quotes, escaping backslashes and
// quotes. A value already wrapped in quotes is returned unchanged.
func quoteString(value string) string {
	if len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		return value
	}
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

// quoteTXT formats a TXT value as one or more quoted character-strings. DNS
// limits each character-string to 255 bytes, so longer values are split into
// 255-byte chunks joined by spaces. A pre-quoted value is returned unchanged.
func quoteTXT(value string) string {
	if len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		return value
	}
	const maxChunk = 255
	chunks := make([]string, 0, len(value)/maxChunk+1)
	for len(value) > maxChunk {
		chunks = append(chunks, quoteString(value[:maxChunk]))
		value = value[maxChunk:]
	}
	chunks = append(chunks, quoteString(value))
	return strings.Join(chunks, " ")
}
