package internal

import "fmt"

/* I know that I could just import a tree to use for this project.
Why not use my DSA skills to make my own?

Go does not have classes. However, you can define methods on types.
A method is a function with a special receiver argument.
The receiver appears in its own argument list between the func keyword and the method name. */

var printTreeBuffer string // used to build str rep of tree

// capitalize its fields for global access
type Node struct {
	Type     string // token (terminals), or name of parse block (expr, IfStatement, etc)
	Token    *Token // null for non-terminals!!
	Children []*Node
}

type TokenTree struct {
	rootNode *Node
}

func NewNode(nodeType string, token *Token) *Node {
	return &Node{Type: nodeType, Token: token, Children: []*Node{}}
}

// copies the node but without the relationships (for building AST from CST)
func CopyNode(original *Node) *Node {
	if original == nil {
		return nil
	}

	newNode := &Node{
		Type:     original.Type,
		Token:    original.Token, // Shallow copy of token (not modifying original token)
		Children: []*Node{},      // Empty children
	}

	return newNode
}

func (node *Node) AddChild(newChild *Node) {
	node.Children = append(node.Children, newChild)
}

func (node *Node) PrintNode(level int) {
	for i := 0; i < level; i++ {
		printTreeBuffer += "-"
	}

	if node.Token != nil {
		if node.Token.trueContent == " " { // we have a token
			printTreeBuffer += fmt.Sprintf("{%s [ space ]}\n", node.Token.content)
		} else {
			printTreeBuffer += fmt.Sprintf("{%s [ %s ]}\n", node.Token.content, node.Token.trueContent)
		}
	} else {
		printTreeBuffer += node.Type + "\n" // non terminal
	}

	for _, child := range node.Children {
		child.PrintNode(level + 1) // Recursively print children
	}
}

func (tree *TokenTree) drawTree() string {
	printTreeBuffer = ""
	tree.rootNode.PrintNode(0)
	return printTreeBuffer
}
