package internal

import "fmt"

// careful: this slice will update the 2d tokenstream from lexer!
var tokens []Token
var liveTokenIdx int = 0
var liveToken Token
var parseError bool = false
var alternateWarning string
var pNum int // program num
var currentParent *Node

// hold on to the CSTs
var cstList []TokenTree

func consumeCurrentToken(lastToken ...bool) {
	// this is just go syntax for an optional argument (variadic arg -  really a slice of bools)
	endOfTokens := false
	if len(lastToken) > 0 {
		endOfTokens = lastToken[0]
	}

	Debug(fmt.Sprintf("\tFound terminal %s [ %s ] in token stream",
		tokens[liveTokenIdx].content, tokens[liveTokenIdx].trueContent), "PARSER")
	var newNode *Node = NewNode("Token", &tokens[liveTokenIdx])
	currentParent.AddChild(newNode)

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
	pNum = programNum
	tokens = tokenStream
	// starts at first token (pos 0)
	liveToken = tokens[liveTokenIdx]
	// start new CST for this program
	cstList = append(cstList, TokenTree{})

	parseProgram()

	if !parseError {
		Pass(fmt.Sprintf("Parser successfully evaluated program %d with no errors.", programNum+1), "PARSER")
		Info("Generated Concrete Syntax Tree (CST):", "GOPILER", true)
		cstList[programNum].PrintTree()
		SemanticAnalysis(cstList[programNum], tokenStream, programNum)
	} else {
		Fail("Parsing aborted due to an error.", "PARSER")
		cstList[programNum] = TokenTree{} // free memory from the CST since it cannot be used
		Info("Compilation halted due to parser error.", "GOPILER", true)
	}

	// reset global vars for next program
	liveTokenIdx = 0
	liveToken = Token{}
	parseError = false
	alternateWarning = ""
	// assign new empty slice (tokens no longer can update tokenStream)
	tokens = []Token{}
	currentParent = nil
}

// match Block, EOP
func parseProgram() {
	Debug("! Parsing at Program Level !", "PARSER")
	// start off our CST
	var progRootNode *Node = NewNode("<Program>", nil)
	cstList[pNum].rootNode = progRootNode
	currentParent = cstList[pNum].rootNode

	parseBlock()

	if parseError {
		return
	}
	currentParent = cstList[pNum].rootNode
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
	var blockNode *Node = NewNode("<Block>", nil)
	currentParent.AddChild(blockNode)
	currentParent = blockNode

	if liveToken.content == "OPEN_BRACE" && liveToken.tType == Symbol {
		consumeCurrentToken()
	} else {
		wrongToken("OPEN_BRACE [ { ]")
	}

	parseStatementList()

	if parseError {
		return
	}
	currentParent = blockNode
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
	var statementListNode *Node = NewNode("<StatementList>", nil)
	currentParent.AddChild(statementListNode)
	currentParent = statementListNode

	if liveToken.content == "CLOSE_BRACE" && liveToken.tType == Symbol {
		epsilonProduction()
	} else {
		parseStatement()

		if parseError {
			return
		} else {
			currentParent = statementListNode
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
	var statementNode *Node = NewNode("<Statement>", nil)
	currentParent.AddChild(statementNode)
	currentParent = statementNode

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
	currentParent = statementNode
}

// Match Print, Open Paren, Expr, Close Paren
func parsePrintStatement() {
	if parseError {
		return
	}
	Debug("! Parsing at PrintStatement Level !", "PARSER")
	var printStatementNode *Node = NewNode("<PrintStatement>", nil)
	currentParent.AddChild(printStatementNode)
	currentParent = printStatementNode

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
	currentParent = printStatementNode
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
	var exprNode *Node = NewNode("<Expr>", nil)
	currentParent.AddChild(exprNode)
	currentParent = exprNode

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
	var intExprNode *Node = NewNode("<IntExpr>", nil)
	currentParent.AddChild(intExprNode)
	currentParent = intExprNode

	if liveToken.content == "DIGIT" && liveToken.tType == Digit {
		consumeCurrentToken()
	} else {
		wrongToken("DIGIT [ 0-9 ]")
	}

	// this one is optional since just a digit will suffice
	currentParent = intExprNode
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
	var strExprNode *Node = NewNode("<StringExpr>", nil)
	currentParent.AddChild(strExprNode)
	currentParent = strExprNode

	if liveToken.content == "QUOTE" && liveToken.tType == Symbol {
		consumeCurrentToken()
	} else {
		wrongToken("QUOTE [ \" ]")
	}

	if parseError {
		return
	} else {
		currentParent = strExprNode
		parseCharList()
	}

	currentParent = strExprNode
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
	var charListNode *Node = NewNode("<CharList>", nil)
	currentParent.AddChild(charListNode)
	currentParent = charListNode

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
	var assignNode *Node = NewNode("<AssignmentStatement>", nil)
	currentParent.AddChild(assignNode)
	currentParent = assignNode

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
		currentParent = assignNode
		parseExpr()
	}
}

// type, id
func parseVarDecl() {
	if parseError {
		return
	}
	Debug("! Parsing at VarDecl Level !", "PARSER")
	var declNode *Node = NewNode("<VarDecl>", nil)
	currentParent.AddChild(declNode)
	currentParent = declNode

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
	var whileNode *Node = NewNode("<WhileStatement>", nil)
	currentParent.AddChild(whileNode)
	currentParent = whileNode

	if liveToken.content == "KEYW_WHILE" && liveToken.tType == Keyword {
		consumeCurrentToken()
	} else {
		wrongToken("KEYW_WHILE [ while ]")
	}

	if parseError {
		return
	} else {
		currentParent = whileNode
		parseBooleanExpr()
	}

	if parseError {
		return
	} else {
		currentParent = whileNode
		parseBlock()
	}
}

// [(, Expr, boolop, Expr, )] | boolval
func parseBooleanExpr() {
	if parseError {
		return
	}
	Debug("! Parsing at BooleanExpression Level !", "PARSER")
	var boolExprNode *Node = NewNode("<BooleanExpression>", nil)
	currentParent.AddChild(boolExprNode)
	currentParent = boolExprNode

	if liveToken.content == "OPEN_PAREN" && liveToken.tType == Symbol {
		consumeCurrentToken()

		parseExpr()

		if parseError {
			return
		} else {
			currentParent = boolExprNode
			parseBoolOp()
		}

		if parseError {
			return
		} else {
			currentParent = boolExprNode
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
	var boolOpNode *Node = NewNode("<BoolOp>", nil)
	currentParent.AddChild(boolOpNode)
	currentParent = boolOpNode

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
	var boolValNode *Node = NewNode("<BoolVal>", nil)
	currentParent.AddChild(boolValNode)
	currentParent = boolValNode

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
	var ifNode *Node = NewNode("<IfStatement>", nil)
	currentParent.AddChild(ifNode)
	currentParent = ifNode

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
		currentParent = ifNode
		parseBlock()
	}
}

func epsilonProduction() {
	/* This is an epsilon production
	No real work will occur here.
	Implemented for code readability */
	Debug(fmt.Sprintf("\tEpsilon [ %c ] production", '\u03B5'), "PARSER")
	var epsToken = Token{
		tType:       Symbol,
		content:     "EPS",
		trueContent: "\u03B5",
	}
	var newNode *Node = NewNode("Token", &epsToken)
	currentParent.AddChild(newNode)
}
