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

func TokensAreEqual(t1, t2 *Token) bool {
	return t1.tType == t2.tType && t1.content == t2.content && t1.location == t2.location
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
