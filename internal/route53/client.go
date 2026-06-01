package route53

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsroute53 "github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"

	"github.com/kespineira/r53ctl/internal/domain"
)

type Service interface {
	ListZones(ctx context.Context) ([]domain.HostedZone, error)
	CreateZone(ctx context.Context, name string, comment string) (domain.HostedZone, domain.ChangeResult, error)
	DeleteZone(ctx context.Context, zoneRef string) (domain.ChangeResult, error)
	ListRecords(ctx context.Context, zoneRef string, filters RecordFilters) ([]domain.RecordSet, error)
	UpsertRecord(ctx context.Context, zoneRef string, record domain.RecordSet) (domain.ChangeResult, error)
	DeleteRecord(ctx context.Context, zoneRef string, name string, recordType string) (domain.ChangeResult, error)
}

type RecordFilters struct {
	Name string
	Type string
}

type API interface {
	ChangeResourceRecordSets(context.Context, *awsroute53.ChangeResourceRecordSetsInput, ...func(*awsroute53.Options)) (*awsroute53.ChangeResourceRecordSetsOutput, error)
	CreateHostedZone(context.Context, *awsroute53.CreateHostedZoneInput, ...func(*awsroute53.Options)) (*awsroute53.CreateHostedZoneOutput, error)
	DeleteHostedZone(context.Context, *awsroute53.DeleteHostedZoneInput, ...func(*awsroute53.Options)) (*awsroute53.DeleteHostedZoneOutput, error)
	ListHostedZones(context.Context, *awsroute53.ListHostedZonesInput, ...func(*awsroute53.Options)) (*awsroute53.ListHostedZonesOutput, error)
	ListResourceRecordSets(context.Context, *awsroute53.ListResourceRecordSetsInput, ...func(*awsroute53.Options)) (*awsroute53.ListResourceRecordSetsOutput, error)
}

type Client struct {
	api API
}

func New(api API) *Client {
	return &Client{api: api}
}

func NewAWSClient(cfg aws.Config, endpointURL string) *Client {
	api := awsroute53.NewFromConfig(cfg, func(options *awsroute53.Options) {
		if endpointURL != "" {
			options.BaseEndpoint = aws.String(endpointURL)
		}
	})
	return New(api)
}

func (c *Client) ListZones(ctx context.Context) ([]domain.HostedZone, error) {
	paginator := awsroute53.NewListHostedZonesPaginator(c.api, &awsroute53.ListHostedZonesInput{})
	zones := []domain.HostedZone{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, hostedZone := range page.HostedZones {
			zones = append(zones, convertHostedZone(hostedZone))
		}
	}
	return zones, nil
}

func (c *Client) CreateZone(ctx context.Context, name string, comment string) (domain.HostedZone, domain.ChangeResult, error) {
	input := &awsroute53.CreateHostedZoneInput{
		CallerReference: aws.String(fmt.Sprintf("route53-cli-%d", time.Now().UnixNano())),
		Name:            aws.String(name),
	}
	if comment != "" {
		input.HostedZoneConfig = &types.HostedZoneConfig{Comment: aws.String(comment)}
	}

	out, err := c.api.CreateHostedZone(ctx, input)
	if err != nil {
		return domain.HostedZone{}, domain.ChangeResult{}, err
	}
	if out.HostedZone == nil {
		return domain.HostedZone{}, convertChange(out.ChangeInfo), nil
	}
	return convertHostedZone(*out.HostedZone), convertChange(out.ChangeInfo), nil
}

func (c *Client) DeleteZone(ctx context.Context, zoneRef string) (domain.ChangeResult, error) {
	zoneID, err := c.resolveZoneID(ctx, zoneRef)
	if err != nil {
		return domain.ChangeResult{}, err
	}
	out, err := c.api.DeleteHostedZone(ctx, &awsroute53.DeleteHostedZoneInput{Id: aws.String(zoneID)})
	if err != nil {
		return domain.ChangeResult{}, err
	}
	return convertChange(out.ChangeInfo), nil
}

func (c *Client) ListRecords(ctx context.Context, zoneRef string, filters RecordFilters) ([]domain.RecordSet, error) {
	zoneID, err := c.resolveZoneID(ctx, zoneRef)
	if err != nil {
		return nil, err
	}

	input := &awsroute53.ListResourceRecordSetsInput{HostedZoneId: aws.String(zoneID)}
	// When filtering by an exact name, ask Route 53 to start listing at that
	// name. Results are sorted with identical names grouped together, so we can
	// stop as soon as we move past the filtered name instead of scanning the
	// whole zone.
	if filters.Name != "" {
		input.StartRecordName = aws.String(filters.Name)
		if filters.Type != "" {
			input.StartRecordType = types.RRType(filters.Type)
		}
	}
	paginator := awsroute53.NewListResourceRecordSetsPaginator(c.api, input)

	records := []domain.RecordSet{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, record := range page.ResourceRecordSets {
			converted := convertRecordSet(record)
			if filters.Name != "" && converted.Name != filters.Name {
				return records, nil
			}
			if filters.Type != "" && converted.Type != filters.Type {
				continue
			}
			records = append(records, converted)
		}
	}
	return records, nil
}

func (c *Client) UpsertRecord(ctx context.Context, zoneRef string, record domain.RecordSet) (domain.ChangeResult, error) {
	zoneID, err := c.resolveZoneID(ctx, zoneRef)
	if err != nil {
		return domain.ChangeResult{}, err
	}
	if record.TTL == nil {
		return domain.ChangeResult{}, errors.New("ttl is required")
	}
	if record.Alias != nil {
		return domain.ChangeResult{}, errors.New("alias upserts are not supported in this MVP")
	}

	out, err := c.api.ChangeResourceRecordSets(ctx, &awsroute53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{{
				Action: types.ChangeActionUpsert,
				ResourceRecordSet: &types.ResourceRecordSet{
					Name:            aws.String(record.Name),
					Type:            types.RRType(record.Type),
					TTL:             record.TTL,
					ResourceRecords: toAWSResourceRecords(record.Values),
				},
			}},
		},
	})
	if err != nil {
		return domain.ChangeResult{}, err
	}
	return convertChange(out.ChangeInfo), nil
}

func (c *Client) DeleteRecord(ctx context.Context, zoneRef string, name string, recordType string) (domain.ChangeResult, error) {
	zoneID, err := c.resolveZoneID(ctx, zoneRef)
	if err != nil {
		return domain.ChangeResult{}, err
	}

	matches, err := c.findAWSRecordSets(ctx, zoneID, name, recordType)
	if err != nil {
		return domain.ChangeResult{}, err
	}
	if len(matches) == 0 {
		return domain.ChangeResult{}, fmt.Errorf("record %s %s was not found", name, recordType)
	}
	if len(matches) > 1 {
		return domain.ChangeResult{}, fmt.Errorf("record %s %s has multiple routing policy variants; delete by set identifier is not supported in this MVP", name, recordType)
	}

	record := matches[0]
	if record.SetIdentifier != nil {
		return domain.ChangeResult{}, fmt.Errorf("record %s %s uses a routing policy; delete by set identifier is not supported in this MVP", name, recordType)
	}

	out, err := c.api.ChangeResourceRecordSets(ctx, &awsroute53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{{
				Action:            types.ChangeActionDelete,
				ResourceRecordSet: &record,
			}},
		},
	})
	if err != nil {
		return domain.ChangeResult{}, err
	}
	return convertChange(out.ChangeInfo), nil
}

func (c *Client) findAWSRecordSets(ctx context.Context, zoneID string, name string, recordType string) ([]types.ResourceRecordSet, error) {
	input := &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(zoneID),
		StartRecordName: aws.String(name),
		StartRecordType: types.RRType(recordType),
	}
	paginator := awsroute53.NewListResourceRecordSetsPaginator(c.api, input)
	matches := []types.ResourceRecordSet{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, record := range page.ResourceRecordSets {
			if aws.ToString(record.Name) != name {
				return matches, nil
			}
			if string(record.Type) == recordType {
				matches = append(matches, record)
			}
		}
	}
	return matches, nil
}

func (c *Client) resolveZoneID(ctx context.Context, zoneRef string) (string, error) {
	if zoneID, ok := cleanZoneID(zoneRef); ok {
		return zoneID, nil
	}

	zones, err := c.ListZones(ctx)
	if err != nil {
		return "", err
	}

	matches := []domain.HostedZone{}
	for _, zone := range zones {
		if zone.Name == zoneRef {
			matches = append(matches, zone)
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("hosted zone %q was not found", zoneRef)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("hosted zone %q is ambiguous; use a zone id", zoneRef)
	}
	return matches[0].ID, nil
}

func cleanZoneID(zoneRef string) (string, bool) {
	zoneRef = strings.TrimSpace(zoneRef)
	zoneRef = strings.TrimPrefix(zoneRef, "/hostedzone/")
	zoneRef = strings.TrimPrefix(zoneRef, "hostedzone/")
	if zoneRef == "" || strings.Contains(zoneRef, ".") {
		return "", false
	}
	if !strings.HasPrefix(strings.ToUpper(zoneRef), "Z") {
		return "", false
	}
	return zoneRef, true
}

func convertHostedZone(zone types.HostedZone) domain.HostedZone {
	comment := ""
	private := false
	if zone.Config != nil {
		comment = aws.ToString(zone.Config.Comment)
		private = zone.Config.PrivateZone
	}
	return domain.HostedZone{
		ID:                     cleanRoute53ID(aws.ToString(zone.Id)),
		Name:                   aws.ToString(zone.Name),
		CallerReference:        aws.ToString(zone.CallerReference),
		Comment:                comment,
		Private:                private,
		ResourceRecordSetCount: aws.ToInt64(zone.ResourceRecordSetCount),
	}
}

func convertRecordSet(record types.ResourceRecordSet) domain.RecordSet {
	converted := domain.RecordSet{
		Name:          aws.ToString(record.Name),
		Type:          string(record.Type),
		TTL:           record.TTL,
		SetIdentifier: aws.ToString(record.SetIdentifier),
	}
	for _, value := range record.ResourceRecords {
		converted.Values = append(converted.Values, aws.ToString(value.Value))
	}
	if record.AliasTarget != nil {
		converted.Alias = &domain.AliasTarget{
			HostedZoneID:         aws.ToString(record.AliasTarget.HostedZoneId),
			DNSName:              aws.ToString(record.AliasTarget.DNSName),
			EvaluateTargetHealth: record.AliasTarget.EvaluateTargetHealth,
		}
	}
	return converted
}

func convertChange(change *types.ChangeInfo) domain.ChangeResult {
	if change == nil {
		return domain.ChangeResult{}
	}
	submittedAt := ""
	if change.SubmittedAt != nil {
		submittedAt = change.SubmittedAt.Format(time.RFC3339)
	}
	return domain.ChangeResult{
		ID:          cleanRoute53ID(aws.ToString(change.Id)),
		Status:      string(change.Status),
		SubmittedAt: submittedAt,
	}
}

func toAWSResourceRecords(values []string) []types.ResourceRecord {
	records := make([]types.ResourceRecord, 0, len(values))
	for _, value := range values {
		records = append(records, types.ResourceRecord{Value: aws.String(value)})
	}
	return records
}

func cleanRoute53ID(id string) string {
	id = strings.TrimPrefix(id, "/hostedzone/")
	id = strings.TrimPrefix(id, "/change/")
	return id
}
