package dns

import (
	"fmt"
	"strings"
)

func NormalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("name is required")
	}
	if !strings.HasSuffix(name, ".") {
		name += "."
	}
	name = strings.ToLower(name)
	if len(name) > 253 {
		return "", fmt.Errorf("dns name %q is longer than 253 characters", name)
	}

	labels := strings.Split(strings.TrimSuffix(name, "."), ".")
	for i, label := range labels {
		if label == "" {
			return "", fmt.Errorf("dns name %q contains an empty label", name)
		}
		if len(label) > 63 {
			return "", fmt.Errorf("dns label %q is longer than 63 characters", label)
		}
		if label == "*" {
			if i != 0 {
				return "", fmt.Errorf("wildcard label must be the first label in %q", name)
			}
			continue
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return "", fmt.Errorf("dns label %q cannot start or end with a hyphen", label)
		}
		for _, r := range label {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
				continue
			}
			return "", fmt.Errorf("dns label %q contains unsupported character %q", label, r)
		}
	}

	return name, nil
}
