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
	parentStack         []*Node // Stack to track parent nodes
	stringBuffer        []*Node
	symbolTableTreeList []*SymbolTableTree
	curSymbolTableTree  *SymbolTableTree
	curSymbolTable      *SymbolTable
	errorCount          int         = 0
	warnCount           int         = 0
	scopeDepth          int         = 0 // just for naming the scopes
	scopePopulation     map[int]int     // see how many tables at depth for naming
)

func resetAll() {
	curAst = nil
	curParent = nil
	parentStack = []*Node{}
	stringBuffer = []*Node{}
	curSymbolTableTree = nil
	curSymbolTable = nil
	errorCount = 0
	warnCount = 0
	scopeDepth = 0
	scopePopulation = make(map[int]int)
}

func populationExists(candidate int) bool {
	_, exists := scopePopulation[candidate]
	return exists
}

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
		CodeGeneration(curAst, curSymbolTableTree, programNum)
	} else {
		Fail(fmt.Sprintf("Semantic Analysis failed with %d error(s) and %d warning(s).",
			errorCount, warnCount), "SEMANTIC ANALYZER")
		astList[programNum] = TokenTree{}                    // free memory from the AST since it cannot be used
		symbolTableTreeList[programNum] = &SymbolTableTree{} // free this up too
		Info(fmt.Sprintf("Compilation of program %d aborted due to semantic analysis error(s).",
			programNum+1), "GOPILER", false)
	}
	resetAll()
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

// Each function call gets its own curParent from the stack, so deeper recursion doesn’t interfere
// something important that we want an AST node and children for
func importantNodeAbstraction(node *Node) {
	importantNode := NewNode(node.Type, nil)
	curParent.AddChild(importantNode)
	Debug(fmt.Sprintf("Added %s to the AST under parent: %s",
		importantNode.Type, curParent.Type), "SEMANTIC ANALYZER")

	// Push current parent to stack and update curParent
	parentStack = append(parentStack, curParent)
	curParent = importantNode

	// Process children
	for _, child := range node.Children {
		extractEssentials(child)
	}

	// Restore the previous parent from stack
	curParent = parentStack[len(parentStack)-1]
	parentStack = parentStack[:len(parentStack)-1] // Pop the last element
}

// ensure intop (add) becomes the parent left-recursively
// we can have boolexprs and string exprs here - is valid for AST
func transformIntExpr(node *Node) {
	if len(node.Children) == 1 { // just an int
		extractEssentials(node.Children[0])
	} else { // we have an intop! 3 parts - digit intop expr
		var additionNode *Node = NewNode("<Addition>", nil)
		curParent.AddChild(additionNode)
		Debug(fmt.Sprintf("Added <Addition> operation to AST under parent: %s",
			curParent.Type), "SEMANTIC ANALYZER")

		// Push current parent to stack and update curParent
		parentStack = append(parentStack, curParent)
		curParent = additionNode

		// Add the digit as a child
		curParent.AddChild(CopyNode(node.Children[0].Children[0]))
		// Process the expression following the operator
		extractEssentials(node.Children[2])

		// Restore the previous parent from stack
		curParent = parentStack[len(parentStack)-1]
		parentStack = parentStack[:len(parentStack)-1] // Pop from stack
	}
}

// buffer chars and combine them
func transformStringExpr(node *Node) {
	for _, child := range node.Children {
		extractEssentials(child)
	}
	var concatNode *Node = collapseCharList()
	Debug(fmt.Sprintf("Added StringExpr to AST under parent: %s; Collapsed CharList",
		curParent.Type), "SEMANTIC ANALYZER")
	curParent.AddChild(concatNode)
}

// trust the parser!!! we can hardcode!!
func transformBoolExpr(node *Node) {
	if len(node.Children) == 1 { // just a boolVal
		extractEssentials(node.Children[0])
	} else {
		var boolOpNode *Node
		if node.Children[2].Children[0].Token.content == "EQUAL_OP" {
			boolOpNode = NewNode("<Equality>", nil)
		} else { // N-EQUAL_OP
			boolOpNode = NewNode("<Inequality>", nil)
		}

		curParent.AddChild(boolOpNode)
		Debug(fmt.Sprintf("Added %s comparison to AST under parent: %s",
			boolOpNode.Type, curParent.Type), "SEMANTIC ANALYZER")

		// Push current parent to stack and update curParent
		parentStack = append(parentStack, curParent)
		curParent = boolOpNode

		// Process the two expressions
		extractEssentials(node.Children[1]) // Left expr
		extractEssentials(node.Children[3]) // Right expr

		// Restore the previous parent from stack
		curParent = parentStack[len(parentStack)-1]
		parentStack = parentStack[:len(parentStack)-1] // Pop from stack
	}
}

// individual token
func transformToken(node *Node) {
	var tokenNode *Node = CopyNode(node)

	if node.Type == "Token" && (node.Token.tType == Character || node.Token.content == "QUOTE") {
		stringBuffer = append(stringBuffer, node)
	} else if node.Type == "Token" && !isGarbage(node.Token.content) {
		curParent.AddChild(tokenNode)
		Debug(fmt.Sprintf("Added Token [ %s ] to AST under parent: %s",
			tokenNode.Token.content, curParent.Type), "SEMANTIC ANALYZER")
	}
}

// empty buffer of char nodes into one node w a string value
func collapseCharList() *Node {
	var collapsedStr string = ""
	// we don't want the quotes, but use them for position data when empty string
	for _, charNode := range stringBuffer {
		if charNode.Token.content != "QUOTE" {
			collapsedStr += charNode.Token.trueContent
		}
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
	case "<Block>":
		newDownScope()
		for _, child := range node.Children {
			scopeTypeCheck(child)
		}
		goUpScope()

	case "<VarDecl>":
		analyzeVarDecl(node)
	case "<AssignmentStatement>":
		analyzeAssign(node)

	// intexpr, boolexprs can have type and id issues within
	case "<Addition>":
		analyzeAdd(node)
	case "<Equality>", "<Inequality>":
		analyzeCompare(node)

	// print an id
	case "Token":
		if node.Token.tType == Identifier {
			symbol, err := lookup(node.Token.trueContent, node.Token.location)
			if err == nil {
				if !symbol.isInit {
					Warn(fmt.Sprintf("Usage of uninitialized symbol [ %s ] in scope [ %s ] at (%d:%d)",
						symbol.name, curSymbolTable.scopeID, node.Token.location.line, node.Token.location.startPos), "SEMANTIC ANALYZER")
					warnCount++
				}

				symbol.beenUsed = true
				Debug(fmt.Sprintf("Used entry [ %s ] in scope [ %s ] at (%d:%d)",
					symbol.name, curSymbolTable.scopeID, node.Token.location.line, node.Token.location.startPos), "SEMANTIC ANALYZER")
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
	scopePopulation = make(map[int]int)
}

func newDownScope() {
	// root - has nil parent
	if curSymbolTableTree.rootTable == nil {
		curSymbolTableTree.rootTable = NewSymbolTable("0", nil)
		curSymbolTable = curSymbolTableTree.rootTable
		scopePopulation[0] = 0
		Debug(fmt.Sprintf("Encountered <Block>; Created root symbol table scope [ %s ]",
			curSymbolTable.scopeID), "SEMANTIC ANALYZER")
	} else {
		// not using 1a, 1b bc limit of alpha is 26 possible blocks at certain depth
		// name will be: depth.table number (ex. third table in scope 1)
		var newScopeName string
		if populationExists(scopeDepth) {
			scopePopulation[scopeDepth] = scopePopulation[scopeDepth] + 1
			newScopeName = fmt.Sprintf("%d.%d", scopeDepth, scopePopulation[scopeDepth])
		} else {
			scopePopulation[scopeDepth] = 0
			newScopeName = fmt.Sprintf("%d.%d", scopeDepth, 0)
		}

		var newScope *SymbolTable = NewSymbolTable(newScopeName, curSymbolTable)
		curSymbolTable.AddSubTable(newScope)
		curSymbolTable = newScope
		Debug(fmt.Sprintf("Encountered <Block>; Created new symbol table scope [ %s ] under parent table scope [ %s ]",
			curSymbolTable.scopeID, curSymbolTable.parentTable.scopeID), "SEMANTIC ANALYZER")
	}
	scopeDepth++
}

func goUpScope() {
	// if nil, we are at highest scope
	if curSymbolTable.parentTable != nil {
		curSymbolTable = curSymbolTable.parentTable
		scopeDepth--
	}
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
			warnCount++
		} else if !entry.beenUsed {
			Warn(fmt.Sprintf("ID [ %s ] from scope [ %s ] was declared and initialized but never used.",
				entry.name, table.scopeID), "SEMANTIC ANALYZER")
			warnCount++
		}
	}

	for _, subTable := range table.subTables {
		issueUsageWarnings(subTable)
	}
}

func typeMismatch(operation string, pos Location, leftType string, rightType string) {
	Error(fmt.Sprintf("Type mismatch on (%d:%d): cannot %s type [ %s ] to type [ %s ]",
		pos.line, pos.startPos, operation, rightType, leftType), "SEMANTIC ANALYZER")
	errorCount++
}

// digit, add -> int
// string -> string
// equality, boolval -> bool
// id -> type of symbol
func getNodeType(node *Node, examineChildren bool, markUsed bool) string {
	if node.Type == "<Addition>" {
		if examineChildren {
			analyzeAdd(node)
		}
		return "int"

	} else if node.Type == "<Equality>" || node.Type == "<Inequality>" {
		if examineChildren {
			analyzeCompare(node)
		}
		return "boolean"

	} else if node.Type == "Token" {
		if node.Token.tType == Digit {
			return "int"

		} else if node.Token.content == "STRING" {
			return "string"

		} else if node.Token.tType == Identifier {
			symbol, err := lookup(node.Token.trueContent, node.Token.location)
			if err != nil {
				return "" // id doesn't exist - bail
			}

			if markUsed {
				if !symbol.isInit {
					Warn(fmt.Sprintf("Usage of uninitialized symbol [ %s ] in scope [ %s ] at (%d:%d)",
						symbol.name, curSymbolTable.scopeID, node.Token.location.line, node.Token.location.startPos), "SEMANTIC ANALYZER")
					warnCount++
				}

				symbol.beenUsed = true
				Debug(fmt.Sprintf("Used entry [ %s ] in scope [ %s ] at (%d:%d)",
					symbol.name, curSymbolTable.scopeID, node.Token.location.line, node.Token.location.startPos), "SEMANTIC ANALYZER")
			}
			return symbol.dataType

		} else { // boolval
			return "boolean"

		}
	} else {
		return ""
	}
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
		Debug(fmt.Sprintf("Declared new entry [ %s ] of type [ %s ] in scope [ %s ] at (%d:%d)",
			name, dType, curSymbolTable.scopeID, pos.line, pos.startPos), "SEMANTIC ANALYZER")
	}
}

func analyzeAssign(node *Node) {
	var assigneeNode *Node = node.Children[0]
	assignee, err := lookup(assigneeNode.Token.trueContent, node.Children[0].Token.location)
	// assignee does not exist, we are done here
	if err != nil {
		return
	}
	var assignTo *Node = node.Children[1]
	var assignToType string = getNodeType(assignTo, true, true)

	if assignToType == "" {
		return // bad ID - go no further
	} else if assignee.dataType != assignToType {
		typeMismatch("assign", assigneeNode.Token.location, assignee.dataType, assignToType)
	} else {
		Debug(fmt.Sprintf("Type checked assignment of entry [ %s ] in scope [ %s ] at (%d:%d)",
			assignee.name, curSymbolTable.scopeID, assignee.position.line, assignee.position.startPos), "SEMANTIC ANALYZER")
		if !assignee.isInit {
			assignee.isInit = true
			Debug(fmt.Sprintf("Initialized entry [ %s ] in scope [ %s ] at (%d:%d)",
				assignee.name, curSymbolTable.scopeID, assignee.position.line, assignee.position.startPos), "SEMANTIC ANALYZER")
		}
	}
}

// easier bc no valid subexpressions that aren't add
// we don't even need to check the left side of intop because parser did (digit)
// right can be add, string, digit, id, bool, equality
func analyzeAdd(node *Node) {
	var leftAdd *Node = node.Children[0] // always a digit!!
	var rightAdd *Node = node.Children[1]
	var rightAddType string = getNodeType(rightAdd, true, true)

	if rightAddType == "" {
		return // bad ID - go no further
	} else if rightAddType != "int" {
		typeMismatch("compare", leftAdd.Token.location, "int", rightAddType)
	} else {
		Debug(fmt.Sprintf("Type checked <Addition> at (%d:%d)",
			leftAdd.Token.location.line, leftAdd.Token.location.startPos), "SEMANTIC ANALYZER")
	}
}

// -boolop
// --expr
// --expr
func analyzeCompare(node *Node) {
	// the types of these must match
	var leftCompare *Node = node.Children[0]
	var leftType string = getNodeType(leftCompare, true, true)
	var rightCompare *Node = node.Children[1]
	var rightType string = getNodeType(rightCompare, true, true)

	if leftType == "" || rightType == "" {
		return // bad ID - go no further
	} else if leftType != rightType {
		typeMismatch("compare", leftCompare.Token.location, leftType, rightType)
	} else {
		Debug(fmt.Sprintf("Type checked %s", node.Type), "SEMANTIC ANALYZER")
	}
}
