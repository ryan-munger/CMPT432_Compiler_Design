package internal

import (
	"fmt"
	"strings"
)

var (
	astList             []TokenTree
	curAst              TokenTree
	curParent           *Node
	prevParent          *Node
	stringBuffer        []*Node
	symbolTableTreeList []*SymbolTableTree
	curSymbolTableTree  *SymbolTableTree
	curSymbolTable      *SymbolTable
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
func SemanticAnalysis(cst TokenTree, tokenStream []Token, programNum int) {
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
	Info(fmt.Sprintf("Program %d Symbol Table:\n%s\n%s", programNum+1, strings.Repeat("-", 52),
		curSymbolTableTree.ToString()), "GOPILER", true)
}

// Initialize AST for a program
func initAst(pNum int) {
	for len(astList) <= pNum {
		astList = append(astList, TokenTree{})
	}
	curAst = astList[pNum]
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

// Transform Assignment Statement
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

// add everything to buffer so we can fix orderings
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
	println(node.Type)

	switch node.Type {
	case "<VarDecl>":
		analyzeVarDecl(node)

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

// new symbol
// children of varDecl: type id
func analyzeVarDecl(node *Node) {
	var name string = node.Children[1].Token.trueContent
	if curSymbolTable.EntryExists(name) {
		// id already used in this scope
		// error here
	} else {
		var dType string = node.Children[0].Token.trueContent
		var pos Location = node.Children[1].Token.location
		var entry *SymbolEntry = NewTableEntry(name, dType, pos)
		curSymbolTable.AddEntry(name, entry)
	}

}
