package internal

// constrain tokentypes to the 5 values
type TokenType string

const (
	Keyword    TokenType = "keyword"
	Identifier TokenType = "identifier"
	Symbol     TokenType = "symbol"
	Digit      TokenType = "digit"
	Character  TokenType = "character"
)

type Location struct {
	line     int
	startPos int
}

type Token struct {
	tType    TokenType
	location Location
	content  string
}

var SymbolMap = map[rune]string{
	'{': "OPEN_BRACE",
	'}': "CLOSE_BRACE",
	'(': "OPEN_PAREN",
	')': "CLOSE_PAREN",
	'$': "EOP",
	'=': "ASSIGN_OP",
	'"': "QUOTE",
	'+': "ADD",
}
