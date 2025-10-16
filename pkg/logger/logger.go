// Package logger ...
package logger

import (
	"context"
	"fmt"
	"strings"
	"sync"

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
	logDepth          = 1
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
	InfoDepth(depth int, args ...interface{})
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
	username string
	session  string
}

// NewUHCLogger creates a new logger instance with a default verbosity of 1
func NewUHCLogger(ctx context.Context) UHCLogger {
	logger := &logger{
		context:  ctx,
		level:    1,
		username: getUsernameFromClaims(ctx),
		session:  getSessionFromClaims(ctx),
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
	glog.InfoDepth(logDepth, prefixed)
}

// InfoDepth logs an info log message, wrapping glogs info depth
func (l *logger) InfoDepth(depth int, args ...interface{}) {
	glog.InfoDepth(logDepth+depth, args...)
}

var cache = sync.Map{}

// InfoChangedInt32 logs whenever the value of the passed counter reference changes when it is called.
func InfoChangedInt32(counter *int32, msg string, args ...interface{}) {
	doIfChangedInt32(counter, func() {
		glog.InfoDepth(logDepth, fmt.Sprintf(msg, args...))
	})
}

func doIfChangedInt32(counter *int32, f func()) {
	prev, loaded := cache.Swap(counter, *counter)
	if loaded && prev == *counter {
		return // counter not changed, do not log
	}

	f()
}

// Warningf ...
func (l *logger) Warningf(format string, args ...interface{}) {
	prefixed := l.prepareLogPrefix(format, args...)
	glog.WarningDepth(logDepth, prefixed)
}

// Errorf ...
func (l *logger) Errorf(format string, args ...interface{}) {
	prefixed := l.prepareLogPrefix(format, args...)
	glog.Errorln(prefixed)
	glog.ErrorDepth(logDepth, prefixed)
}

// Error ...
func (l *logger) Error(err error) {
	glog.ErrorDepth(logDepth, err)
}

// Fatalf ...
func (l *logger) Fatalf(format string, args ...interface{}) {
	prefixed := l.prepareLogPrefix(format, args...)
	glog.FatalDepth(logDepth, prefixed)
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
