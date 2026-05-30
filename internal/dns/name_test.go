package dns

import "testing"

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "adds trailing dot", in: "Example.COM", want: "example.com."},
		{name: "keeps wildcard", in: "*.Example.COM", want: "*.example.com."},
		{name: "allows service labels", in: "_sip._tcp.example.com", want: "_sip._tcp.example.com."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeName(tt.in)
			if err != nil {
				t.Fatalf("NormalizeName returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeNameRejectsInvalidNames(t *testing.T) {
	for _, in := range []string{"", "example..com", "bad label.example.com", "www.*.example.com"} {
		t.Run(in, func(t *testing.T) {
			if _, err := NormalizeName(in); err == nil {
				t.Fatalf("NormalizeName(%q) returned nil error", in)
			}
		})
	}
}
