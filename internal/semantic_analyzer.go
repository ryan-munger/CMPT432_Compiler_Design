package internal

import (
	"errors"
	"fmt"
	"strings"
)

var (
	astList             []TokenTree
	curAst              *TokenTree
	curParent           *Node
	prevParent          *Node
	stringBuffer        []*Node
	symbolTableTreeList []*SymbolTableTree
	curSymbolTableTree  *SymbolTableTree
	curSymbolTable      *SymbolTable
	errorCount          int
	warnCount           int
)

func clearStringBuffer() {
	stringBuffer = []*Node{}
}

// Garbage tokens to filter out of AST
var GarbageMap = map[string]struct{}{
	"KEYW_PRINT":  {},
	"KEYW_WHILE":  {},
	"KEYW_IF":     {},
	"OPEN_BRACE":  {},
	"CLOSE_BRACE": {},
	"OPEN_PAREN":  {},
	"CLOSE_PAREN": {},
	"ASSIGN_OP":   {},
	"QUOTE":       {},
	"EPS":         {},
	"EOP":         {},
	"CHAR":        {},
}

// Helper function to check if a token is garbage
func isGarbage(candidate string) bool {
	_, exists := GarbageMap[candidate]
	return exists
}

// entry point
func SemanticAnalysis(cst TokenTree, programNum int) {
	defer func() {
		if r := recover(); r != nil {
			CriticalError("semantic analyzer", r)
		}
	}()

	Info(fmt.Sprintf("Semantically Analyzing program %d", programNum+1), "GOPILER", true)

	// build AST from cst
	Debug("Generating AST...", "SEMANTIC ANALYZER")
	initAst(programNum)
	buildAST(cst)
	Info(fmt.Sprintf("Program %d Abstract Syntax Tree (AST):\n%s\n%s", programNum+1, strings.Repeat("-", 75),
		curAst.drawTree()), "GOPILER", true)

	// perform semantic analysis
	Debug("Performing Scope and Type checks...", "SEMANTIC ANALYZER")
	initSymbolTableTree(programNum)
	scopeTypeCheck(curAst.rootNode) // recursive traversal starting from root

	issueUsageWarnings(curSymbolTableTree.rootTable) // recursive
	if errorCount == 0 {
		Pass(fmt.Sprintf("Successfully analyzed program %d with 0 errors and %d warning(s).",
			programNum+1, warnCount), "SEMANTIC ANALYZER")
		Info(fmt.Sprintf("Program %d Symbol Table:\n%s\n%s", programNum+1, strings.Repeat("-", 54),
			curSymbolTableTree.ToString()), "GOPILER", true)
		CodeGeneration(curAst, programNum)
	} else {
		Fail(fmt.Sprintf("Semantic Analysis failed with %d error(s) and %d warning(s).",
			errorCount, warnCount), "SEMANTIC ANALYZER")
		astList[programNum] = TokenTree{}                    // free memory from the AST since it cannot be used
		symbolTableTreeList[programNum] = &SymbolTableTree{} // free this up too
		Info(fmt.Sprintf("Compilation of program %d aborted due to semantic analysis error(s).",
			programNum+1), "GOPILER", false)
	}
}

// Initialize AST for a program
func initAst(pNum int) {
	for len(astList) <= pNum {
		astList = append(astList, TokenTree{})
	}
	curAst = &astList[pNum]
}

// start recursion
func buildAST(cst TokenTree) {
	curAst.rootNode = CopyNode(cst.rootNode)
	curParent = curAst.rootNode
	// Process children of the root
	for _, child := range cst.rootNode.Children {
		extractEssentials(child)
	}
}

// Recursive AST extraction
func extractEssentials(node *Node) {
	// Handle different types of nodes
	switch node.Type {
	case "<Block>", "<PrintStatement>", "<AssignmentStatement>", "<VarDecl>",
		"<WhileStatement>", "<IfStatement>":
		importantNodeAbstraction(node)
	case "<IntExpr>":
		transformIntExpr(node)
	case "<StringExpr>":
		transformStringExpr(node)
	case "<BooleanExpression>":
		transformBoolExpr(node)
	case "Token":
		transformToken(node)
	default:
		// the children of the node are important even if the node itself is not
		for _, child := range node.Children {
			extractEssentials(child)
		}
	}
}

// something important that we want an AST node and children for
func importantNodeAbstraction(node *Node) {
	importantNode := NewNode(node.Type, nil)
	curParent.AddChild(importantNode)

	prevParent = curParent
	curParent = importantNode

	for _, child := range node.Children {
		extractEssentials(child)
	}

	// Restore previous parent
	curParent = prevParent
}

// ensure intop (add) becomes the parent left-recursively
// we can have boolexprs and string exprs here - is valid for AST
func transformIntExpr(node *Node) {
	if len(node.Children) == 1 { // just an int
		extractEssentials(node.Children[0])
	} else { // we have an intop! 3 parts - digit intop expr
		var originalParent *Node = curParent

		var additionNode *Node = NewNode("<Addition>", nil)
		curParent.AddChild(additionNode)

		curParent = additionNode

		// add the digit as a child
		curParent.AddChild(CopyNode(node.Children[0].Children[0]))
		// examine the expression following
		extractEssentials(node.Children[2])

		// Restore previous parent
		curParent = originalParent
	}
}

// buffer chars and combine them
func transformStringExpr(node *Node) {
	for _, child := range node.Children {
		extractEssentials(child)
	}
	var concatNode *Node = collapseCharList()
	curParent.AddChild(concatNode)
}

// trust the parser!!! we can hardcode!!
func transformBoolExpr(node *Node) { // ( expr boolop expr ) | boolVal
	if len(node.Children) == 1 { // just a boolVal
		extractEssentials(node.Children[0])
	} else {
		var boolOpNode *Node
		if node.Children[2].Children[0].Token.content == "EQUAL_OP" {
			boolOpNode = NewNode("<Equality>", nil)
		} else { // N-EQUAL_OP
			boolOpNode = NewNode("<Inequality>", nil)
		}

		var originalParent *Node = curParent

		curParent.AddChild(boolOpNode)
		curParent = boolOpNode

		extractEssentials(node.Children[1]) // examine expr1
		extractEssentials(node.Children[3]) // examine expr2

		curParent = originalParent
	}
}

// individual token
func transformToken(node *Node) {
	var tokenNode *Node = CopyNode(node)

	if node.Type == "Token" && node.Token.tType == Character {
		stringBuffer = append(stringBuffer, node)
	} else if node.Type == "Token" && !isGarbage(node.Token.content) {
		curParent.AddChild(tokenNode)
	}
}

// empty buffer of char nodes into one node w a string value
func collapseCharList() *Node {
	var collapsedStr string = ""
	for _, charNode := range stringBuffer {
		collapsedStr += charNode.Token.trueContent
	}

	var collapsedCharToken Token = Token{
		tType: Character,
		location: Location{ // use the first char for pos data
			line:     stringBuffer[0].Token.location.line,
			startPos: stringBuffer[0].Token.location.startPos,
		},
		content:     "STRING",
		trueContent: collapsedStr,
	}

	clearStringBuffer()
	return NewNode("Token", &collapsedCharToken)
}

func scopeTypeCheck(node *Node) {
	switch node.Type {
	// case "<Block>":
	// 	// scope stuff!!
	case "<VarDecl>":
		analyzeVarDecl(node)
	case "<AssignmentStatement>":
		analyzeAssign(node)

	// intexpr, boolexprs can have type and id issues within
	case "<Addition>":
		analyzeAdd(node)
	case "<Equality>", "<Inequality>":
		analyzeEquality(node)

	// print an id
	case "Token":
		if node.Token.tType == Identifier {
			symbol, err := lookup(node.Token.trueContent, node.Token.location)
			if err == nil {
				symbol.beenUsed = true
			}
		}

	default:
		for _, child := range node.Children {
			scopeTypeCheck(child)
		}
	}
}

func initSymbolTableTree(pNum int) {
	for len(symbolTableTreeList) <= pNum {
		symbolTableTreeList = append(symbolTableTreeList, &SymbolTableTree{})
	}
	curSymbolTableTree = symbolTableTreeList[pNum]
	curSymbolTableTree.rootTable = NewSymbolTable("0")
	curSymbolTable = curSymbolTableTree.rootTable
}

// find an entry in accessible tables, pos is just for err reporting
func lookup(name string, pos Location) (*SymbolEntry, error) {
	var searchTable *SymbolTable = curSymbolTable
	for {
		if searchTable.EntryExists(name) {
			return searchTable.entries[name], nil

		} else if searchTable.parentTable != nil {
			searchTable = searchTable.parentTable // look above

		} else {
			Error(fmt.Sprintf("Undeclared variable (%d:%d): ID [ %s ] was used but not declared",
				pos.line, pos.startPos, name), "SEMANTIC ANALYZER")
			errorCount++
			return nil, errors.New("symbol not found")
		}
	}
}

func issueUsageWarnings(table *SymbolTable) {
	for _, entry := range table.entries {
		if !entry.isInit {
			Warn(fmt.Sprintf("ID [ %s ] from scope [ %s ] was declared but never initialized.",
				entry.name, table.scopeID), "SEMANTIC ANALYZER")
		} else if !entry.beenUsed {
			Warn(fmt.Sprintf("ID [ %s ] from scope [ %s ] was declared and initialized but never used.",
				entry.name, table.scopeID), "SEMANTIC ANALYZER")
		}
	}

	for _, subTable := range table.subTables {
		issueUsageWarnings(subTable)
	}
}

func typeMismatch(operation string, pos Location, leftType string, rightType string) {
	Error(fmt.Sprintf("Type mismatch on (%d:%d): cannot %s type [ %s ] to type [ %s ]",
		pos.line, pos.startPos, operation, rightType, leftType), "SEMANTIC ANALYZER")
}

// new symbol
// children of varDecl: type id
func analyzeVarDecl(node *Node) {
	var name string = node.Children[1].Token.trueContent
	var pos Location = node.Children[1].Token.location

	if curSymbolTable.EntryExists(name) {
		// id already used in this scope
		Error(fmt.Sprintf("Declaration Error on (%d:%d): ID [ %s ] is already declared in scope [ %s ].",
			pos.line, pos.startPos, name, curSymbolTable.scopeID), "SEMANTIC ANALYZER")
		errorCount++
	} else {
		var dType string = node.Children[0].Token.trueContent
		var entry *SymbolEntry = NewTableEntry(name, dType, pos)
		curSymbolTable.AddEntry(name, entry)
	}
}

func analyzeAssign(node *Node) {
	var assigneeNode *Node = node.Children[0]
	assignee, err := lookup(assigneeNode.Token.trueContent, node.Children[0].Token.location)
	// assignee does not exist, we are done here
	if err != nil {
		return
	}

	var assignError bool
	var assignTo *Node = node.Children[1]

	switch assignTo.Type {
	case "Token": // digit, str, id, bool
		if assignTo.Token.tType == Digit {
			if assignee.dataType != "int" {
				typeMismatch("assign", assigneeNode.Token.location, assignee.dataType, "int")
				assignError = true
			}
		} else if assignTo.Token.content == "STRING" {
			if assignee.dataType != "string" {
				typeMismatch("assign", assigneeNode.Token.location, assignee.dataType, "string")
				assignError = true
			}
		} else if assignTo.Token.tType == Identifier {
			assignedToSymbol, err := lookup(assignTo.Token.trueContent, assignTo.Token.location)
			if err != nil {
				return // id we are assigning to doesn't exist
			}

			if assignee.dataType != assignedToSymbol.dataType {
				typeMismatch("assign", assigneeNode.Token.location, assignee.dataType, assignedToSymbol.dataType)
				assignError = true
			}
		} else {
			if assignee.dataType != "boolean" {
				typeMismatch("assign", assigneeNode.Token.location, assignee.dataType, "boolean")
				assignError = true
			}
		}

	case "<Addition>": // int expr
		if assignee.dataType != "int" {
			typeMismatch("assign", assigneeNode.Token.location, assignee.dataType, "int")
			assignError = true
		}
		analyzeAdd(assignTo)

	case "<Equality>", "<Inequality>": // bool expr
		if assignee.dataType != "boolean" {
			typeMismatch("assign", assigneeNode.Token.location, assignee.dataType, "boolean")
			assignError = true
		}
		analyzeEquality(assignTo)
	}

	if assignError {
		errorCount++
	} else {
		assignee.isInit = true
	}
}

// easier bc no valid subexpressions that aren't add
// we don't even need to check the left side of intop because parser did (digit)
// right can be add, string, digit, id, bool, equality
func analyzeAdd(node *Node) {
	var leftAdd *Node = node.Children[0]
	var rightAdd *Node = node.Children[1]

	if rightAdd.Type == "<Addition>" {
		analyzeAdd(rightAdd)

		// id, digit, string
	} else if rightAdd.Type == "Token" {
		if rightAdd.Token.tType == Identifier {
			addSym, err := lookup(rightAdd.Token.trueContent, rightAdd.Token.location)
			if err != nil {
				return // id we are adding doesn't exist
			}

			if addSym.dataType != "int" {
				typeMismatch("add", leftAdd.Token.location, "int", addSym.dataType)
				errorCount++
			} else {
				addSym.beenUsed = true
			}
		} else if rightAdd.Token.content == "STRING" {
			typeMismatch("add", leftAdd.Token.location, "int", "string")
			errorCount++
		} else if rightAdd.Token.tType == Digit {
			// digit, no action needed
		} else { // boolean
			typeMismatch("add", leftAdd.Token.location, "int", "boolean")
			errorCount++
		}

	} else if rightAdd.Type == "<Equality>" || rightAdd.Type == "<Inequality>" {
		typeMismatch("add", leftAdd.Token.location, "int", "boolean")
		errorCount++
	}

}

func analyzeEquality(node *Node) {

}
