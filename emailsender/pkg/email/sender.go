package email

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/emailsender/config"
	"github.com/stackrox/acs-fleet-manager/emailsender/pkg/metrics"
)

const (
	// EmailProviderAWSSES is the type name for the AWSEmailSender implementation of the Sender interface
	EmailProviderAWSSES = "AWS_SES"
	// EmailProviderLog is the type name for the LogEmailSender implementation of the Sender interface
	EmailProviderLog = "LOG"

	toFormat   = "To: %s\r\n"
	fromFormat = "From: %s <%s>\r\n"
)

// Sender defines the interface to send emails
type Sender interface {
	Send(ctx context.Context, to []string, rawMessage []byte, tenantID string) error
}

// NewEmailSender return a initialized Sender implementation according to the provider configured in cfg.EmailProvider
func NewEmailSender(ctx context.Context, cfg *config.Config, rateLimiter RateLimiter) (Sender, error) {
	switch cfg.EmailProvider {
	case EmailProviderLog:
		return &LogEmailSender{
			from: cfg.SenderAddress,
		}, nil
	case EmailProviderAWSSES:
		// EmailProviderAWSSES is the default
		ses, err := NewSES(ctx, cfg.SesMaxBackoffDelay, cfg.SesMaxAttempts)
		if err != nil {
			return nil, err
		}
		return &AWSMailSender{
			from:        cfg.SenderAddress,
			fromAlias:   cfg.SenderAlias,
			ses:         ses,
			rateLimiter: rateLimiter,
		}, nil
	}
	panic("Unknown email provider: " + cfg.EmailProvider)
}

// LogEmailSender is a Sender implementation that logs email messages to glog
type LogEmailSender struct {
	from string
}

// Send simulates sending an email by logging given message to glog.
func (l *LogEmailSender) Send(ctx context.Context, to []string, rawMessage []byte, tenantID string) error {
	glog.Infof("LogEmailSender.Send called with: to: %s, rawMessage: '%s', tenantID: '%s', from: '%s'", to, string(rawMessage), tenantID, l.from)
	return nil
}

// AWSMailSender is the default implementation for the Sender interface
type AWSMailSender struct {
	from        string
	fromAlias   string
	ses         *SES
	rateLimiter RateLimiter
}

// Send sends an email to the given AWS SES
func (s *AWSMailSender) Send(ctx context.Context, to []string, rawMessage []byte, tenantID string) error {
	// Even though AWS adds the "from" handler we need to set it to the message to show
	// an alias in email inboxes. It is more human friendly (noreply@rhacs-dev.com vs. RHACS Cloud Service)
	if !s.rateLimiter.IsAllowed(tenantID) {
		metrics.DefaultInstance().IncThrottledSendEmail(tenantID)
		return fmt.Errorf("rate limit exceeded for tenant: %s", tenantID)
	}
	fromBytes := []byte(fmt.Sprintf(fromFormat, s.fromAlias, s.from))
	toBytes := []byte(fmt.Sprintf(toFormat, strings.Join(to, ",")))

	raw := bytes.Join([][]byte{fromBytes, toBytes, rawMessage}, nil)
	metrics.DefaultInstance().IncSendEmail(tenantID)
	_, err := s.ses.SendRawEmail(ctx, s.from, to, raw)
	if err != nil {
		metrics.DefaultInstance().IncFailedSendEmail(tenantID)
		return fmt.Errorf("failed to send email: %v", err)
	}
	if err = s.rateLimiter.PersistEmailSendEvent(tenantID); err != nil {
		return fmt.Errorf("failed to store email sent event for teantnt %s: %v", tenantID, err)
	}

	return nil
}
