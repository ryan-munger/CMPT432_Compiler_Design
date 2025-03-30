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
	bufferFlag bool = false
	exprParent *Node
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
func collapseCharList(buffer []*Node) *Node {
	var collapsedStr string = ""
	for _, charNode := range buffer {
		collapsedStr += charNode.Token.trueContent
	}

	var collapsedCharToken Token = Token{
		tType: Character,
		location: Location{ // use the first char for pos data
			line:     buffer[0].Token.location.line,
			startPos: buffer[0].Token.location.startPos,
		},
		content:     "STRING",
		trueContent: collapsedStr,
	}

	return NewNode("Token", &collapsedCharToken)
}

func clearNodeBuffer() {
	nodeBuffer = []*Node{}
}

func dumpNodeBuffer() {
	for _, node := range nodeBuffer {
		curParent.AddChild(node)
	}
	clearNodeBuffer()
}

func evaluateExprNodeBuffer() *Node {
	if len(nodeBuffer) == 1 { // just a digit or ID - not a full expr
		var singleReturn *Node = nodeBuffer[0]
		clearNodeBuffer()
		return singleReturn
	}

	var stringMode bool = false
	var stringNodeBuffer []*Node
	var currentParent *Node

	for _, node := range nodeBuffer {
		switch node.Token.content {
		case "QUOTE":
			if stringMode {
				// End of string: collapse nodes
				concatNode := collapseCharList(stringNodeBuffer)
				if currentParent != nil {
					currentParent.AddChild(concatNode)
				} else {
					// if no parent set yet, we just have a string
					// nothing else can follow so don't worry about it having kids
					currentParent = concatNode
				}
				stringNodeBuffer = nil
			}
			stringMode = !stringMode

		// case "OPEN_PAREN":

		// case "CLOSE_PAREN":

		// case "EQUAL_OP", "N-EQUAL_OP", "ADD":

		default:
			if stringMode {
				stringNodeBuffer = append(stringNodeBuffer, node)
			}
			// } else if currentParent != nil {
			// 	currentParent.AddChild(node)
			// } else {
			// 	currentParent = node
			// }
		}
	}

	clearNodeBuffer()
	if currentParent == nil {
		return &Node{Type: "EMPTY"} // Prevent returning nil
	}
	return currentParent
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
	// end of expr
	if len(nodeBuffer) > 0 && (node.Type == "<StatementList>" || node.Type == "<Block>") {
		bufferFlag = false

		var exprTree *Node = evaluateExprNodeBuffer()
		exprParent.AddChild(exprTree)
		println("---------------------------\n")
	}
	// Handle different types of nodes
	switch node.Type {
	case "<Block>", "<PrintStatement>", "<AssignmentStatement>", "<VarDecl>",
		"<WhileStatement>", "<IfStatement>":
		importantNodeAbstraction(node)
	case "<IntExpr>":
		transformExpr(node)
	case "<StringExpr>":
		transformExpr(node)
	case "<BooleanExpression>":
		transformExpr(node)
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
	exprParent = importantNode

	for _, child := range node.Children {
		extractEssentials(child)
	}

	// Restore previous parent
	curParent = prevParent
}

// add everything to buffer so we can fix orderings
func transformExpr(node *Node) {
	bufferFlag = true
	for _, child := range node.Children {
		extractEssentials(child)
	}
}

// individual token
func transformToken(node *Node) {
	var tokenNode *Node = CopyNode(node)

	// allow parens, etc into buffer to help expr parsing
	if node.Type == "Token" && bufferFlag && node.Token.content != "EPS" {
		nodeBuffer = append(nodeBuffer, tokenNode)
	} else if node.Type == "Token" && !isGarbage(node.Token.content) {
		curParent.AddChild(tokenNode)
	}
}
