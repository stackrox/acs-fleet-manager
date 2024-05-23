// Package email contains methods to send emails via AWS SES
package email

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/golang/glog"
)

// SES struct keeps necessary configuration for sending email via AWS SES
type SES struct {
	sesClient SESClient
}

// NewSES creates a new SES instance with initialised AWS SES client using AWS Config
func NewSES(ctx context.Context) (*SES, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to laod AWS SDK config: %v", err)
	}
	sesClient := ses.NewFromConfig(cfg)

	return &SES{sesClient: sesClient}, nil
}

// SESClient is an interface that sends email using provided function
type SESClient interface {
	SendEmail(ctx context.Context, params *ses.SendEmailInput, optFns ...func(*ses.Options)) (*ses.SendEmailOutput, error)
}

// SendEmail sends email via AWS SES API
func (s *SES) SendEmail(ctx context.Context, sender string, to []string, subject, htmlBody, textBody string) (string, error) {
	input := &ses.SendEmailInput{
		Source: aws.String(sender),
		Destination: &types.Destination{
			ToAddresses: to,
		},
		Message: &types.Message{
			Subject: &types.Content{
				Data: aws.String(subject),
			},
			Body: &types.Body{
				Html: &types.Content{
					Data: aws.String(htmlBody),
				},
				Text: &types.Content{
					Data: aws.String(textBody),
				},
			},
		},
	}

	result, err := s.sesClient.SendEmail(ctx, input)
	if err != nil {
		glog.Errorf("Failed sending email: %v", err)
		return "", fmt.Errorf("failed to send email: %v", err)
	}

	return *result.MessageId, nil
}
