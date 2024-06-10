// Package email contains methods to send emails via AWS SES
package email

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/cenkalti/backoff/v4"
	"github.com/golang/glog"
)

// SES struct keeps necessary configuration for sending email via AWS SES
type SES struct {
	sesClient          SESClient
	backoffMaxDuration time.Duration
}

// NewSES creates a new SES instance with initialised AWS SES client using AWS Config
func NewSES(ctx context.Context, backoffMaxDuration time.Duration) (*SES, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to laod AWS SDK config: %v", err)
	}
	sesClient := ses.NewFromConfig(cfg)

	return &SES{sesClient: sesClient, backoffMaxDuration: backoffMaxDuration}, nil
}

// SESClient is an interface that sends email using provided function
type SESClient interface {
	SendEmail(ctx context.Context, params *ses.SendEmailInput, optFns ...func(*ses.Options)) (*ses.SendEmailOutput, error)
	SendRawEmail(ctx context.Context, params *ses.SendRawEmailInput, optFns ...func(*ses.Options)) (*ses.SendRawEmailOutput, error)
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

// SendRawEmail sends raw email message via AWS SES API
// this is a flexible method which allows sending multipart MINE emails with attachments
func (s *SES) SendRawEmail(ctx context.Context, sender string, to []string, rawMessage []byte) (string, error) {
	input := &ses.SendRawEmailInput{
		Source:       aws.String(sender),
		Destinations: to,
		RawMessage: &types.RawMessage{
			Data: rawMessage,
		},
	}

	sendRawEmailFunc := func() (*ses.SendRawEmailOutput, error) {
		return s.sesClient.SendRawEmail(ctx, input)
	}
	backoffNotifyFunc := func(err error, duration time.Duration) {
		glog.Warningf("email has not been sent yet, waited %3.2f: %v", duration.Seconds(), err)
	}
	backoffConfig := backoff.NewExponentialBackOff()
	backoffConfig.MaxElapsedTime = s.backoffMaxDuration

	result, err := backoff.RetryNotifyWithData(sendRawEmailFunc, backoffConfig, backoffNotifyFunc)
	if err != nil {
		return "", fmt.Errorf("could not send raw email after %3.2f seconds: %v", s.backoffMaxDuration.Seconds(), err)
	}

	return *result.MessageId, nil
}
