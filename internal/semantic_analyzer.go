package internal

import (
	"fmt"
	"strings"
)

var (
	astList    []TokenTree
	curAst     TokenTree
	curParent  *Node
	prevParent *Node
	nodeBuffer []*Node
)

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
}

// Helper function to check if a token is garbage
func isGarbage(candidate string) bool {
	_, exists := GarbageMap[candidate]
	return exists
}

// empty buffer of char nodes into one node w a string value
func collapseCharList() *Node {
	var collapsedStr string = ""
	for _, charNode := range nodeBuffer {
		collapsedStr += charNode.Token.trueContent
	}

	var collapsedCharToken Token = Token{
		tType: Character,
		location: Location{ // use the first char for pos data
			line:     nodeBuffer[0].Token.location.line,
			startPos: nodeBuffer[0].Token.location.startPos,
		},
		content:     "STRING",
		trueContent: collapsedStr,
	}

	clearNodeBuffer()
	return NewNode("Token", &collapsedCharToken)
}

func clearNodeBuffer() {
	nodeBuffer = []*Node{}
}

// Initialize AST for a program
func initAst(pNum int) {
	for len(astList) <= pNum {
		astList = append(astList, TokenTree{})
	}
	curAst = astList[pNum]
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
	//	fmt.Printf("Node: %s \n", node.Type)
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
	printNode := NewNode(node.Type, nil)
	curParent.AddChild(printNode)

	prevParent = curParent
	curParent = printNode

	for _, child := range node.Children {
		extractEssentials(child)
	}

	// Restore previous parent
	curParent = prevParent
}

func transformIntExpr(node *Node) {

}

func transformStringExpr(node *Node) {
	// extract essentials will handle charlist collapse
	for _, child := range node.Children {
		extractEssentials(child)
	}
	var concatNode *Node = collapseCharList()
	curParent.AddChild(concatNode)
}

func transformBoolExpr(node *Node) {
	println("Bool expr!")
}

// individual token
func transformToken(node *Node) {
	// don't add things like '{'
	// if char, we want to collapse those
	if node.Token != nil && !isGarbage(node.Token.content) {
		tokenNode := CopyNode(node)

		if node.Token.tType == Character {
			nodeBuffer = append(nodeBuffer, node)

		} else {
			curParent.AddChild(tokenNode)
		}
	}
}
