package dns

import (
	"testing"

	"github.com/kespineira/r53ctl/internal/domain"
)

func TestExportBIND(t *testing.T) {
	ttl := int64(300)
	got, err := ExportBIND([]domain.RecordSet{
		{Name: "www.example.com.", Type: "A", TTL: &ttl, Values: []string{"192.0.2.1"}},
		{Name: "txt.example.com.", Type: "TXT", TTL: &ttl, Values: []string{`"hello"`}},
	})
	if err != nil {
		t.Fatalf("ExportBIND returned error: %v", err)
	}
	want := "txt.example.com. 300 IN TXT \"hello\"\nwww.example.com. 300 IN A 192.0.2.1\n"
	if string(got) != want {
		t.Fatalf("ExportBIND() = %q, want %q", string(got), want)
	}
}
