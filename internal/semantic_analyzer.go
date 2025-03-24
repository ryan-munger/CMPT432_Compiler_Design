package internal

import (
	"fmt"
	"strings"
)

// to store them in case needed later
var astList []TokenTree
var curAst TokenTree
var curParent *Node
var mode string

func startAst(pNum int) {
	// if a program fails in lexer, it never even got to parse
	// we still need to index using program num though
	for len(astList) <= pNum {
		astList = append(astList, TokenTree{})
		curAst = astList[pNum]
		mode = "init"
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

func extractEssentials(node *Node) {
	switch mode {
	case "init": // start the tree
		curAst.rootNode = CopyNode(node) // will always be <program> node
		curParent = curAst.rootNode
		mode = "block"

	case "block":

	default:
		fmt.Println("Erm")
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
