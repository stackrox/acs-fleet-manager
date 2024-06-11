package email

import (
	"bytes"
	"context"
	"fmt"

	"github.com/golang/glog"
)

const fromTemplate = "From: RHACS Cloud Service <%s>\r\n"

// Sender defines the interface to send emails
type Sender interface {
	Send(ctx context.Context, to []string, rawMessage []byte) error
}

// MailSender is the default implementation for the Sender interface
type MailSender struct {
	from string
	ses  *SES
}

// NewEmailSender returns a new MailSender instance
func NewEmailSender(from string, ses *SES) *MailSender {
	return &MailSender{
		from: from,
		ses:  ses,
	}
}

// Send sends an email to the given AWS SES
func (s *MailSender) Send(ctx context.Context, to []string, rawMessage []byte) error {
	// Even though AWS adds the from handler we need to set it the the message to show
	// an alias in email inboxes that is more human friendly (noreply@rhacs-dev.com vs. RHACS Cloud Service)
	fromBytes := []byte(fmt.Sprintf(fromTemplate, s.from))
	raw := bytes.Join([][]byte{fromBytes, rawMessage}, nil)
	_, err := s.ses.SendRawEmail(ctx, s.from, to, raw)
	if err != nil {
		glog.Errorf("Failed sending email: %v", err)
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}
