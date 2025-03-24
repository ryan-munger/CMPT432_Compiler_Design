package internal

import (
	"fmt"
	"strings"
)

// to store them in case needed later
var astList []TokenTree
var curAst TokenTree
var curParent *Node
var prevParent *Node
var mode string
var nodeBuffer []*Node
var charListBuffer string = ""

// stuff we do not want to see in AST ever
var GarbageMap map[string]struct{} = map[string]struct{}{
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
}

func isGarbage(candidate string) bool {
	_, exists := GarbageMap[candidate]
	return exists
}

func startAst(pNum int) {
	// if a program fails in lexer, it never even got to parse
	// we still need to index using program num though
	for len(astList) <= pNum {
		astList = append(astList, TokenTree{})
		curAst = astList[pNum]
		mode = "<Init>"
	}
}

func SemanticAnalysis(cst TokenTree, tokenStream []Token, programNum int) {
	// recover from error, pass it up to parser, lexer, main
	defer func() {
		if r := recover(); r != nil {
			CriticalError("semantic analyzer", r)
		}
	}()

	Info(fmt.Sprintf("Semantically Analyzing program %d", programNum+1), "GOPILER", true)

	// build AST
	Debug("Generating AST...", "SEMANTIC ANALYZER")
	startAst(programNum)
	buildAST(cst)
	Info(fmt.Sprintf("Program %d Abstract Syntax Tree (AST):\n%s\n%s", programNum+1, strings.Repeat("-", 75),
		curAst.drawTree()), "GOPILER", true)

	// use AST

}

func buildAST(cst TokenTree) {
	// explain
	extractEssentials(cst.rootNode)
}

func addAstNode(node *Node) {
	var newShallowCopy *Node = CopyNode(node)
	curParent.AddChild(newShallowCopy)

	// token leaves are never parents
	if node.Type != "Token" {
		prevParent = curParent
		curParent = newShallowCopy
	}
}

func moveUp() {
	mode = prevParent.Type // moveUp
	curParent = prevParent
}

// empty buffer of char nodes into one node w a string
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

	clearBuffer()
	return NewNode("Token", &collapsedCharToken)
}

func clearBuffer() {
	nodeBuffer = []*Node{}
}

func extractEssentials(node *Node) {
	switch mode {
	case "<Init>": // start the tree
		curAst.rootNode = CopyNode(node) // will always be <program> node
		curParent = curAst.rootNode
		mode = "<Block>"

	case "<Block>":
		if node.Type == "<Block>" || node.Type == "<PrintStatement>" || node.Type == "<VarDecl>" ||
			node.Type == "<AssignmentStatement>" || node.Type == "<WhileStatement>" || node.Type == "<IfStatement>" {
			addAstNode(node)
			mode = node.Type
		}

	case "<PrintStatement>", "<AssignmentStatement>": // these are surprisingly the same under the hood
		if node.Type == "<StatementList>" { // end of print or assign
			moveUp()
		} else if node.Type == "Token" && node.Token.content == "EPS" { // end of a charlist
			var collapsedCharNode *Node = collapseCharList()
			addAstNode(collapsedCharNode)

		} else if node.Type == "Token" && !isGarbage(node.Token.content) { // leave out the fluff!
			if node.Token.tType == Character { // need to collapse charList
				nodeBuffer = append(nodeBuffer, node)
			} else {
				addAstNode(node)
			}
		}

	case "<VarDecl>":
		if node.Type == "Token" {
			addAstNode(node)
		} else if len(curParent.Children) == 2 { // guaranteed from parser to have type and ID
			println(prevParent.Type)
			moveUp()
		}

	// case "<WhileStatement>", "<IfStatement>":
	// 	if node.Type == "<StatementList>" {
	// 		moveUp()
	// 	} else if node.Type == "Token" {
	// 		addAstNode(node)
	// 	} else if node.Type == "<Block>" {
	// 		addAstNode(node)
	// 	}

	default:
		mode = "<Block>"
		// skip the node
	}

	// if node.Token != nil {
	// 	if node.Token.trueContent == " " { // we have a token
	// 		fmt.Printf("{%s [ space ]}\n", node.Token.content)
	// 	} else {
	// 		fmt.Printf("{%s [ %s ]}\n", node.Token.content, node.Token.trueContent)
	// 	}
	// } else {
	// 	fmt.Println(node.Type) // non terminal
	// }

	for _, child := range node.Children {
		extractEssentials(child) // Recursively print children
	}
}
