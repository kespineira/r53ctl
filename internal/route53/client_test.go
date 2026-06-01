package route53

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsroute53 "github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

type fakeAPI struct {
	pages       []*awsroute53.ListResourceRecordSetsOutput
	listInputs  []*awsroute53.ListResourceRecordSetsInput
	changeInput *awsroute53.ChangeResourceRecordSetsInput
}

func (f *fakeAPI) ListResourceRecordSets(_ context.Context, in *awsroute53.ListResourceRecordSetsInput, _ ...func(*awsroute53.Options)) (*awsroute53.ListResourceRecordSetsOutput, error) {
	idx := len(f.listInputs)
	f.listInputs = append(f.listInputs, in)
	if idx < len(f.pages) {
		return f.pages[idx], nil
	}
	return &awsroute53.ListResourceRecordSetsOutput{}, nil
}

func (f *fakeAPI) ChangeResourceRecordSets(_ context.Context, in *awsroute53.ChangeResourceRecordSetsInput, _ ...func(*awsroute53.Options)) (*awsroute53.ChangeResourceRecordSetsOutput, error) {
	f.changeInput = in
	return &awsroute53.ChangeResourceRecordSetsOutput{
		ChangeInfo: &types.ChangeInfo{Id: aws.String("/change/C1"), Status: types.ChangeStatusPending},
	}, nil
}

func (f *fakeAPI) CreateHostedZone(context.Context, *awsroute53.CreateHostedZoneInput, ...func(*awsroute53.Options)) (*awsroute53.CreateHostedZoneOutput, error) {
	return &awsroute53.CreateHostedZoneOutput{}, nil
}

func (f *fakeAPI) DeleteHostedZone(context.Context, *awsroute53.DeleteHostedZoneInput, ...func(*awsroute53.Options)) (*awsroute53.DeleteHostedZoneOutput, error) {
	return &awsroute53.DeleteHostedZoneOutput{}, nil
}

func (f *fakeAPI) ListHostedZones(context.Context, *awsroute53.ListHostedZonesInput, ...func(*awsroute53.Options)) (*awsroute53.ListHostedZonesOutput, error) {
	return &awsroute53.ListHostedZonesOutput{}, nil
}

func rrset(name, rtype string, ttl int64, value string) types.ResourceRecordSet {
	return types.ResourceRecordSet{
		Name:            aws.String(name),
		Type:            types.RRType(rtype),
		TTL:             aws.Int64(ttl),
		ResourceRecords: []types.ResourceRecord{{Value: aws.String(value)}},
	}
}

func TestListRecordsStartsAtFilteredName(t *testing.T) {
	fake := &fakeAPI{pages: []*awsroute53.ListResourceRecordSetsOutput{{
		ResourceRecordSets: []types.ResourceRecordSet{
			rrset("www.example.com.", "A", 300, "192.0.2.1"),
			rrset("www.example.com.", "AAAA", 300, "2001:db8::1"),
			rrset("zzz.example.com.", "A", 300, "192.0.2.9"),
		},
	}}}
	client := New(fake)

	got, err := client.ListRecords(context.Background(), "Z123", RecordFilters{Name: "www.example.com."})
	if err != nil {
		t.Fatalf("ListRecords error: %v", err)
	}
	if len(fake.listInputs) != 1 {
		t.Fatalf("expected 1 list call, got %d", len(fake.listInputs))
	}
	if aws.ToString(fake.listInputs[0].StartRecordName) != "www.example.com." {
		t.Fatalf("StartRecordName = %q, want www.example.com.", aws.ToString(fake.listInputs[0].StartRecordName))
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 matching records, got %d: %#v", len(got), got)
	}
}

func TestListRecordsStopsPaginatingPastFilteredName(t *testing.T) {
	fake := &fakeAPI{pages: []*awsroute53.ListResourceRecordSetsOutput{
		{
			IsTruncated:    true,
			NextRecordName: aws.String("zzz.example.com."),
			NextRecordType: types.RRTypeA,
			ResourceRecordSets: []types.ResourceRecordSet{
				rrset("www.example.com.", "A", 300, "192.0.2.1"),
				rrset("zzz.example.com.", "A", 300, "192.0.2.9"),
			},
		},
		{ // must never be fetched once we pass the filtered name
			ResourceRecordSets: []types.ResourceRecordSet{
				rrset("zzz.example.com.", "A", 300, "192.0.2.9"),
			},
		},
	}}
	client := New(fake)

	got, err := client.ListRecords(context.Background(), "Z123", RecordFilters{Name: "www.example.com."})
	if err != nil {
		t.Fatalf("ListRecords error: %v", err)
	}
	if len(fake.listInputs) != 1 {
		t.Fatalf("expected to stop after 1 page, got %d calls", len(fake.listInputs))
	}
	if len(got) != 1 || got[0].Name != "www.example.com." {
		t.Fatalf("got %#v, want only www.example.com.", got)
	}
}

func TestDeleteRecordDeletesMatchingRecord(t *testing.T) {
	fake := &fakeAPI{pages: []*awsroute53.ListResourceRecordSetsOutput{{
		ResourceRecordSets: []types.ResourceRecordSet{
			rrset("www.example.com.", "A", 300, "192.0.2.1"),
			rrset("zzz.example.com.", "A", 300, "192.0.2.9"),
		},
	}}}
	client := New(fake)

	if _, err := client.DeleteRecord(context.Background(), "Z123", "www.example.com.", "A"); err != nil {
		t.Fatalf("DeleteRecord error: %v", err)
	}
	if aws.ToString(fake.listInputs[0].StartRecordName) != "www.example.com." {
		t.Fatalf("findAWSRecordSets should start at the name, got %q", aws.ToString(fake.listInputs[0].StartRecordName))
	}
	if fake.changeInput == nil {
		t.Fatal("expected a ChangeResourceRecordSets call")
	}
	change := fake.changeInput.ChangeBatch.Changes[0]
	if change.Action != types.ChangeActionDelete {
		t.Fatalf("action = %v, want DELETE", change.Action)
	}
	if aws.ToString(change.ResourceRecordSet.Name) != "www.example.com." {
		t.Fatalf("deleted record name = %q, want www.example.com.", aws.ToString(change.ResourceRecordSet.Name))
	}
}
