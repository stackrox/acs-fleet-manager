package services

const (
	StartState = "__$$_START_$$__"
	EndState   = "__$$_END_$$__"
)

type TokenDefinition struct {
	Name          string
	Family        string
	AcceptPattern string
	OnNewToken    func(token *ParsedToken) error
}

type TransitionDefinition struct {
	TokenName        string
	ValidTransitions []string
}

type Grammar struct {
	Tokens      []TokenDefinition
	Transitions []TransitionDefinition
}

type NewTokenHandler func(token *ParsedToken) error

type StateMachineBuilder interface {
	OnNewToken(handler NewTokenHandler) StateMachineBuilder
	Build() State
}

type stateMachineBuilder struct {
	grammar    *Grammar
	onNewToken NewTokenHandler
}

var _ StateMachineBuilder = &stateMachineBuilder{}

func NewStateMachineBuilder(grammar *Grammar) StateMachineBuilder {
	return &stateMachineBuilder{grammar: grammar}
}

func (smb *stateMachineBuilder) OnNewToken(handler NewTokenHandler) StateMachineBuilder {
	smb.onNewToken = handler
	return smb
}

func (smb *stateMachineBuilder) Build() State {

	tokenMap := make(map[string]State)

	tokenMap[StartState] = NewStartState()
	tokenMap[EndState] = NewEndState()

	// build all the tokens
	for _, t := range smb.grammar.Tokens {
		b := NewStateBuilder(t.Name).
			Family(t.Family).
			AcceptPattern(t.AcceptPattern).
			OnNewToken(smb.onNewToken)

		token := b.Build()
		tokenMap[t.Name] = token
	}

	// add all the transitions
	for _, transition := range smb.grammar.Transitions {
		token := tokenMap[transition.TokenName]
		for _, targetTokenName := range transition.ValidTransitions {
			targetToken := tokenMap[targetTokenName]
			token.addNextState(targetToken)
		}
	}

	return tokenMap[StartState]
}
