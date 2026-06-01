package dns

import (
	"reflect"
	"strings"
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

func TestNormalizeRecordValuesQuotesCAA(t *testing.T) {
	got, err := NormalizeRecordValues("CAA", []string{"0 issue letsencrypt.org"})
	if err != nil {
		t.Fatalf("NormalizeRecordValues returned error: %v", err)
	}
	want := []string{`0 issue "letsencrypt.org"`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeRecordValues() = %#v, want %#v", got, want)
	}
}

func TestNormalizeRecordValuesChunksLongTXT(t *testing.T) {
	long := strings.Repeat("a", 300)
	got, err := NormalizeRecordValues("TXT", []string{long})
	if err != nil {
		t.Fatalf("NormalizeRecordValues returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 RDATA value, got %d", len(got))
	}
	want := `"` + strings.Repeat("a", 255) + `" "` + strings.Repeat("a", 45) + `"`
	if got[0] != want {
		t.Fatalf("NormalizeRecordValues() = %q, want %q", got[0], want)
	}
}

func TestNormalizeRecordValuesKeepsQuotedCAA(t *testing.T) {
	got, err := NormalizeRecordValues("CAA", []string{`0 issue "letsencrypt.org"`})
	if err != nil {
		t.Fatalf("NormalizeRecordValues returned error: %v", err)
	}
	want := []string{`0 issue "letsencrypt.org"`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeRecordValues() = %#v, want %#v", got, want)
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
