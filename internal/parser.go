package internal

import "fmt"

var tokens *[]Token
var tokenIdxPtr int = 0

func consumeCurrentToken() {
	tokenIdxPtr++
	Debug("Token taken", "PARSER")
}

func Parse(tokenStream []Token, programNum int) {
	// recover from error (will pass it up to lexer, then main)
	defer func() {
		if r := recover(); r != nil {
			CriticalError("parser", r)
		}
	}()

	Info(fmt.Sprintf("Parsing program %d", programNum+1), "GOPILER", true)
	tokens = &tokenStream
	parseProgram()
}

func parseProgram() {
	parseBlock()
	// match EOP
}

func parseBlock() {
	// match open paren
	parseStatementList()
	// match close paren
}

func parseStatementList() {
	parseStatement()
	parseStatementList()
	epsilonProduction()
}

func parseStatement() {

}

func parsePrintStatement() {
	// match PRINT
	// match (
	parseExpr()
	// match )
}

func parseExpr() {

}

func epsilonProduction() {
	/* This is an epsilon production
	Nothing will occur here.
	Implemented for code readability */
}
