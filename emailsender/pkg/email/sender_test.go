package email

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"

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
		mockedSES,
		mockedRateLimiter,
	}

	err := mockedSender.Send(context.Background(), []string{"to@example.com"}, rawMessage, "test-tenant-id")

	assert.ErrorContains(t, err, "rate limit exceeded")
	assert.True(t, mockedRateLimiter.calledIsAllowed)
	assert.False(t, mockedRateLimiter.calledPersistEmailSendEvent)
}
