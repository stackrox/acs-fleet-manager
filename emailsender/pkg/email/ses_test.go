package email

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type MockSESClient struct {
	sender   string
	to       []string
	subject  string
	htmlBody string
	textBody string

	SendEmailFunc func(ctx context.Context, params *ses.SendEmailInput, optFns ...func(*ses.Options)) (*ses.SendEmailOutput, error)
}

func (m *MockSESClient) SendEmail(ctx context.Context, params *ses.SendEmailInput, optFns ...func(*ses.Options)) (*ses.SendEmailOutput, error) {
	m.sender = *params.Source
	m.to = params.Destination.ToAddresses
	m.subject = *params.Message.Subject.Data
	m.htmlBody = *params.Message.Body.Html.Data
	m.textBody = *params.Message.Body.Text.Data

	return m.SendEmailFunc(ctx, params, optFns...)
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
	mockedSES := SES{sesClient: mockClient}

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
