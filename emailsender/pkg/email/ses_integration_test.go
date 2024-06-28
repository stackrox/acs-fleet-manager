package email

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	// awsDevSender is the domain configured for sending in our AWS dev account, see aws config repository
	awsDevSender        = "noreply@mail.rhacs-dev.com"
	invalidSender       = "noreply@some.invalid.integration.test.domain"
	sesSimulatorSuccess = "success@simulator.amazonses.com"
	defaultMessage      = "this is a test email for AWS integration tests"
)

type emailTestCase struct {
	sender       string
	to           string
	msg          string
	requireError bool
}

var commonTests = map[string]emailTestCase{
	// The combination of valid/invalid sender tests makes sure the actual AWS SES API is used as opposed
	// to a mock which would accept both sender identity, even one that is not configure in the aws config repository
	"succesful email": {
		sender:       awsDevSender,
		to:           sesSimulatorSuccess,
		msg:          defaultMessage,
		requireError: false,
	},
	"error for invalid sender": {
		sender:       invalidSender,
		to:           sesSimulatorSuccess,
		msg:          defaultMessage,
		requireError: true,
	},
}

func TestSendEmail(t *testing.T) {
	skipIfNotAwsIntegrationTest(t)

	ses, err := NewSES(context.Background(), time.Second*10, 3)
	require.NoError(t, err, "failed to initizializ SES")

	for name, tc := range commonTests {
		t.Run(name, func(t *testing.T) {
			msgID, err := ses.SendEmail(context.Background(), tc.sender, []string{tc.to}, "Test Email", "", tc.msg)
			if tc.requireError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err, "unexpected error on SendEmail")
			require.NotEmpty(t, msgID)
		})
	}
}

func TestSendRawEmail(t *testing.T) {
	skipIfNotAwsIntegrationTest(t)

	ses, err := NewSES(context.Background(), time.Second*10, 3)
	require.NoError(t, err, "failed to initizializ SES")

	tests := map[string]emailTestCase{
		"invalid from header": {
			sender:       awsDevSender,
			to:           sesSimulatorSuccess,
			msg:          fmt.Sprintf("From: %s\n%s", invalidSender, defaultMessage),
			requireError: true,
		},
	}

	for k, v := range commonTests {
		tests[k] = v
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			msgID, err := ses.SendRawEmail(context.Background(), tc.sender, []string{tc.to}, []byte(tc.msg))
			if tc.requireError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err, "unexpected error on SendEmail")
			require.NotEmpty(t, msgID)
		})
	}
}

func skipIfNotAwsIntegrationTest(t *testing.T) {
	if os.Getenv("RUN_AWS_INTEGRATION") != "true" {
		t.Skip("Skip SES integration tests. Set RUN_AWS_INTEGRATION=true env variable to enable.")
	}
}
