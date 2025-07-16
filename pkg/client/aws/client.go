// Package aws ...
package aws

import (
	"context"
	"fmt"

	errors "github.com/zgalor/weberr"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

// Client ...
//
//go:generate moq -out client_moq.go . Client
type Client interface {
	// route53
	ListHostedZonesByNameInput(dnsName string) (*route53.ListHostedZonesByNameOutput, error)
	ChangeResourceRecordSets(dnsName string, recordChangeBatch *types.ChangeBatch) (*route53.ChangeResourceRecordSetsOutput, error)
	GetChange(changeID string) (*route53.GetChangeOutput, error)
}

// ClientFactory ...
type ClientFactory interface {
	NewClient(credentials Config, region string) (Client, error)
}

// DefaultClientFactory ...
type DefaultClientFactory struct{}

// NewClient ...
func (f *DefaultClientFactory) NewClient(credentials Config, region string) (Client, error) {
	return newClient(credentials, region)
}

// NewDefaultClientFactory ...
func NewDefaultClientFactory() *DefaultClientFactory {
	return &DefaultClientFactory{}
}

// Config contains the AWS settings
type Config struct {
	// AccessKeyID is the AWS access key identifier.
	AccessKeyID string
	// SecretAccessKey is the AWS secret access key.
	SecretAccessKey string
}

func newClient(creds Config, region string) (Client, error) {
	credentialsCache := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
		creds.AccessKeyID,
		creds.SecretAccessKey,
		""))

	if _, err := credentialsCache.Retrieve(context.Background()); err != nil {
		// retrieve credentials early to fail if they are not set properly, we do this to mimic
		// the behaviour of config.LoadDefaultConfig of AWS SDK V2 / session.NewSession of AWS SDK V1
		// while keeping the logic to set credentials statically
		return nil, err
	}

	cfg := aws.Config{
		Credentials: credentialsCache,
		Region:      region,
		Retryer:     func() aws.Retryer { return retry.AddWithMaxAttempts(retry.NewStandard(), 2) },
	}

	return &awsClient{
		route53Client: route53.NewFromConfig(cfg),
	}, nil
}

type awsClient struct {
	route53Client *route53.Client
}

// GetChange ...
func (client *awsClient) GetChange(changeID string) (*route53.GetChangeOutput, error) {
	changeInput := &route53.GetChangeInput{
		Id: &changeID,
	}

	change, err := client.route53Client.GetChange(context.TODO(), changeInput)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get DNS Change")
	}

	return change, nil
}

// ListHostedZonesByNameInput ...
func (client *awsClient) ListHostedZonesByNameInput(dnsName string) (*route53.ListHostedZonesByNameOutput, error) {
	requestInput := &route53.ListHostedZonesByNameInput{
		DNSName:  &dnsName,
		MaxItems: aws.Int32(1),
	}

	zone, err := client.route53Client.ListHostedZonesByName(context.TODO(), requestInput)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get DNS zone")
	}

	return zone, nil
}

// ChangeResourceRecordSets ...
func (client *awsClient) ChangeResourceRecordSets(dnsName string, recordChangeBatch *types.ChangeBatch) (*route53.ChangeResourceRecordSetsOutput, error) {
	zones, err := client.ListHostedZonesByNameInput(dnsName)
	if err != nil {
		return nil, err
	}
	if len(zones.HostedZones) == 0 {
		return nil, fmt.Errorf("No Hosted Zones found")
	}

	hostedZoneID := zones.HostedZones[0].Id

	recordChanges := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: hostedZoneID,
		ChangeBatch:  recordChangeBatch,
	}

	recordSetsOutput, err := client.route53Client.ChangeResourceRecordSets(context.TODO(), recordChanges)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to change resource record sets")
	}
	return recordSetsOutput, nil
}
