package services

import "github.com/pkg/errors"

// Op ...
const (
	Op = iota
	Brace
	Literal
	QuotedLiteral
	NoToken
)

// Token ...
type Token struct {
	TokenType int
	Value     string
	Position  int
}

// Scanner ...
type Scanner interface {
	// Next - Move to the next Token. Return false if no next Token are available
	Next() bool
	// Peek - Look at the next Token without moving. Return false if no next Token are available
	Peek() (bool, *Token)
	// Token - Return the current Token Value. Panics if current Position is invalid.
	Token() *Token
	// Init - Initialise the scanner with the given string
	Init(s string)
}

type scanner struct {
	tokens []Token
	pos    int
}

var _ Scanner = &scanner{}

// Init ...
func (s *scanner) Init(txt string) {
	var tokens []Token
	currentTokenType := NoToken

	quoted := false
	escaped := false

	sendCurrentTokens := func() {
		res := ""
		for _, token := range tokens {
			res += token.Value
		}
		if res != "" {
			s.tokens = append(s.tokens,
				Token{
					TokenType: currentTokenType,
					Value:     res,
					Position:  tokens[0].Position,
				},
			)
		}
		tokens = nil
		currentTokenType = NoToken
	}

	// extract all the tokens from the string
	for i, currentChar := range txt {
		switch currentChar {
		case ' ':
			if quoted {
				tokens = append(tokens, Token{
					TokenType: Literal,
					Value:     " ",
					Position:  i,
				})
			} else {
				sendCurrentTokens()
			}
		case '(':
			fallthrough
		case ')':
			// found closebrace Token
			sendCurrentTokens()
			s.tokens = append(s.tokens, Token{
				TokenType: Brace,
				Value:     string(currentChar),
				Position:  i,
			})
		case '=':
			fallthrough
		case '<':
			fallthrough
		case '>':
			// found op Token
			if currentTokenType != NoToken && currentTokenType != Op {
				sendCurrentTokens()
			}
			tokens = append(tokens, Token{
				TokenType: Op,
				Value:     string(currentChar),
				Position:  i,
			})
			currentTokenType = Op
		case '\\':
			if quoted {
				escaped = true
				tokens = append(tokens, Token{
					TokenType: QuotedLiteral,
					Value:     "\\",
					Position:  i,
				})
			} else {
				if currentTokenType != NoToken && currentTokenType != Literal && currentTokenType != QuotedLiteral {
					sendCurrentTokens()
				}
				currentTokenType = Literal
				tokens = append(tokens, Token{
					TokenType: Literal,
					Value:     `\`,
					Position:  i,
				})
			}
		case '\'':
			if quoted {
				tokens = append(tokens, Token{
					TokenType: QuotedLiteral,
					Value:     "'",
					Position:  i,
				})
				if !escaped {
					sendCurrentTokens()
					quoted = false
					currentTokenType = NoToken
				}
				escaped = false
			} else {
				sendCurrentTokens()
				quoted = true
				currentTokenType = QuotedLiteral
				tokens = append(tokens, Token{
					TokenType: Op,
					Value:     "'",
					Position:  i,
				})
			}
			// none of the previous: LITERAL
		default:
			if currentTokenType != NoToken && currentTokenType != Literal && currentTokenType != QuotedLiteral {
				sendCurrentTokens()
			}
			currentTokenType = Literal
			tokens = append(tokens, Token{
				TokenType: Literal,
				Value:     string(currentChar),
				Position:  i,
			})
		}
	}

	sendCurrentTokens()
}

// Next ...
func (s *scanner) Next() bool {
	if s.pos < (len(s.tokens) - 1) {
		s.pos++
		return true
	}
	return false
}

// Peek ...
func (s *scanner) Peek() (bool, *Token) {
	if s.pos < (len(s.tokens) - 1) {
		return true, &s.tokens[s.pos+1]
	}
	return false, nil
}

// Token ...
func (s *scanner) Token() *Token {
	if s.pos < 0 || s.pos >= len(s.tokens) {
		panic(errors.Errorf("Invalid scanner Position %d", s.pos))
	}
	return &s.tokens[s.pos]
}

// NewScanner ...
func NewScanner() Scanner {
	return &scanner{
		pos: -1,
	}
}
