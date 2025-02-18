package internal

import "fmt"

// note: this slice will update the 2d tokenstream from lexer!
var tokens []Token
var liveTokenIdx int = 0
var liveToken Token
var parseError bool = false

func consumeCurrentToken(lastToken ...bool) {
	// this is just go syntax for an optional argument (variadic arg -  really a slice of bools)
	endOfTokens := false
	if len(lastToken) > 0 {
		endOfTokens = lastToken[0]
	}

	Debug(fmt.Sprintf("\tFound terminal %s [ %s ] in token stream",
		tokens[liveTokenIdx].content, tokens[liveTokenIdx].trueContent), "PARSER")

	// don't go out of bounds
	if !endOfTokens {
		liveTokenIdx++
		liveToken = tokens[liveTokenIdx]
	}
}

func wrongToken(expected string) {
	Error(fmt.Sprintf("Error at (%d:%d). Expected %s. Found %s [ %s ].",
		liveToken.location.line, liveToken.location.startPos, expected,
		liveToken.content, liveToken.trueContent), "PARSER")
	parseError = true
}

func isTypeKeyword(candidate string) bool {
	_, exists := TypeMap[candidate]
	return exists
}

func Parse(tokenStream []Token, programNum int) {
	// recover from error (will pass it up to lexer, then main)
	defer func() {
		if r := recover(); r != nil {
			CriticalError("parser", r)
		}
	}()

	Info(fmt.Sprintf("Parsing program %d", programNum+1), "GOPILER", true)
	tokens = tokenStream
	// starts at first token (pos 0)
	liveToken = tokens[liveTokenIdx]
	parseProgram()

	if !parseError {
		Pass("Parser passed", "PARSER")
	} else {
		Fail("Parser failed", "PARSER")
	}
}

// match Block, EOP
func parseProgram() {
	Debug("! Parsing at Program Level !", "PARSER")
	parseBlock()

	if parseError {
		return
	}

	Debug("! Parsing at Program Level !", "PARSER")
	// don't consume if right as it is the end
	if liveToken.content == "EOP" {
		consumeCurrentToken(true)
	} else {
		wrongToken("EOP [ $ ]")
	}
}

// match Open Brace, StatementList, Close Brace
func parseBlock() {
	Debug("! Parsing at Block Level !", "PARSER")
	if liveToken.content == "OPEN_BRACE" && liveToken.tType == Symbol {
		consumeCurrentToken()
	} else {
		wrongToken("OPEN_BRACE [ { ]")
	}

	parseStatementList()

	if parseError {
		return
	}
	Debug("! Parsing at Block Level !", "PARSER")
	if liveToken.content == "CLOSE_BRACE" && liveToken.tType == Symbol {
		consumeCurrentToken()
	} else {
		wrongToken("CLOSE_BRACE [ { ]")
	}
}

// [statement statementList] or epsilon
func parseStatementList() {
	if parseError {
		return
	}

	Debug("! Parsing at StatementList Level !", "PARSER")
	parseStatement()
	// [statement statementList] didn't work out
	if parseError {
		parseError = false
		epsilonProduction()
	} else {
		parseStatementList()
	}
}

// PrintStatement | AssignmentStatement | VarDecl | WhileStatement | IfStatement | Block
func parseStatement() {
	if parseError {
		return
	}
	Debug("! Parsing at Statement Level !", "PARSER")
	if liveToken.content == "KEYW_PRINT" && liveToken.tType == Keyword {
		parsePrintStatement()
	} else if liveToken.content == "ID" && liveToken.tType == Identifier {
		parseAssignmentStatement()
	} else if isTypeKeyword(liveToken.trueContent) && liveToken.tType == Keyword {
		parseVarDecl()
	} else if liveToken.content == "KEYW_WHILE" && liveToken.tType == Keyword {
		parseWhileStatement()
	} else if liveToken.content == "KEYW_IF" && liveToken.tType == Keyword {

	} else if liveToken.content == "{" && liveToken.tType == Symbol {

	} else {
		wrongToken("")
	}

}

// Match Printz, Open Paren, Expr, Close Paren
func parsePrintStatement() {
	if parseError {
		return
	}

	Debug("! Parsing at PrintStatement Level !", "PARSER")
	if liveToken.content == "KEYW_PRINT" && liveToken.tType == Keyword {
		consumeCurrentToken()
	} else {
		wrongToken("KEYW_PRINT [ print ]")
	}

	if liveToken.content == "OPEN_PAREN" && liveToken.tType == Symbol {
		consumeCurrentToken()
	} else {
		wrongToken("OPEN_PAREN [ ( ]")
	}

	parseExpr()

	if parseError {
		return
	}
	Debug("! Parsing at PrintStatement Level !", "PARSER")
	if liveToken.content == "CLOSE_PAREN" && liveToken.tType == Symbol {
		consumeCurrentToken()
	} else {
		wrongToken("CLOSE_PAREN [ ) ]")
	}
}

func parseExpr() {
	Debug("! Parsing at Expression Level !", "PARSER")

}

// ID, =, Expr
func parseAssignmentStatement() {
	if parseError {
		return
	}

	Debug("! Parsing at AssignmentStatement Level !", "PARSER")
	if liveToken.content == "ID" && liveToken.tType == Identifier {
		consumeCurrentToken()
	} else {
		wrongToken("ID [ char ]")
	}

	if parseError {
		return
	} else if liveToken.content == "ASSIGN_OP" && liveToken.tType == Symbol {
		consumeCurrentToken()
	} else {
		wrongToken("ASSIGN_OP [ = ]")
	}

	if parseError {
		return
	} else {
		parseExpr()
	}
}

// type, id
func parseVarDecl() {
	if parseError {
		return
	}
	Debug("! Parsing at VarDecl Level !", "PARSER")

	if isTypeKeyword(liveToken.trueContent) && liveToken.tType == Keyword {
		consumeCurrentToken()
	} else {
		wrongToken("type keyword {I_TYPE [ int ], B_TYPE [ boolean ], S_TYPE [ string ]}")
	}

	if parseError {
		return
	} else if liveToken.content == "ID" && liveToken.tType == Identifier {
		consumeCurrentToken()
	} else {
		wrongToken("ID [ char ]")
	}
}

// while, BooleanExpr, Block
func parseWhileStatement() {
	if parseError {
		return
	}
	Debug("! Parsing at WhileStatement Level !", "PARSER")

	if liveToken.content == "KEYW_WHILE" && liveToken.tType == Keyword {
		consumeCurrentToken()
	} else {
		wrongToken("KEYW_WHILE [ while ]")
	}

	if parseError {
		return
	} else {
		parseBooleanExpr()
	}

	if parseError {
		return
	} else {
		parseBlock()
	}
}

// [(, Expr, boolop, Expr, )] | boolval
func parseBooleanExpr() {
	if parseError {
		return
	}
	Debug("! Parsing at BooleanExpression Level !", "PARSER")

	if liveToken.content == "OPEN_PAREN" && liveToken.tType == Symbol {
		consumeCurrentToken()

		parseExpr()

		if parseError {
			return
		} else {
			parseBoolOp()
		}

		if parseError {
			return
		} else {
			parseExpr()
		}

		if parseError {
			return
		} else if liveToken.content == "CLOSE_PAREN" && liveToken.tType == Symbol {
			consumeCurrentToken()
		} else {
			wrongToken("CLOSE_PAREN [ ) ]")
		}

	} else {
		parseBoolVal()
	}

}

// == | !=
func parseBoolOp() {
	if parseError {
		return
	}
	Debug("! Parsing at BoolOp Level !", "PARSER")

	if (liveToken.content == "EQUAL_OP" || liveToken.content == "N-EQUAL_OP") && liveToken.tType == Symbol {
		consumeCurrentToken()
	} else {
		wrongToken("token in: {EQUAL_OP [ == ], N-EQUAL_OP [ != ]}")
	}
}

// true | false
func parseBoolVal() {
	if parseError {
		return
	}
	Debug("! Parsing at BoolVal Level !", "PARSER")

	if (liveToken.content == "KEYW_TRUE" || liveToken.content == "KEYW_FALSE") && liveToken.tType == Keyword {
		consumeCurrentToken()
	} else {
		wrongToken("token in: {KEYW_TRUE [ true ], KEYW_FALSE [ false ]}")
	}
}

func epsilonProduction() {
	/* This is an epsilon production
	Nothing will occur here.
	Implemented for code readability */
}
