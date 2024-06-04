package email

import (
	"context"
	"fmt"
	"github.com/golang/glog"
)

type Sender interface {
	Send(ctx context.Context, to []string, rawMessage []byte) error
}

type MailSender struct {
	from string
	ses  *SES
}

func NewEmailSender(from string, ses *SES) *MailSender {
	return &MailSender{
		from: from,
		ses:  ses,
	}
}

func (s *MailSender) Send(ctx context.Context, to []string, rawMessage []byte) error {
	_, err := s.ses.SendRawEmail(ctx, s.from, to, rawMessage)
	if err != nil {
		glog.Errorf("Failed sending email: %v", err)
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}
