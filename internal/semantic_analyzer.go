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
	case "<Block>":
		transformBlock(node)
	case "<PrintStatement>":
		transformPrintStatement(node)
	case "<VarDecl>":
		transformVarDecl(node)
	case "<AssignmentStatement>":
		transformAssignmentStatement(node)
	case "<WhileStatement>":
		transformWhileStatement(node)
	case "<IfStatement>":
		transformIfStatement(node)
	case "<IntExpr>":
		transformIntExpr(node)
	case "<StringExpr>":
		transformStringExpr(node)
	case "<BoolExpr>":
		transformBooleanExpr(node)
	case "Token":
		transformToken(node)
	default:
		// Recursively process children if not transformed
		for _, child := range node.Children {
			extractEssentials(child)
		}
	}
}

// Transform Block node
func transformBlock(node *Node) *Node {
	blockNode := NewNode("<Block>", nil)
	curParent.AddChild(blockNode)

	// Temporarily set current parent to the new block node
	prevParent = curParent
	curParent = blockNode

	// Process children of the block
	for _, child := range node.Children {
		extractEssentials(child)
	}

	// done with block
	curParent = prevParent

	return blockNode
}

// Transform Variable Declaration
func transformVarDecl(node *Node) {
	varDeclNode := NewNode("<VarDecl>", nil)
	curParent.AddChild(varDeclNode)

	prevParent = curParent
	curParent = varDeclNode

	// Collect type and ID
	for _, child := range node.Children {
		extractEssentials(child)
	}

	// Restore previous parent
	curParent = prevParent
}

// Transform Assignment Statement
func transformAssignmentStatement(node *Node) {
	assignNode := NewNode("<AssignmentStatement>", nil)
	curParent.AddChild(assignNode)

	prevParent = curParent
	curParent = assignNode

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

func transformBooleanExpr(node *Node) {

}

func transformPrintStatement(node *Node) {

}

func transformIfStatement(node *Node) {

}

func transformWhileStatement(node *Node) {

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
