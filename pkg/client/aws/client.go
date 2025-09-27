// Package aws ...
package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// Client ...
//
//go:generate moq -out client_moq.go . Client
type Client interface {
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

	// For future AWS service integrations, the config would be used here
	_ = aws.Config{
		Credentials: credentialsCache,
		Region:      region,
		Retryer:     func() aws.Retryer { return retry.AddWithMaxAttempts(retry.NewStandard(), 2) },
	}

	return &awsClient{}, nil
}

type awsClient struct{}
