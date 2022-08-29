package aws

import (
	"fmt"
	"strings"

	errors "github.com/zgalor/weberr"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	awscredentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
)

// Client ...
//
//go:generate moq -out client_moq.go . Client
type Client interface {
	// route53
	ListHostedZonesByNameInput(dnsName string) (*route53.ListHostedZonesByNameOutput, error)
	ChangeResourceRecordSets(dnsName string, recordChangeBatch *route53.ChangeBatch) (*route53.ChangeResourceRecordSetsOutput, error)
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

// MockClientFactory ...
type MockClientFactory struct {
	mock Client
}

// NewClient ...
func (m *MockClientFactory) NewClient(credentials Config, region string) (Client, error) {
	return m.mock, nil
}

// NewMockClientFactory ...
func NewMockClientFactory(client Client) *MockClientFactory {
	return &MockClientFactory{
		mock: client,
	}
}

type awsClient struct {
	route53Client route53iface.Route53API
}

// Config contains the AWS settings
type Config struct {
	// AccessKeyID is the AWS access key identifier.
	AccessKeyID string
	// SecretAccessKey is the AWS secret access key.
	SecretAccessKey string
}

func newClient(credentials Config, region string) (Client, error) {
	cfg := &aws.Config{
		Credentials: awscredentials.NewStaticCredentials(
			credentials.AccessKeyID,
			credentials.SecretAccessKey,
			""),
		Region:  aws.String(region),
		Retryer: client.DefaultRetryer{NumMaxRetries: 2},
	}
	sess, err := session.NewSession(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating new session: %w", err)
	}
	return &awsClient{
		route53Client: route53.New(sess),
	}, nil
}

// GetChange ...
func (client *awsClient) GetChange(changeID string) (*route53.GetChangeOutput, error) {
	changeInput := &route53.GetChangeInput{
		Id: &changeID,
	}

	change, err := client.route53Client.GetChange(changeInput)
	if err != nil {
		return nil, wrapAWSError(err, "Failed to get Change.")
	}

	return change, nil
}

// ListHostedZonesByNameInput ...
func (client *awsClient) ListHostedZonesByNameInput(dnsName string) (*route53.ListHostedZonesByNameOutput, error) {
	maxItems := "1"
	requestInput := &route53.ListHostedZonesByNameInput{
		DNSName:  &dnsName,
		MaxItems: &maxItems,
	}

	zone, err := client.route53Client.ListHostedZonesByName(requestInput)
	if err != nil {
		return nil, wrapAWSError(err, "Failed to get DNS zone.")
	}
	return zone, nil
}

// ChangeResourceRecordSets ...
func (client *awsClient) ChangeResourceRecordSets(dnsName string, recordChangeBatch *route53.ChangeBatch) (*route53.ChangeResourceRecordSetsOutput, error) {
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

	recordSetsOutput, err := client.route53Client.ChangeResourceRecordSets(recordChanges)

	if err != nil {
		awsErr := err.(awserr.Error)
		if awsErr.Code() == "InvalidChangeBatch" {
			errorMessage := awsErr.Message()

			// Record set not created in the first place
			recordSetNotFound := strings.Contains(errorMessage, "but it was not found")

			// Dinosaur cluster failed to create on the cluster, we have an entry in the database.
			recordSetDomainNameEmpty := strings.Contains(errorMessage, "Domain name is empty")

			// Record set has already been created
			recordSetAlreadyExists := strings.Contains(errorMessage, "but it already exists")

			if recordSetNotFound || recordSetDomainNameEmpty || recordSetAlreadyExists {
				return nil, nil
			}
		}
		return nil, wrapAWSError(err, "Failed to get DNS zone.")
	}
	return recordSetsOutput, nil
}

func wrapAWSError(err error, msg string) error {
	switch err.(type) {
	case awserr.RequestFailure:
		return errors.BadRequest.UserWrapf(err, msg) //nolint:wrapcheck
	default:
		return err
	}
}
