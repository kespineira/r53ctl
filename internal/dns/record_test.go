package dns

import (
	"reflect"
	"testing"
)

func TestNormalizeRecordValues(t *testing.T) {
	tests := []struct {
		name       string
		recordType string
		values     []string
		want       []string
	}{
		{
			name:       "A",
			recordType: "a",
			values:     []string{"192.0.2.10"},
			want:       []string{"192.0.2.10"},
		},
		{
			name:       "CNAME",
			recordType: "CNAME",
			values:     []string{"Target.Example.COM"},
			want:       []string{"target.example.com."},
		},
		{
			name:       "MX",
			recordType: "MX",
			values:     []string{"10 mail.Example.COM"},
			want:       []string{"10 mail.example.com."},
		},
		{
			name:       "TXT quotes",
			recordType: "TXT",
			values:     []string{"v=spf1 include:example.com ~all"},
			want:       []string{`"v=spf1 include:example.com ~all"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeRecordValues(tt.recordType, tt.values)
			if err != nil {
				t.Fatalf("NormalizeRecordValues returned error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("NormalizeRecordValues() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestNormalizeRecordValuesRejectsInvalidARecord(t *testing.T) {
	if _, err := NormalizeRecordValues("A", []string{"not-an-ip"}); err == nil {
		t.Fatal("NormalizeRecordValues returned nil error")
	}
}

func TestNormalizeFilterTypeAllowsReadOnlyTypes(t *testing.T) {
	got, err := NormalizeFilterType("soa")
	if err != nil {
		t.Fatalf("NormalizeFilterType returned error: %v", err)
	}
	if got != "SOA" {
		t.Fatalf("NormalizeFilterType() = %q, want SOA", got)
	}
}
