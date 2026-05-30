package domain

type HostedZone struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	CallerReference        string `json:"caller_reference,omitempty"`
	Comment                string `json:"comment,omitempty"`
	Private                bool   `json:"private"`
	ResourceRecordSetCount int64  `json:"resource_record_set_count"`
}

type AliasTarget struct {
	HostedZoneID         string `json:"hosted_zone_id"`
	DNSName              string `json:"dns_name"`
	EvaluateTargetHealth bool   `json:"evaluate_target_health"`
}

type RecordSet struct {
	Name          string       `json:"name"`
	Type          string       `json:"type"`
	TTL           *int64       `json:"ttl,omitempty"`
	Values        []string     `json:"values,omitempty"`
	SetIdentifier string       `json:"set_identifier,omitempty"`
	Alias         *AliasTarget `json:"alias,omitempty"`
}

type ChangeResult struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	SubmittedAt string `json:"submitted_at,omitempty"`
}
