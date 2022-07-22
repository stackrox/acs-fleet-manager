package services

import (
	"fmt"
	"regexp"

	"github.com/pkg/errors"
)

// State ...
type State interface {
	accept(tok string) bool
	parse(tok string) (State, error)
	eof() error

	addNextState(s State)
}

var _ State = &state{}

// ParsedToken - Structure sent to the callback everytime a new ParsedToken is parsed
type ParsedToken struct {
	// tokenName - the name of this Token
	tokenName string
	// family - the family assigned to this Token. Used when something needs to be done for each Token of the same family
	family string
	// value - the value of the Token
	value string
}

// state - internal structure defining a state
type state struct {
	tokenName string
	family    string
	// acceptPattern - pattern used to decide if the current character can be accepted as part of this Token Value
	acceptPattern string

	// last - this is set to true if this Token can be the last Token (just before the EOF)
	last bool

	// isEOF - used internally to define the END Token. Not to be used by other tokens
	isEOF bool

	// next - the list of valid transitions from this state
	next []State

	// onNewToken - handler to be invoked when a Token has been successfully parsed
	onNewToken func(token *ParsedToken) error
}

// NewStartState ...
func NewStartState() State {
	return &state{
		tokenName:     "START",
		acceptPattern: `^$`,
	}
}

// NewEndState ...
func NewEndState() State {
	return &state{
		tokenName: "END",
		isEOF:     true,
	}
}

func (s *state) accept(tok string) bool {
	matched, _ := regexp.Match(s.acceptPattern, []byte(tok))
	return matched
}

func (s *state) parse(tok string) (State, error) {
	for _, next := range s.next {
		if next.accept(tok) {
			// valid Value
			if next.(*state).onNewToken != nil {
				if err := next.(*state).onNewToken(&ParsedToken{
					tokenName: next.(*state).tokenName,
					family:    next.(*state).family,
					value:     tok,
				}); err != nil {
					return nil, err
				}
			}
			return next, nil
		}
	}

	return nil, errors.Errorf("Unexpected Token `%s`", tok)
}

// eof - this function must be called when the whole string has been parsed to check if the current state is a valid eof state
func (s *state) eof() error {
	// EOF has been reached. Check if the current Token can be the last one
	if !s.last {
		return errors.Errorf(`EOF encountered while parsing string`)
	}

	return nil
}

func (s *state) addNextState(next State) {
	n := next.(*state)
	if n.isEOF {
		// if the passed in next state is an eof state, means this is a valid 'last' state
		// Just save the info and discard the 'next' state
		s.last = true
	} else {
		s.next = append(s.next, next)
	}
}

// StateBuilder - builder of State objects
type StateBuilder interface {
	Family(family string) StateBuilder
	AcceptPattern(acceptRegex string) StateBuilder
	OnNewToken(handler func(token *ParsedToken) error) StateBuilder
	Build() State
}

type stateBuilder struct {
	s *state
}

var _ StateBuilder = &stateBuilder{}

// Family ...
func (sb *stateBuilder) Family(family string) StateBuilder {
	sb.s.family = family
	return sb
}

// AcceptPattern ...
func (sb *stateBuilder) AcceptPattern(acceptPattern string) StateBuilder {
	sb.s.acceptPattern = acceptPattern
	return sb
}

// OnNewToken ...
func (sb *stateBuilder) OnNewToken(handler func(token *ParsedToken) error) StateBuilder {
	sb.s.onNewToken = handler
	return sb
}

// Build ...
func (sb *stateBuilder) Build() State {
	sb.s.acceptPattern = fmt.Sprintf(`^%s$`, sb.s.acceptPattern)
	return sb.s
}

// NewStateBuilder ...
func NewStateBuilder(tokenName string) StateBuilder {
	return &stateBuilder{s: &state{
		last:      false,
		tokenName: tokenName,
	}}
}
