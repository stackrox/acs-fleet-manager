package logger

import (
	"context"
	"fmt"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/golang-jwt/jwt/v4"
	"github.com/golang/glog"
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

// UHCLogger ...
type UHCLogger interface {
	V(level int32) UHCLogger
	Infof(format string, args ...interface{})
	Warningf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Error(err error)
	Fatalf(format string, args ...interface{})
}

// Logger is a logger with a background context
var Logger = NewUHCLogger(context.Background())
var _ UHCLogger = &logger{}

type logger struct {
	context   context.Context
	level     int32
	accountID string
	// TODO username is unused, should we be logging it? Could be pii
	username  string
	session   string
	sentryHub *sentry.Hub
}

// NewUHCLogger creates a new logger instance with a default verbosity of 1
func NewUHCLogger(ctx context.Context) UHCLogger {
	logger := &logger{
		context:   ctx,
		level:     1,
		username:  getUsernameFromClaims(ctx),
		sentryHub: sentry.GetHubFromContext(ctx),
		session:   getSessionFromClaims(ctx),
	}
	return logger
}

func (l *logger) prepareLogPrefix(format string, args ...interface{}) string {
	orig := fmt.Sprintf(format, args...)
	prefix := ""

	if l.username != "" {
		prefix = strings.Join([]string{prefix, "user='", l.username, "' "}, "")
	}

	if event, ok := l.context.Value(ActionKey).(string); ok {
		prefix = strings.Join([]string{prefix, "action='", event, "' "}, "")
		if eventStatus, ok := l.context.Value(ActionResultKey).(string); ok {
			prefix = strings.Join([]string{prefix, "result='", eventStatus, "' "}, "")
		}
	}

	if remoteAddr, ok := l.context.Value(RemoteAddrKey).(string); ok {
		prefix = strings.Join([]string{prefix, "src_ip='", remoteAddr, "' "}, "")
	}

	if l.session != "" {
		prefix = strings.Join([]string{prefix, "session='", l.session, "' "}, "")
	}

	if txid, ok := l.context.Value("txid").(int64); ok {
		prefix = strings.Join([]string{prefix, "tx_id='", fmt.Sprintf("%v", txid), "' "}, "")
	}

	if l.accountID != "" {
		prefix = strings.Join([]string{prefix, "accountID='", l.accountID, "' "}, "")
	}

	if opid, ok := l.context.Value(OpIDKey).(string); ok {
		prefix = strings.Join([]string{prefix, "opid='", opid, "' "}, "")
	}

	return prefix + orig
}

// V ...
func (l *logger) V(level int32) UHCLogger {
	return &logger{
		context:   l.context,
		accountID: l.accountID,
		username:  l.username,
		session:   l.session,
		level:     level,
	}
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

// Infof ...
func (l *logger) Infof(format string, args ...interface{}) {
	prefixed := l.prepareLogPrefix(format, args...)
	glog.V(glog.Level(l.level)).Infof(prefixed)
}

// Warningf ...
func (l *logger) Warningf(format string, args ...interface{}) {
	prefixed := l.prepareLogPrefix(format, args...)
	glog.Warningln(prefixed)
	l.captureSentryEvent(sentry.LevelWarning, format, args...)
}

// Errorf ...
func (l *logger) Errorf(format string, args ...interface{}) {
	prefixed := l.prepareLogPrefix(format, args...)
	glog.Errorln(prefixed)
	l.captureSentryEvent(sentry.LevelError, format, args...)
}

// Error ...
func (l *logger) Error(err error) {
	glog.Error(err)
	if l.sentryHub == nil {
		sentry.CaptureException(err)
		return
	}
	l.sentryHub.CaptureException(err)
}

// Fatalf ...
func (l *logger) Fatalf(format string, args ...interface{}) {
	prefixed := l.prepareLogPrefix(format, args...)
	glog.Fatalln(prefixed)
	l.captureSentryEvent(sentry.LevelFatal, format, args...)
}

func (l *logger) captureSentryEvent(level sentry.Level, format string, args ...interface{}) {
	event := sentry.NewEvent()
	event.Level = level
	event.Message = fmt.Sprintf(format, args...)
	if l.sentryHub == nil {
		glog.Warning("Sentry hub not present in logger")
		sentry.CaptureEvent(event)
		return
	}
	l.sentryHub.CaptureEvent(event)
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
