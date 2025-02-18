package internal

import "fmt"

// careful: this slice will update the 2d tokenstream from lexer!
var tokens []Token
var liveTokenIdx int = 0
var liveToken Token
var parseError bool = false
var alternateWarning string

var cstList []TokenTree
var currentCST TokenTree

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
	Error(fmt.Sprintf("Error at (%d:%d). Expected %s. Found %s [ %s ]. %s",
		liveToken.location.line, liveToken.location.startPos, expected,
		liveToken.content, liveToken.trueContent, alternateWarning), "PARSER")
	parseError = true
	alternateWarning = ""
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
	// start new CST for this program
	currentCST = TokenTree{}

	parseProgram()

	// save off CST
	cstList = append(cstList, currentCST)

	if !parseError {
		Pass(fmt.Sprintf("Parser successfully evaluated program %d with no errors.", programNum+1), "PARSER")
		// currentCST.PrintTree()
		SemanticAnalysis(cstList[programNum], tokenStream, programNum)
	} else {
		Fail("Parsing aborted due to an error.", "PARSER")
		Info("Compilation halted due to parser error.", "GOPILER", true)
	}

	// reset global vars for next program
	liveTokenIdx = 0
	liveToken = Token{}
	parseError = false
	alternateWarning = ""
	// assign new empty slice (tokens no longer can update tokenStream)
	tokens = []Token{}
	currentCST = TokenTree{}
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
	if liveToken.content == "CLOSE_BRACE" && liveToken.tType == Symbol {
		epsilonProduction()
	} else {
		parseStatement()

		if parseError {
			return
		} else {
			parseStatementList()
		}
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
		parseIfStatement()
	} else {
		parseBlock()
	}
}

// Match Print, Open Paren, Expr, Close Paren
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

// IntExpr | StringExpr | BooleanExpr | ID
func parseExpr() {
	if parseError {
		return
	}
	Debug("! Parsing at Expression Level !", "PARSER")

	if liveToken.content == "DIGIT" && liveToken.tType == Digit {
		parseIntExpr()
	} else if liveToken.content == "QUOTE" && liveToken.tType == Symbol {
		parseStringExpr()
	} else if (liveToken.content == "OPEN_PAREN" && liveToken.tType == Symbol) ||
		((liveToken.content == "KEYW_TRUE" || liveToken.content == "KEYW_FALSE") && liveToken.tType == Keyword) {
		parseBooleanExpr()
	} else if liveToken.content == "ID" && liveToken.tType == Identifier {
		consumeCurrentToken()
	} else {
		wrongToken("ID [ char ], IntExpr, StringExpr, or BooleanExpr")
	}
}

// [digit, intop, Expr] | digit
func parseIntExpr() {
	if parseError {
		return
	}
	Debug("! Parsing at IntExpr Level !", "PARSER")

	if liveToken.content == "DIGIT" && liveToken.tType == Digit {
		consumeCurrentToken()
	} else {
		wrongToken("DIGIT [ 0-9 ]")
	}

	// this one is optional since just a digit will suffice
	if parseError {
		return
	} else if liveToken.content == "ADD" && liveToken.tType == Symbol {
		consumeCurrentToken()

		parseExpr()
	} else if liveToken.content == "DIGIT" && liveToken.tType == Digit {
		alternateWarning = "Hint: Possible missing ADD [ + ]."
	}
}

// ", charlist, "
func parseStringExpr() {
	if parseError {
		return
	}
	Debug("! Parsing at StringExpr Level !", "PARSER")

	if liveToken.content == "QUOTE" && liveToken.tType == Symbol {
		consumeCurrentToken()
	} else {
		wrongToken("QUOTE [ \" ]")
	}

	if parseError {
		return
	} else {
		parseCharList()
	}

	if parseError {
		return
	} else if liveToken.content == "QUOTE" && liveToken.tType == Symbol {
		consumeCurrentToken()
	} else {
		wrongToken("QUOTE [ \" ]")
	}
}

// [char, CharList], [space, CharList], epsilon
func parseCharList() {
	if parseError {
		return
	}
	Debug("! Parsing at CharList Level !", "PARSER")

	// char includes space and chars
	if liveToken.content == "CHAR" && liveToken.tType == Character {
		consumeCurrentToken()
		parseCharList()
	} else {
		epsilonProduction()
	}
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

// if, BooleanExpr, Block
func parseIfStatement() {
	if parseError {
		return
	}
	Debug("! Parsing at IfStatement Level !", "PARSER")

	if liveToken.content == "KEYW_IF" && liveToken.tType == Keyword {
		consumeCurrentToken()
	} else {
		wrongToken("KEYW_IF [ if ]")
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

func epsilonProduction() {
	/* This is an epsilon production
	Nothing will occur here.
	Implemented for code readability */
}
