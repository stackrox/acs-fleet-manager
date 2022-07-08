# Parsing and validating complex strings
## Overview
To parse and validate a string, the first thing that should be done would be to understand **what** are the *tokens* that
compose our syntax.
After we have a full list of tokens, there is one more step that must be performed: understand what are the valid transitions
between the tokens.

By using a list of tokens and all the valid transitions between the tokens, validating a string according to a syntax/grammar will be possible

## Implementation
Implementing a state machine to parse and validate a string according to a specific grammar/syntax can be time-consuming and
error prone. For that reason, we provided some objects that simplifies the process:
* `Scanner`: this is the object responsible for splitting a string into a list of tokens
* `StateMachine`: this moves from one state to another according to the received token and the valid transitions from the current state
* `StateMachineBuilder`: receives a `Grammar` object and configure a `StateMachine` accordingly
* `QueryParser`: this is the only ad-hoc object. This configures the `StateMachine` to make it ready to be used
to parse the SQL subset we use in the dinosaur list endpoint

### The `Scanner`
The scanner object is the lowest level object of the parser.
Takes a string as input and splits it into tokens.
Recognized tokens are:
* **Braces**: open and closed round braces
* **Operator**. Recognized operator tokens are ‘=’,’<’,’>’ and any string composed by only operators, for example: ‘==’, ‘>=’, ‘>>><<<===’. The scanner doesn’t perform any validation: it simply recognises the token.
* **Quoted String**: any string surrounded by single quotes. Escaped single quotes are supported too and included into the quoted string (ie: ‘I\’m Massimiliano’ is a valid quoted string)
* **Literal**: a sequence of non spaces, non brace characters

Three public methods are provided:
* `Next`: move the internal index to the next available token and return `true` if EOF has not been reached
* `Token`: returns the current token
* `Peek`: if available, return the next token without updating the internal index

To loop through the tokens, this construct can be used:
```go
s := /// scanner
for s.Next() {
	t := s.Token()
	// deal with the token
}
```
### The `StateMachine`
The state machine is the most important component of the parser and is used to know what is the valid next token we should receive according to the current position.
To understand how it works, let's start with a very simple example:

    We need to validate a simple statement describing a multiplication or a division. In the case of the division,
    the dividend can't be 0.
    Valid statements:
        10x50
        20x2
    Invalid statements:
        10x
        x10
        10/0
Our tokens will be:
* `FIRST_OPERAND=[0-9]+` - Match one or more digit
* `SECOND_OPERAND=[0-9]+` - Match one or more digit
* `DIVIDEND=[1-9][0-9]*`  - Match one or more digit not starting with 0
* `MUL=x` - Match x
* `DIV=/` - Match /

The valid transitions between the tokens will be:
* START -> FIRST_OPERAND
* FIRST_OPERAND -> \[MUL,DIV\]
* MUL -> SECOND_OPERAND
* DIV -> DIVIDEND
* SECOND_OPERAND -> END
* DIVIDEND -> END

#### Example 1 - Valid Multiplication
Let’s try to follow that to validate our first valid example "20x300"

| CURRENT STATE | RECEIVED TOKEN | NEW STATE      |
|---------------|----------------|----------------|
| START         | 20             | FIRST_OPERAND  |
| FIRST_OPERAND | x              | MUL            |
| MUL           | 300            | SECOND_OPERAND |
| SECOND_OPERAND| EOF            | END            |

We reached the end state. The string is valid!.

#### Example 2 - Valid Division
Let’s try to validate the division example "50/2"

| CURRENT STATE | RECEIVED TOKEN | NEW STATE     |
|---------------|----------------|---------------|
| START         | 50             | FIRST_OPERAND |
| FIRST_OPERAND | /              | DIV           |
| DIV           | 2              | DIVIDEND      |
| DIVIDEND      | EOF            | END           |

We reached the end state. The string is valid!.

#### Example 3 - Division by ZERO
Let’s try a division by zero: "50/0"

| CURRENT STATE | RECEIVED TOKEN | NEW STATE     |
|---------------|----------------|---------------|
| START         | 50             | FIRST_OPERAND |
| FIRST_OPERAND | /              | DIV           |
| DIV           | 0              |               |


None of the valid transitions from the DIV state are able to accept the `0` value, so an error is returned.

#### Example 4 - Incomplete operation
Let’s validate an incomplete statement: "50/"

| CURRENT STATE | RECEIVED TOKEN | NEW STATE     |
|---------------|----------------|---------------|
| START         | 50             | FIRST_OPERAND |
| FIRST_OPERAND | /              | DIV           |
| DIV           | EOF            |               |

The DIV state doesn’t have a transition to END, so an error is returned

### The `StateMachineBuilder`
Configuring the statemachine can be boring and error prone.
To make the code more readable the `StateMachineBuilder` and `Grammar` couple have been created.
The `Grammar` simply defines the tokens that composes our language and the valid transitions between each token.

#### Example
Let’s keep on with the previous example.

Tokens:
* `FIRST_OPERAND=[0-9]+` - Match one or more digit
* `SECOND_OPERAND=[0-9]+` - Match one or more digit
* `DIVIDEND=[1-9][0-9]*`  - Match one or more digit not starting with 0
* `MUL=x` - Match x
* `DIV=/` - Match /

Transitions:
* START -> FIRST_OPERAND
* FIRST_OPERAND -> \[MUL,DIV\]
* MUL -> SECOND_OPERAND
* DIV -> DIVIDEND
* SECOND_OPERAND -> END
* DIVIDEND -> END

The `Grammar` object that we could use to configure our `StateMachine` to validate this string could be:
```go
grammar := Grammar{
    Tokens: []TokenDefinition{
        {Name: "FIRST_OPERAND", AcceptPattern: "[0-9]+"},
        {Name: "SECOND_OPERAND", AcceptPattern: "[0-9]+"},
        {Name: "DIVIDEND", AcceptPattern: "[1-9][0-9]*"},
        {Name:"MUL", AcceptPattern:"x"},
        {Name:"DIV", AcceptPattern:`\/`},
    },
    Transitions: []TransitionDefinition[]{
        {TokenName: StartState, ValidTransitions: []string{"FIRST_OPERAND"}},
        {TokenName: "FIRST_OPERAND", ValidTransitions: []string{"MUL","DIV"}},
        {TokenName: "MUL", ValidTransitions: []string{"SECOND_OPERAND"}},
        {TokenName: "DIV", ValidTransitions: []string{"DIVIDEND"}},
        {TokenName: "SECOND_OPERAND", ValidTransitions: []string{EndState}},
        {TokenName: "DIVIDEND", ValidTransitions: []string{EndState}},
    },
}

// Create the StateMachine
stateMachine := NewStateMachineBuilder(&grammar).Build()
```
Now that the `StateMachine` is ready, we can start parsing our strings.
An `onNewToken` handler function can be passed to the builder to receive tokens and perform custom operations/validations.

### The `QueryParser`

The QueryParser is the topmost object and hides all the configuration needed to parse the strings for our specific case.
It does the following:
* Creates a `Grammar` able to parse and validate our SQL subset
* Uses the Grammar and the StateMachineBuilder to create the StateMachine
* Passes an `onNewToken` handler to the builder to perform custom operations (limit the maximum number of joins,
  constraint the column names to a specified set of column names, etc)

By using the parser, parsing and validating our SQL subset becomes a one liner:

```go
qry := services.NewQueryParser().Parse(s)
```

Where:
* `s` is the string to be parsed
* `qry` is an object containing the parsed string and an array containing all the field values.
