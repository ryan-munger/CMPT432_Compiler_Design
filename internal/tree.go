package internal

import "fmt"

/* I know that I could just import a tree to use for this project.
Why not use my DSA skills to make my own?

Go does not have classes. However, you can define methods on types.
A method is a function with a special receiver argument.
The receiver appears in its own argument list between the func keyword and the method name. */

// capitalize its fields for global access
type Node struct {
	Value    Token
	Children []*Node // slice (not array) of node ptrs
}

type TokenTree struct {
	rootNode *Node
}

func NewNode(token Token) *Node {
	return &Node{Value: token, Children: []*Node{}}
}

// method can only be performed on node!
func (node *Node) AddChild(newChild *Node) {
	node.Children = append(node.Children, newChild)
}

func (node *Node) PrintNode(level int) {
	for i := 0; i < level; i++ {
		fmt.Print("  ") // hierarchy
	}
	fmt.Println(node.Value)

	for _, child := range node.Children {
		child.PrintNode(level + 1) // recursively print children
	}
}

func (tree *TokenTree) PrintTree() {
	tree.rootNode.PrintNode(0)
}

func (node *Node) RemoveChild(value Token) bool {
	for i, child := range node.Children {
		if TokensAreEqual(&child.Value, &value) {
			// slice manipulation to remove
			node.Children = append(node.Children[:i], node.Children[i+1:]...)
			return true
		}
	}
	return false
}

func RemoveNode(root *Node, value Token) bool {
	if TokensAreEqual(&root.Value, &value) {
		fmt.Println("Cannot remove root")
		return false
	}

	for _, child := range root.Children {
		if TokensAreEqual(&child.Value, &value) {
			root.RemoveChild(value)
			return true
		} else {
			if RemoveNode(child, value) { // search children
				return true
			}
		}
	}
	return false
}

func (tree TokenTree) RemoveNode(value Token) bool {
	return RemoveNode(tree.rootNode, value)
}

func (node *Node) Search(targetToken *Token) bool {
	if node == nil {
		return false // base case
	}

	if TokensAreEqual(&node.Value, targetToken) {
		return true
	}

	// search children
	for _, child := range node.Children {
		found := child.Search(targetToken) // Recursive call
		return found
	}

	return false
}

func (tree TokenTree) SearchTree(target *Token) bool {
	return tree.rootNode.Search(target)
}

/* Tree test cases!
func TreeTest() {
	// Test Case 1: make a tree
	var tokenTree TokenTree

	token1 := Token{tType: Identifier, content: "x"}
	tokenTree.rootNode = NewNode(token1)
	token2 := Token{tType: Identifier, content: "test"}
	child1 := NewNode(token2)
	tokenTree.rootNode.AddChild(child1)
	token3 := Token{tType: Symbol, content: "+"}
	child2 := NewNode(token3)
	tokenTree.rootNode.AddChild(child2)

	fmt.Println("Test Case 1: Make a Tree")
	tokenTree.PrintTree()

	// Test Case 2:find a node
	foundNode := tokenTree.SearchTree(&token1)
	if foundNode {
		fmt.Println("\nTest Case 2: Found token!")
	} else {
		fmt.Println("\nTest Case 2: Token not found")
	}

	targetToken2 := Token{tType: Symbol, content: "="}
	foundNode2 := tokenTree.SearchTree(&targetToken2)
	if foundNode2 {
		fmt.Println("\nTest Case 2b: Found token!")
	} else {
		fmt.Println("\nTest Case 2b: Token not found")
	}

	// Test Case 3: remove node
	fmt.Println("\nTest Case 3: Remove a node")
	if tokenTree.RemoveNode(token2) {
		fmt.Println("Removed node successfully.")
	} else {
		fmt.Println("Node not found. It was there though...")
	}

	tokenTree.PrintTree()

	// Test Case 4: remove node and its children
	token4 := Token{tType: Character, content: "a"}
	child3 := NewNode(token4)
	token5 := Token{tType: Character, content: "b"}
	grandchild1 := NewNode(token5)
	child3.AddChild(grandchild1)
	tokenTree.rootNode.AddChild(child3)

	fmt.Println("\nTest Case 4: Remove a node and children")
	if tokenTree.RemoveNode(token4) {
		fmt.Println("Removed node successfully!")
	} else {
		fmt.Println("Node not found. Uh oh")
	}
	tokenTree.PrintTree()

}
*/
