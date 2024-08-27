package email

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aws/aws-sdk-go-v2/service/ses"
)

type MockedRateLimiter struct {
	calledIsAllowed             bool
	calledPersistEmailSendEvent bool

	IsAllowedFunc             func(tenantID string) bool
	PersistEmailSendEventFunc func(tenantID string) error
}

func (m *MockedRateLimiter) IsAllowed(tenantID string) bool {
	m.calledIsAllowed = true
	return m.IsAllowedFunc(tenantID)
}

func (m *MockedRateLimiter) PersistEmailSendEvent(tenantID string) error {
	m.calledPersistEmailSendEvent = true
	return m.PersistEmailSendEventFunc(tenantID)
}

func TestSend_Success(t *testing.T) {
	from := "sender@example.com"
	fromAlias := "RHACS Cloud Service"
	to := []string{"to1@example.com", "to2@example.com"}
	subject := "Test subject"
	textBody := "text body"
	var messageBuf bytes.Buffer
	messageBuf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	messageBuf.WriteString(textBody)
	rawMessage := messageBuf.Bytes()
	called := false
	tenantID := "test-tenant-id"

	mockClient := &MockSESClient{
		SendRawEmailFunc: func(ctx context.Context, params *ses.SendRawEmailInput, optFns ...func(*ses.Options)) (*ses.SendRawEmailOutput, error) {
			called = true
			return &ses.SendRawEmailOutput{
				MessageId: aws.String("test-message-id"),
			}, nil
		},
	}
	mockedRateLimiter := &MockedRateLimiter{
		IsAllowedFunc: func(tenantID string) bool {
			return true
		},
		PersistEmailSendEventFunc: func(tenantID string) error {
			return nil
		},
	}
	mockedSES := &SES{sesClient: mockClient}
	mockedSender := AWSMailSender{
		from,
		fromAlias,
		mockedSES,
		mockedRateLimiter,
	}

	err := mockedSender.Send(context.Background(), to, rawMessage, tenantID)

	assert.NoError(t, err)
	assert.True(t, called)
	assert.True(t, mockedRateLimiter.calledIsAllowed)
	assert.True(t, mockedRateLimiter.calledPersistEmailSendEvent)
}

func TestSend_LimitExceeded(t *testing.T) {
	var messageBuf bytes.Buffer
	rawMessage := messageBuf.Bytes()

	mockClient := &MockSESClient{}
	mockedRateLimiter := &MockedRateLimiter{
		IsAllowedFunc: func(tenantID string) bool {
			return false
		},
	}
	mockedSES := &SES{sesClient: mockClient}
	mockedSender := AWSMailSender{
		"from@example.com",
		"from-alias",
		mockedSES,
		mockedRateLimiter,
	}

	err := mockedSender.Send(context.Background(), []string{"to@example.com"}, rawMessage, "test-tenant-id")

	assert.ErrorContains(t, err, "rate limit exceeded")
	assert.True(t, mockedRateLimiter.calledIsAllowed)
	assert.False(t, mockedRateLimiter.calledPersistEmailSendEvent)
}

func TestSendAppendsFromAndTo(t *testing.T) {
	from := "sender@example.com"
	fromAlias := "RHACS Cloud Service"
	to := []string{"to1@example.com", "to2@example.com"}
	textBody := "text body"
	tenantID := "test-tenant-id"

	var calledWith *ses.SendRawEmailInput

	mockClient := &MockSESClient{
		SendRawEmailFunc: func(ctx context.Context, params *ses.SendRawEmailInput, optFns ...func(*ses.Options)) (*ses.SendRawEmailOutput, error) {
			calledWith = params

			return &ses.SendRawEmailOutput{
				MessageId: aws.String("test-message-id"),
			}, nil
		},
	}
	mockedRateLimiter := &MockedRateLimiter{
		IsAllowedFunc: func(tenantID string) bool {
			return true
		},
		PersistEmailSendEventFunc: func(tenantID string) error {
			return nil
		},
	}
	mockedSES := &SES{sesClient: mockClient}
	sender := AWSMailSender{
		from,
		fromAlias,
		mockedSES,
		mockedRateLimiter,
	}

	err := sender.Send(context.Background(), to, []byte(textBody), tenantID)
	require.NoError(t, err)
	require.NotNil(t, calledWith)
	require.NotNil(t, calledWith.RawMessage)

	msg := calledWith.RawMessage.Data
	require.NotEmpty(t, msg)

	stringMsg := string(msg)
	require.Contains(t, stringMsg, "From: RHACS Cloud Service <sender@example.com>\r\n")
	require.Contains(t, stringMsg, "To: to1@example.com,to2@example.com\r\n")

}
