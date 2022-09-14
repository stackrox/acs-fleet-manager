package logging

import (
	"context"
	"fmt"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/golang-jwt/jwt/v4"
	"github.com/openshift-online/ocm-sdk-go/authentication"
)

// LoggerKeys ...
type LoggerKeys string

// ActionKey ...
const (
	ActionKey       LoggerKeys = "Action"
	ActionResultKey LoggerKeys = "EventResult"
	RemoteAddrKey   LoggerKeys = "RemoteAddr"

	ActionFailed  LoggerKeys = "failed"
	ActionSuccess LoggerKeys = "success"

	logEventSeparator = "$$"
)

// LogEvent ...
type LogEvent struct {
	Type        string
	Description string
}

// NewLogEventFromString ...
func NewLogEventFromString(eventTypeAndDescription string) (logEvent LogEvent) {
	typeAndDesc := strings.Split(eventTypeAndDescription, logEventSeparator)
	sliceLen := len(typeAndDesc)

	if sliceLen > 0 {
		logEvent.Type = typeAndDesc[0]
	}

	if sliceLen > 1 {
		logEvent.Description = typeAndDesc[1]
	}

	return logEvent
}

// NewLogEvent ...
func NewLogEvent(eventType string, description ...string) LogEvent {
	res := LogEvent{
		Type: eventType,
	}

	if len(description) != 0 {
		res.Description = description[0]
	}

	return res
}

// ToString ...
func (l LogEvent) ToString() string {
	if l.Description != "" {
		return fmt.Sprintf("%s%s%s", l.Type, logEventSeparator, l.Description)
	}

	return l.Type
}

// NewUHCLogger creates a new logger instance enriched with session information.
func NewUHCLogger(ctx context.Context) *Logger {
	logger := currentModule(4).Logger()
	logger.SugaredLogger = logger.SugaredLogger.With(
		"username", getUsernameFromClaims(ctx),
		"session", getSessionFromClaims(ctx),
		"sentryHub", sentry.GetHubFromContext(ctx),
	)
	return logger
}

func getSessionFromClaims(ctx context.Context) string {
	var claims jwt.MapClaims
	token, err := authentication.TokenFromContext(ctx)
	if err != nil {
		return ""
	}

	if token != nil && token.Claims != nil {
		claims = token.Claims.(jwt.MapClaims)
	}

	if claims["session_state"] != nil {
		// return username from ocm token
		return claims["session_state"].(string)
	}

	return ""
}

func getUsernameFromClaims(ctx context.Context) string {
	var claims jwt.MapClaims
	token, err := authentication.TokenFromContext(ctx)
	if err != nil {
		return ""
	}

	if token != nil && token.Claims != nil {
		claims = token.Claims.(jwt.MapClaims)
	}

	if claims["username"] != nil {
		// return username from ocm token
		return claims["username"].(string)
	} else if claims["preferred_username"] != nil {
		// return username from sso token
		return claims["preferred_username"].(string)
	}

	return ""
}
