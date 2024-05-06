package email

import (
	"net/smtp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSMTP struct {
	address string
	from    string
	to      []string
	message []byte
	called  bool
}

func (m *mockSMTP) SendMail(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
	m.address = addr
	m.called = true
	m.from = from
	m.to = to
	m.message = msg
	return nil
}

func TestSend(t *testing.T) {
	testSMTP := new(mockSMTP)
	mockedSendMail := testSMTP.SendMail

	host := "smtp.example.com"
	port := "587"
	from := "sender@example.com"
	to := "to@example.com"
	subject := "subject"
	password := "smtppassword" // pragma: allowlist secret
	identity := "smtpidentity"

	testSender := &Sender{
		smtpHost: host,
		smtpPort: port,
		from:     from,
		password: password, // pragma: allowlist secret
		identity: identity,
		smtpSend: mockedSendMail,
	}

	err := testSender.Send(to)
	require.NoError(t, err)

	body := "This is a test"
	expectedMessage := []byte("To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body + "\r\n")

	assert.True(t, testSMTP.called)
	assert.Equal(t, host+":"+port, testSMTP.address)
	assert.Equal(t, from, testSMTP.from)
	assert.Equal(t, to, testSMTP.to[0])
	assert.Equal(t, expectedMessage, testSMTP.message)
}
