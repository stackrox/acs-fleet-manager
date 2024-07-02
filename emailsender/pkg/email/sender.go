package email

import (
	"bytes"
	"context"
	"fmt"
	"github.com/stackrox/acs-fleet-manager/emailsender/pkg/metrics"
)

const fromTemplate = "From: RHACS Cloud Service <%s>\r\n"

// Sender defines the interface to send emails
type Sender interface {
	Send(ctx context.Context, to []string, rawMessage []byte, tenantID string) error
}

// MailSender is the default implementation for the Sender interface
type MailSender struct {
	from        string
	ses         *SES
	rateLimiter RateLimiter
}

// NewEmailSender returns a new MailSender instance
func NewEmailSender(from string, ses *SES, rateLimiter RateLimiter) *MailSender {
	return &MailSender{
		from:        from,
		ses:         ses,
		rateLimiter: rateLimiter,
	}
}

// Send sends an email to the given AWS SES
func (s *MailSender) Send(ctx context.Context, to []string, rawMessage []byte, tenantID string) error {
	// Even though AWS adds the "from" handler we need to set it to the message to show
	// an alias in email inboxes. It is more human friendly (noreply@rhacs-dev.com vs. RHACS Cloud Service)
	if !s.rateLimiter.IsAllowed(tenantID) {
		metrics.DefaultInstance().IncThrottledSendEmail(tenantID)
		return fmt.Errorf("rate limit exceeded for tenant: %s", tenantID)
	}
	fromBytes := []byte(fmt.Sprintf(fromTemplate, s.from))
	raw := bytes.Join([][]byte{fromBytes, rawMessage}, nil)
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
