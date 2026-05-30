package config

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

const DefaultRegion = "us-east-1"

type AWSOptions struct {
	Profile string
	Region  string
	RoleARN string
}

func LoadAWS(ctx context.Context, opts AWSOptions) (aws.Config, error) {
	loadOptions := []func(*awsconfig.LoadOptions) error{}
	region := opts.Region
	if region == "" {
		region = DefaultRegion
	}
	loadOptions = append(loadOptions, awsconfig.WithRegion(region))
	if opts.Profile != "" {
		loadOptions = append(loadOptions, awsconfig.WithSharedConfigProfile(opts.Profile))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return aws.Config{}, err
	}
	if cfg.Region == "" {
		cfg.Region = DefaultRegion
	}
	if opts.RoleARN != "" {
		stsClient := sts.NewFromConfig(cfg)
		provider := stscreds.NewAssumeRoleProvider(stsClient, opts.RoleARN, func(options *stscreds.AssumeRoleOptions) {
			options.RoleSessionName = "route53-cli"
		})
		cfg.Credentials = aws.NewCredentialsCache(provider)
	}
	return cfg, nil
}
