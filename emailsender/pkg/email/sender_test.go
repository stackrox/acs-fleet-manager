package email

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ses"
)

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

	mockClient := &MockSESClient{
		SendRawEmailFunc: func(ctx context.Context, params *ses.SendRawEmailInput, optFns ...func(*ses.Options)) (*ses.SendRawEmailOutput, error) {
			called = true
			return &ses.SendRawEmailOutput{
				MessageId: aws.String("test-message-id"),
			}, nil
		},
	}
	mockedSES := &SES{sesClient: mockClient}
	mockedSender := MailSender{
		from,
		mockedSES,
	}

	err := mockedSender.Send(context.Background(), to, rawMessage)

	assert.NoError(t, err)
	assert.True(t, called)
}
