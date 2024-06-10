package email

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type MockSESClient struct {
	sender     string
	to         []string
	subject    string
	htmlBody   string
	textBody   string
	rawMessage []byte

	SendEmailFunc    func(ctx context.Context, params *ses.SendEmailInput, optFns ...func(*ses.Options)) (*ses.SendEmailOutput, error)
	SendRawEmailFunc func(ctx context.Context, params *ses.SendRawEmailInput, optFns ...func(*ses.Options)) (*ses.SendRawEmailOutput, error)
}

func (m *MockSESClient) SendEmail(ctx context.Context, params *ses.SendEmailInput, optFns ...func(*ses.Options)) (*ses.SendEmailOutput, error) {
	m.sender = *params.Source
	m.to = params.Destination.ToAddresses
	m.subject = *params.Message.Subject.Data
	m.htmlBody = *params.Message.Body.Html.Data
	m.textBody = *params.Message.Body.Text.Data

	return m.SendEmailFunc(ctx, params, optFns...)
}

func (m *MockSESClient) SendRawEmail(ctx context.Context, params *ses.SendRawEmailInput, optFns ...func(*ses.Options)) (*ses.SendRawEmailOutput, error) {
	m.sender = *params.Source
	m.to = params.Destinations
	m.rawMessage = params.RawMessage.Data

	return m.SendRawEmailFunc(ctx, params, optFns...)
}

func TestSendEmail_Success(t *testing.T) {
	sender := "sender@example.com"
	to := []string{"to1@example.com", "to2@example.com"}
	subject := "subject"
	htmlBody := "<h1>HTML body</h1>"
	textBody := "text body"

	testMessageID := "test-message-id"

	mockClient := &MockSESClient{
		SendEmailFunc: func(ctx context.Context, params *ses.SendEmailInput, optFns ...func(*ses.Options)) (*ses.SendEmailOutput, error) {
			return &ses.SendEmailOutput{
				MessageId: aws.String(testMessageID),
			}, nil
		},
	}
	mockedSES := SES{sesClient: mockClient}

	messageID, err := mockedSES.SendEmail(context.Background(), sender, to, subject, htmlBody, textBody)
	assert.NoError(t, err)
	assert.Equal(t, testMessageID, messageID)
	assert.Equal(t, sender, mockClient.sender)
	assert.Equal(t, to, mockClient.to)
	assert.Equal(t, subject, mockClient.subject)
	assert.Equal(t, htmlBody, mockClient.htmlBody)
	assert.Equal(t, textBody, mockClient.textBody)
}

func TestSendEmail_Failure(t *testing.T) {
	errorText := "failed to send email"

	mockClient := &MockSESClient{
		SendEmailFunc: func(ctx context.Context, params *ses.SendEmailInput, optFns ...func(*ses.Options)) (*ses.SendEmailOutput, error) {
			return nil, errors.New(errorText)
		},
	}
	mockedSES := SES{sesClient: mockClient, backoffMaxDuration: 1 * time.Second}

	messageID, err := mockedSES.SendEmail(
		context.Background(),
		"sender@example.com",
		[]string{"to@example.com"},
		"Test Subject",
		"<h1>HTML body</h1>",
		"Text body",
	)
	assert.Error(t, err)
	assert.Equal(t, "", messageID)
	assert.Contains(t, err.Error(), errorText)
}

func TestSendRawEmail_Success(t *testing.T) {
	sender := "sender@example.com"
	to := []string{"to1@example.com", "to2@example.com"}
	subject := "Test subject"
	textBody := "text body"
	var messageBuf bytes.Buffer
	messageBuf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	messageBuf.WriteString(textBody)
	rawMessage := messageBuf.Bytes()

	testMessageID := "test-message-id"

	mockClient := &MockSESClient{
		SendRawEmailFunc: func(ctx context.Context, params *ses.SendRawEmailInput, optFns ...func(*ses.Options)) (*ses.SendRawEmailOutput, error) {
			return &ses.SendRawEmailOutput{
				MessageId: aws.String(testMessageID),
			}, nil
		},
	}
	mockedSES := SES{sesClient: mockClient}

	messageID, err := mockedSES.SendRawEmail(context.Background(), sender, to, rawMessage)
	assert.NoError(t, err)
	assert.Equal(t, testMessageID, messageID)
	assert.Equal(t, sender, mockClient.sender)
	assert.Equal(t, to, mockClient.to)
	assert.Equal(t, rawMessage, mockClient.rawMessage)
}

func TestSendRawEmail_Failure(t *testing.T) {
	errorText := "failed to send email"

	mockClient := &MockSESClient{
		SendRawEmailFunc: func(ctx context.Context, params *ses.SendRawEmailInput, optFns ...func(*ses.Options)) (*ses.SendRawEmailOutput, error) {
			return nil, errors.New(errorText)
		},
	}
	mockedSES := SES{sesClient: mockClient, backoffMaxDuration: 1 * time.Second}

	buf := bytes.NewBuffer(nil)
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", "Test Subject"))
	buf.WriteString("test body")
	rawMessage := buf.Bytes()

	messageID, err := mockedSES.SendRawEmail(
		context.Background(),
		"sender@example.com",
		[]string{"to@example.com"},
		rawMessage,
	)
	assert.Error(t, err)
	assert.Equal(t, "", messageID)
	assert.Contains(t, err.Error(), errorText)
}
