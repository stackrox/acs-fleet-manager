package services

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

var validColumns = []string{"region", "name", "cloud_provider", "status", "owner"}

// BraceTokenFamily ...
const (
	BraceTokenFamily       = "BRACE"
	OpTokenFamily          = "OP"
	LogicalOpTokenFamily   = "LOGICAL"
	ColumnTokenFamily      = "COLUMN"
	ValueTokenFamily       = "VALUE"
	QuotedValueTokenFamily = "QUOTED"

	OpenBrace   = "OPEN_BRACE"
	ClosedBrace = "CLOSED_BRACE"
	Column      = "COLUMN"
	Value       = "VALUE"
	QuotedValue = "QUOTED_VALUE"
	Eq          = "EQ"
	NotEq       = "NOT_EQ"
	LikeState   = "LIKE"
	AndState    = "AND"
	OrState     = "OR"
)

// MaximumComplexity ...
const MaximumComplexity = 10

type checkUnbalancedBraces func() error

// DBQuery ...
type DBQuery struct {
	Query        string
	Values       []interface{}
	ValidColumns []string
}

// QueryParser ...
type QueryParser interface {
	Parse(sql string) (*DBQuery, error)
}

type queryParser struct {
	dbqry DBQuery
}

var _ QueryParser = &queryParser{}

// initStateMachine
// This will be our grammar (each Token will eat the spaces after the Token itself):
// Tokens:
// OPEN_BRACE       = (
// CLOSED_BRACE     = )
// COLUMN -         = [A-Za-z][A-Za-z0-9_]*
// VALUE            = [^ ^(^)]+
// QUOTED_VALUE     = `'([^']|\\')*'`
// EQ               = =
// NOT_EQ           = <>
// LIKE             = [Ll][Ii][Kk][Ee]
// AND              = [Aa][Nn][Dd]
// OR               = [Oo][Rr]
//
// VALID TRANSITIONS:
// START        -> COLUMN | OPEN_BRACE
// OPEN_BRACE   -> OPEN_BRACE | COLUMN
// COLUMN       -> EQ | NOT_EQ | LIKE
// EQ           -> VALUE | QUOTED_VALUE
// NOT_EQ       -> VALUE | QUOTED_VALUE
// LIKE         -> VALUE | QUOTED_VALUE
// VALUE        -> OR | AND | CLOSED_BRACE | [END]
// QUOTED_VALUE -> OR | AND | CLOSED_BRACE | [END]
// CLOSED_BRACE -> OR | AND | CLOSED_BRACE | [END]
// AND          -> COLUMN | OPEN_BRACE
// OR           -> COLUMN | OPEN_BRACE
func (p *queryParser) initStateMachine() (State, checkUnbalancedBraces) {

	// counts the number of joins
	complexity := 0

	contains := func(s []string, value string) bool {
		for _, item := range s {
			if item == value {
				return true
			}
		}
		return false
	}

	// This variable counts the open openBraces
	openBraces := 0
	countOpenBraces := func(tok string) error {
		switch tok {
		case "(":
			openBraces++
		case ")":
			openBraces--
		}
		if openBraces < 0 {
			return errors.Errorf("unexpected ')'")
		}
		return nil
	}

	onNewToken := func(token *ParsedToken) error {
		switch token.family {
		case BraceTokenFamily:
			if err := countOpenBraces(token.value); err != nil {
				return err
			}
			p.dbqry.Query += token.value
			return nil
		case ValueTokenFamily:
			p.dbqry.Query += " ?"
			p.dbqry.Values = append(p.dbqry.Values, token.value)
			return nil
		case QuotedValueTokenFamily:
			p.dbqry.Query += " ?"
			// unescape
			tmp := strings.ReplaceAll(token.value, `\'`, "'")
			// remove quotes:
			if len(tmp) > 1 {
				tmp = string([]rune(tmp)[1 : len(tmp)-1])
			}
			p.dbqry.Values = append(p.dbqry.Values, tmp)
			return nil
		case LogicalOpTokenFamily:
			complexity++
			if complexity > MaximumComplexity {
				return errors.Errorf("maximum number of permitted joins (%d) exceeded", MaximumComplexity)
			}
			p.dbqry.Query += " " + token.value + " "
			return nil
		case ColumnTokenFamily:
			// we want column names to be lowercase
			columnName := strings.ToLower(token.value)
			if !contains(p.dbqry.ValidColumns, columnName) {
				return fmt.Errorf("invalid column name: '%s'", token.value)
			}
			p.dbqry.Query += columnName
			return nil
		default:
			p.dbqry.Query += " " + token.value
			return nil
		}
	}

	grammar := Grammar{
		Tokens: []TokenDefinition{
			{Name: OpenBrace, Family: BraceTokenFamily, AcceptPattern: `\(`},
			{Name: ClosedBrace, Family: BraceTokenFamily, AcceptPattern: `\)`},
			{Name: Column, Family: ColumnTokenFamily, AcceptPattern: `[A-Za-z][A-Za-z0-9_]*`},
			{Name: Value, Family: ValueTokenFamily, AcceptPattern: `[^'][^ ^(^)]*`},
			{Name: QuotedValue, Family: QuotedValueTokenFamily, AcceptPattern: `'([^']|\\')*'`},
			{Name: Eq, Family: OpTokenFamily, AcceptPattern: `=`},
			{Name: NotEq, Family: OpTokenFamily, AcceptPattern: `<>`},
			{Name: LikeState, Family: OpTokenFamily, AcceptPattern: `[Ll][Ii][Kk][Ee]`},
			{Name: AndState, Family: LogicalOpTokenFamily, AcceptPattern: `[Aa][Nn][Dd]`},
			{Name: OrState, Family: LogicalOpTokenFamily, AcceptPattern: `[Oo][Rr]`},
		},
		Transitions: []TransitionDefinition{
			{TokenName: StartState, ValidTransitions: []string{Column, OpenBrace}},
			{TokenName: OpenBrace, ValidTransitions: []string{Column, OpenBrace}},
			{TokenName: Column, ValidTransitions: []string{Eq, NotEq, LikeState}},
			{TokenName: Eq, ValidTransitions: []string{QuotedValue, Value}},
			{TokenName: NotEq, ValidTransitions: []string{QuotedValue, Value}},
			{TokenName: LikeState, ValidTransitions: []string{QuotedValue, Value}},
			{TokenName: QuotedValue, ValidTransitions: []string{OrState, AndState, ClosedBrace, EndState}},
			{TokenName: Value, ValidTransitions: []string{OrState, AndState, ClosedBrace, EndState}},
			{TokenName: ClosedBrace, ValidTransitions: []string{OrState, AndState, ClosedBrace, EndState}},
			{TokenName: AndState, ValidTransitions: []string{Column, OpenBrace}},
			{TokenName: OrState, ValidTransitions: []string{Column, OpenBrace}},
		},
	}

	start := NewStateMachineBuilder(&grammar).
		OnNewToken(onNewToken).
		Build()

	return start, func() error {
		if openBraces > 0 {
			return fmt.Errorf("EOF while searching for closing brace ')'")
		}

		return nil
	}
}

// Parse ...
func (p *queryParser) Parse(sql string) (*DBQuery, error) {
	state, checkBalancedBraces := p.initStateMachine()

	scanner := NewScanner()
	scanner.Init(sql)

	for scanner.Next() {
		next, err := state.parse(scanner.Token().Value)
		if err != nil {
			return nil, errors.Errorf("[%d] error parsing the filter: %v", scanner.Token().Position+1, err)
		}
		state = next
	}

	if err := state.eof(); err != nil {
		return nil, err
	}

	if err := checkBalancedBraces(); err != nil {
		return nil, err
	}
	p.dbqry.Query = strings.Trim(p.dbqry.Query, " ")
	return &p.dbqry, nil
}

// NewQueryParser ...
func NewQueryParser(columns ...string) QueryParser {
	query := DBQuery{}
	if len(columns) == 0 {
		query.ValidColumns = validColumns
	} else {
		query.ValidColumns = columns
	}
	return &queryParser{dbqry: query}
}
