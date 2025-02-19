package internal

import "fmt"

/* I know that I could just import a tree to use for this project.
Why not use my DSA skills to make my own?

Go does not have classes. However, you can define methods on types.
A method is a function with a special receiver argument.
The receiver appears in its own argument list between the func keyword and the method name. */

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

func (node *Node) AddChild(newChild *Node) {
	node.Children = append(node.Children, newChild)
}

func (node *Node) PrintNode(level int) {
	for i := 0; i < level; i++ {
		fmt.Print("-")
	}

	if node.Token != nil {
		if node.Token.trueContent == " " {
			fmt.Printf("{%s [ space ]}\n", node.Token.content) // token
		} else {
			fmt.Printf("{%s [ %s ]}\n", node.Token.content, node.Token.trueContent) // token
		}
	} else {
		fmt.Println(node.Type) // non terminal
	}

	for _, child := range node.Children {
		child.PrintNode(level + 1) // Recursively print children
	}
}

func (tree *TokenTree) PrintTree() {
	tree.rootNode.PrintNode(0)
}

func (node *Node) RemoveChild(target *Node) bool {
	for i, child := range node.Children {
		// Compare either token or type (for non-terminals)
		if (child.Token != nil && target.Token != nil && TokensAreEqual(child.Token, target.Token)) ||
			(child.Type == target.Type && target.Token == nil) {
			node.Children = append(node.Children[:i], node.Children[i+1:]...)
			return true
		}
	}
	return false
}

func RemoveNode(root *Node, target *Node) bool {
	if root == nil {
		return false
	}

	if (root.Token != nil && target.Token != nil && TokensAreEqual(root.Token, target.Token)) ||
		(root.Type == target.Type && target.Token == nil) {
		fmt.Println("Cannot remove root node directly")
		return false
	}

	for i, child := range root.Children {
		if (child.Token != nil && target.Token != nil && TokensAreEqual(child.Token, target.Token)) ||
			(child.Type == target.Type && target.Token == nil) {
			// remove child from slice
			root.Children = append(root.Children[:i], root.Children[i+1:]...)
			return true
		} else {
			// Recur into children
			if RemoveNode(child, target) {
				return true
			}
		}
	}

	return false
}

// Wrapper method for tree
func (tree *TokenTree) RemoveNode(target *Node) bool {
	if tree.rootNode == nil {
		return false
	}
	return RemoveNode(tree.rootNode, target)
}

func (node *Node) Search(target *Node) bool {
	if node == nil {
		return false
	}

	if (node.Token != nil && target.Token != nil && TokensAreEqual(node.Token, target.Token)) ||
		(node.Type == target.Type && target.Token == nil) {
		return true
	}

	// search children
	for _, child := range node.Children {
		if child.Search(target) {
			return true
		}
	}

	return false
}

// wrapper for tree
func (tree *TokenTree) SearchTree(target *Node) bool {
	if tree.rootNode == nil {
		return false
	}
	return tree.rootNode.Search(target)
}
