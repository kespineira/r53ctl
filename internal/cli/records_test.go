package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/kespineira/r53ctl/internal/domain"
	r53 "github.com/kespineira/r53ctl/internal/route53"
)

type fakeService struct {
	upsertZoneRef string
	upsertRecord  domain.RecordSet
}

func (f *fakeService) ListZones(context.Context) ([]domain.HostedZone, error) {
	return []domain.HostedZone{{ID: "Z123", Name: "example.com."}}, nil
}

func (f *fakeService) CreateZone(context.Context, string, string) (domain.HostedZone, domain.ChangeResult, error) {
	return domain.HostedZone{}, domain.ChangeResult{}, nil
}

func (f *fakeService) DeleteZone(context.Context, string) (domain.ChangeResult, error) {
	return domain.ChangeResult{ID: "C123", Status: "PENDING"}, nil
}

func (f *fakeService) ListRecords(context.Context, string, r53.RecordFilters) ([]domain.RecordSet, error) {
	ttl := int64(300)
	return []domain.RecordSet{{Name: "www.example.com.", Type: "A", TTL: &ttl, Values: []string{"192.0.2.1"}}}, nil
}

func (f *fakeService) UpsertRecord(_ context.Context, zoneRef string, record domain.RecordSet) (domain.ChangeResult, error) {
	f.upsertZoneRef = zoneRef
	f.upsertRecord = record
	return domain.ChangeResult{ID: "C123", Status: "PENDING"}, nil
}

func (f *fakeService) DeleteRecord(context.Context, string, string, string) (domain.ChangeResult, error) {
	return domain.ChangeResult{ID: "C123", Status: "PENDING"}, nil
}

func TestRecordsUpsertBuildsRecord(t *testing.T) {
	fake := &fakeService{}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := newRootCommand("test", &stdout, &stderr, func(context.Context, AWSFlags) (r53.Service, error) {
		return fake, nil
	})
	cmd.SetArgs([]string{
		"--output", "json",
		"records", "upsert", "example.com",
		"--name", "www.example.com",
		"--type", "A",
		"--ttl", "60",
		"--value", "192.0.2.10",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if fake.upsertZoneRef != "example.com." {
		t.Fatalf("zone ref = %q, want example.com.", fake.upsertZoneRef)
	}
	if fake.upsertRecord.Name != "www.example.com." {
		t.Fatalf("record name = %q, want www.example.com.", fake.upsertRecord.Name)
	}
	if fake.upsertRecord.Type != "A" {
		t.Fatalf("record type = %q, want A", fake.upsertRecord.Type)
	}
	if fake.upsertRecord.TTL == nil || *fake.upsertRecord.TTL != 60 {
		t.Fatalf("record ttl = %#v, want 60", fake.upsertRecord.TTL)
	}
	if len(fake.upsertRecord.Values) != 1 || fake.upsertRecord.Values[0] != "192.0.2.10" {
		t.Fatalf("record values = %#v, want 192.0.2.10", fake.upsertRecord.Values)
	}
}

func TestRecordsDeleteRequiresYes(t *testing.T) {
	fake := &fakeService{}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := newRootCommand("test", &stdout, &stderr, func(context.Context, AWSFlags) (r53.Service, error) {
		return fake, nil
	})
	cmd.SetArgs([]string{
		"records", "delete", "Z123",
		"--name", "www.example.com",
		"--type", "A",
	})

	if err := cmd.Execute(); err == nil {
		t.Fatal("Execute returned nil error")
	}
}
