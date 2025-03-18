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
	// EmailProviderLog is the type name for the LogEmailSender implementation of the Sender interface
	EmailProviderLog = "LOG"

	toFormat   = "To: %s\r\n"
	fromFormat = "From: RHACS Cloud Service %s <%s>\r\n"
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
	default:
		// EmailProviderAWSSES is the default
		ses, err := NewSES(ctx, cfg.SesMaxBackoffDelay, cfg.SesMaxAttempts)
		if err != nil {
			return nil, err
		}
		return &AWSMailSender{
			from:        cfg.SenderAddress,
			ses:         ses,
			rateLimiter: rateLimiter,
		}, nil

	}
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
	ses         *SES
	rateLimiter RateLimiter
}

// RateLimitError is returned when a tenant has reached its email sending limit
type RateLimitError struct {
	TenantID string
}

func (e RateLimitError) Error() string {
	return fmt.Sprintf("email rate limit exceeded for tenant: %s", e.TenantID)
}

// Send sends an email to the given AWS SES
func (s *AWSMailSender) Send(ctx context.Context, to []string, rawMessage []byte, tenantID string) error {

	allowed, err := s.rateLimiter.IsAllowed(tenantID)
	if err != nil {
		return fmt.Errorf("failed to determine rate limit: %w", err)
	}

	if !allowed {
		metrics.DefaultInstance().IncThrottledSendEmail(tenantID)
		return RateLimitError{TenantID: tenantID}
	}
	// Even though AWS adds the "from" handler we need to set it to the message to show
	// an alias in email inboxes. It is more human friendly (noreply@rhacs-dev.com vs. RHACS Cloud Service)
	fromBytes := []byte(fmt.Sprintf(fromFormat, tenantID, s.from))
	toBytes := []byte(fmt.Sprintf(toFormat, strings.Join(to, ",")))

	raw := bytes.Join([][]byte{fromBytes, toBytes, rawMessage}, nil)
	metrics.DefaultInstance().IncSendEmail(tenantID)
	_, err = s.ses.SendRawEmail(ctx, s.from, to, raw)
	if err != nil {
		metrics.DefaultInstance().IncFailedSendEmail(tenantID)
		return fmt.Errorf("failed to send email: %v", err)
	}
	if err = s.rateLimiter.PersistEmailSendEvent(tenantID); err != nil {
		return fmt.Errorf("failed to store email sent event for teantnt %s: %v", tenantID, err)
	}

	return nil
}
