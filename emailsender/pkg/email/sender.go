// Package email contains methods to send emails via SMTP protocol
package email

import (
	"net/smtp"

	"github.com/golang/glog"
)

// SMTPSend an interface for SMTP Email send function
type SMTPSend func(addr string, a smtp.Auth, from string, to []string, msg []byte) error

// Sender keeps all necessary configuration for sending email via SMTP
type Sender struct {
	smtpHost string
	smtpPort string
	from     string
	password string
	identity string

	smtpSend SMTPSend
}

// NewSender creates a new Sender instance
func NewSender(smtpHost string, smtpPort string, from string, password string, identity string) *Sender {
	return &Sender{
		smtpHost: smtpHost,
		smtpPort: smtpPort,
		from:     from,
		password: password, // pragma: allowlist secret
		identity: identity,
		smtpSend: smtp.SendMail,
	}
}

// Send sends a email via SMTP protocol
func (s *Sender) Send(recipient string) error {
	auth := smtp.PlainAuth(s.identity, s.from, s.password, s.smtpHost)

	subject := "subject"
	body := "This is a test"
	message := []byte("To: " + recipient + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body + "\r\n")

	err := s.smtpSend(s.smtpHost+":"+s.smtpPort, auth, s.from, []string{recipient}, message)
	if err != nil {
		glog.Errorf("Failed sending email: %v", err)
		return err
	}

	return nil
}
