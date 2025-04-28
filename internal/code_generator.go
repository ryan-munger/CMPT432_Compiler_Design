package internal

import (
	"fmt"
	"strconv"
	"strings"
)

var (
	memList       []*[256]byte
	curMem        *[256]byte // Array of 256 bytes, all init to 0x00
	asmList       []*strings.Builder
	curAsm        *strings.Builder
	curBytePtr    int = 0
	placeholders  []*placeholder
	curScope      *SymbolTable
	genErrors     int            = 0
	endStackPtr   int            = 0
	topHeapPtr    int            = 256
	storedStrings map[string]int = make(map[string]int)
)

type placeholder struct {
	locations []int        // where it appears in code
	symbol    *SymbolEntry // so we know which var its for - scope dependent
	realAddr  [2]byte      // actual location after backpatching
}

func newPlaceholder(node *Node) *placeholder {
	var symbol = lookupSymbol(node.Token.trueContent)
	return &placeholder{locations: []int{}, symbol: symbol, realAddr: [2]byte{}}
}

// takes in an ID
func addPlaceholderLocation(node *Node, loc int) {
	var symbol *SymbolEntry = lookupSymbol(node.Token.trueContent)
	for _, p := range placeholders {
		if p.symbol == symbol {
			p.locations = append(p.locations, loc)
		}
	}
}

// will always exist (thanks semantic analysis)
func lookupSymbol(name string) *SymbolEntry {
	var searchTable *SymbolTable = curScope
	for {
		if searchTable.EntryExists(name) {
			return searchTable.entries[name]

		} else if searchTable.parentTable != nil {
			searchTable = searchTable.parentTable // look above
		}
	}
}

func initMem(pNum int) {
	for len(memList) <= pNum {
		// new memory
		var newMem [256]byte
		memList = append(memList, &newMem)
		curMem = &newMem

		// new assembler
		var newAsm strings.Builder
		newAsm.WriteString("6502 Assembler:\n")
		asmList = append(asmList, &newAsm)
		curAsm = &newAsm
	}
}

func strIntToByte(strInt string) byte {
	num, _ := strconv.Atoi(strInt)
	return byte(num)
}

func CodeGeneration(ast *TokenTree, symbolTableTree *SymbolTableTree, pNum int) {
	defer func() {
		if r := recover(); r != nil {
			CriticalError("code generator", r)
		}
	}()

	Info(fmt.Sprintf("Generating Code for program %d", pNum+1), "CODE GENERATOR", true)

	Debug("Generating Code from AST...", "CODE GENERATOR")
	initMem(pNum)
	curScope = symbolTableTree.rootTable
	generateCode(ast.rootNode)
	addBytes([]byte{0x00}) // break
	backpatch()

	if genErrors == 0 {
		Pass(fmt.Sprintf("Successfully generated code and assembler for program %d with 0 errors.",
			pNum+1), "CODE GENERATOR")
		Info(fmt.Sprintf("Program %d Assembler:\n%s\n%s", pNum+1, strings.Repeat("-", 75),
			GetAssembler(pNum)), "GOPILER", true)
		Info(fmt.Sprintf("Program %d 6502 Machine Code:\n%s\n%s", pNum+1, strings.Repeat("-", 75),
			GetMachineCode(pNum, true)), "GOPILER", true)
	} else {
		Fail(fmt.Sprintf("Code Generation for program %d failed with %d error(s).",
			pNum+1, errorCount), "CODE GENERATOR")
		errorMap[pNum] = "code generation"
		asmList[pNum] = &strings.Builder{}
		Info(fmt.Sprintf("Compilation of program %d aborted due to code generation error(s).",
			pNum+1), "GOPILER", false)
	}
	// reset for next program
	genErrors = 0
}

func generateCode(node *Node) {
	switch node.Type {
	case "<Block>":
		// go down a scope
		for _, child := range node.Children {
			generateCode(child)
		}
		// go up a scope

	case "<VarDecl>":
		generateVarDecl(node)
	case "<AssignmentStatement>":
		generateAssign(node)
	case "<PrintStatement>":
		generatePrint(node)
	case "<IfStatement>", "<WhileStatement>":
		generateIfWhile(node)
	default:
		// the children of the node are important even if the node itself is not
		for _, child := range node.Children {
			generateCode(child)
		}
	}
}

func addBytes(newMem []byte) {
	for _, newByte := range newMem {
		curMem[curBytePtr] = newByte
		curBytePtr++
	}
}

// type, id
func generateVarDecl(node *Node) {
	// placeholder for var
	var temp *placeholder = newPlaceholder(node.Children[1])
	placeholders = append(placeholders, temp)

	// we initialize bools and ints to 0
	if node.Children[0].Type == "I_TYPE" || node.Children[0].Type == "B_TYPE" {
		// load 0 to accum for init
		addBytes([]byte{0xA9, 0x00})

		// store init value to address (temp 00s for now)
		temp.locations = append(temp.locations, curBytePtr+1)
		addBytes([]byte{0x8D, 0x00, 0x00})
	}
}

// id, expr
func generateAssign(node *Node) {
	// edge case for incrementing an ID by 1
	if node.Children[1].Type == "<Addition>" && node.Children[1].Children[0].Token.trueContent == "1" &&
		node.Children[1].Children[1].Type == "Token" && node.Children[1].Children[1].Token.tType == Identifier {

		addPlaceholderLocation(node.Children[1].Children[1], curBytePtr+1)
		addBytes([]byte{0xEE, 0x00, 0x00}) // increment it!
	} else {
		// load up whatever expr it was
		generateExpr(node.Children[1])
		// store it
		addPlaceholderLocation(node.Children[0], curBytePtr+1)
		addBytes([]byte{0x8D, 0x00, 0x00})
	}
}

func generateExpr(node *Node) {
	switch node.Type {
	case "Token":
		if node.Token.tType == Digit {
			var b byte = strIntToByte(node.Token.trueContent)
			addBytes([]byte{0xA9, b})
		} else if node.Token.tType == Identifier {
			addPlaceholderLocation(node, curBytePtr+1)
			addBytes([]byte{0xAD, 0x00, 0x00}) // load accum from mem
		} else { // string, heap
			// we store the heap addr in a var
			var strHeapLoc byte = addToHeap(node.Token.trueContent)
			addBytes([]byte{0xA9, strHeapLoc})
		}

	case "<Addition>":
		generateAdd(node)

	case "<Inequality>":
		generateComparison(node)

	case "<Equality>":
	}
}

// digit, digit/add
func generateAdd(node *Node) {
	var digAddParams []*Token
	var idAddParams []*Node
	var curAddParent *Node = node
	var param1 *Node
	var param2 *Node

	// collect all things to add
	for { // loop until no more nested add
		param1 = curAddParent.Children[0]
		param2 = curAddParent.Children[1]
		// param1 is always a digit
		digAddParams = append(digAddParams, param1.Token)

		if param2.Type == "Token" { // we are done moving down
			if param2.Token.tType == Digit {
				digAddParams = append(digAddParams, param2.Token)
			} else { // id
				idAddParams = append(idAddParams, param2)
			}
			break
		}
		// nested add
		curAddParent = param2
	}

	// collapse all static digits down
	// I won't constrain it to be below 127 (largest due to 2's comp)
	// If I separated it, adding 120 and 120 instead of storing 240 doesn't help anything
	var digitTotal int
	for _, token := range digAddParams {
		digVal, _ := strconv.Atoi(token.trueContent)
		digitTotal += digVal
	}

	// load collapsed digits to accum for adding
	addBytes([]byte{0xA9, byte(digitTotal)})
	if len(idAddParams) != 0 { // if we don't have IDs no adding needed
		for _, id := range idAddParams {
			// add them up!
			addPlaceholderLocation(id, curBytePtr+1)
			addBytes([]byte{0x6D, 0x00, 0x00})
		}
	}
	// result is in accum when done
}

func generatePrint(node *Node) {
	var toPrint = node.Children[0]
	switch toPrint.Type {
	case "Token":
		if toPrint.Token.tType == Digit {
			var b byte = strIntToByte(toPrint.Token.trueContent)
			addBytes([]byte{0xA0, b})    // load Y with const
			addBytes([]byte{0xA2, 0x01}) // load X with 1 for Y printing
		} else if toPrint.Token.tType == Identifier {
			var sym *SymbolEntry = lookupSymbol(toPrint.Token.trueContent)
			if sym.dataType == "int" || sym.dataType == "boolean" {
				addPlaceholderLocation(node.Children[0], curBytePtr+1)
				addBytes([]byte{0xAC, 0x00, 0x00}) // load Y from mem
				addBytes([]byte{0xA2, 0x01})       // load X with 1 for Y printing
			} else { // string ID
				addPlaceholderLocation(node.Children[0], curBytePtr+1)
				addBytes([]byte{0xAC, 0x00, 0x00}) // load Y w heap addr
				addBytes([]byte{0xA2, 0x02})       // load X with 2 for addr Y printing
			}
		} else if toPrint.Token.content == "STRING" {
			var strHeapLoc byte = addToHeap(toPrint.Token.trueContent)
			addBytes([]byte{0xA0, strHeapLoc}) // load Y with heap addr
			addBytes([]byte{0xA2, 0x02})       // load X with 2 for addr Y printing
		}

	case "<Addition>":
		generateAdd(node.Children[0]) // result is in accum
		// we need to store it, no symbol ref to it though
		var headlessPlaceholder *placeholder = &placeholder{[]int{}, nil, [2]byte{}}
		placeholders = append(placeholders, headlessPlaceholder)

		headlessPlaceholder.locations = append(headlessPlaceholder.locations, curBytePtr+1)
		addBytes([]byte{0x8D, 0x00, 0x00}) // store add result

		headlessPlaceholder.locations = append(headlessPlaceholder.locations, curBytePtr+1)
		addBytes([]byte{0xAC, 0x00, 0x00}) // load stored result to Y
		addBytes([]byte{0xA2, 0x01})       // load X with 1 for Y printing
	}
	addBytes([]byte{0xFF}) // print sys call
}

func backpatch() {
	endStackPtr = curBytePtr
	for _, p := range placeholders {
		p.realAddr = [2]byte{0x00, byte(endStackPtr)}
		endStackPtr += 2

		for _, loc := range p.locations {
			// little endian
			curMem[loc] = p.realAddr[1]
			curMem[loc+1] = p.realAddr[0]
		}
	}
}

func generateIfWhile(node *Node) {
	var whileReturn int = curBytePtr

	var condition *Node = node.Children[0]
	var block *Node = node.Children[1]
	generateComparison(condition)

	// prep jump
	var curBytePos int = curBytePtr
	var jumpPlacehold int = curBytePtr + 1
	addBytes([]byte{0xD0, 0x00})

	generateCode(block)
	var afterBytePos int = curBytePtr

	// calculate jump and backfill
	curMem[jumpPlacehold] = byte(afterBytePos - curBytePos)

	if node.Type == "<WhileStatement>" {
		var jumpDist byte = byte(afterBytePos - whileReturn)
		var jumpVal byte = 0xFF - jumpDist // 2's comp
		addBytes([]byte{0xD0, jumpVal})
	}
}

func generateComparison(node *Node) {

}

func addToHeap(str string) byte {
	loc, exists := storedStrings[str]
	// if we already have it, just say where
	if exists {
		return byte(loc)
	}

	topHeapPtr--
	curMem[topHeapPtr] = 0x00 // 0x00 terminated str
	topHeapPtr -= len(str)    // fills bottom up
	for i, char := range str {
		curMem[topHeapPtr+i] = byte(char)
	}
	storedStrings[str] = topHeapPtr // remember we have it stored
	return byte(topHeapPtr)
}

func GetMachineCode(program int, eightBreaks bool) string {
	if program < 0 || program > len(memList)-1 {
		return "Invalid program number"
	} else if len(memList) == 0 || hadError(program) {
		return fmt.Sprintf("No machine code generated due to %s error", errorMap[program])
	}

	var hexString []string
	if eightBreaks {
		hexString = append(hexString, " ")
	}

	for i, b := range memList[program] {
		if eightBreaks && i%8 == 0 {
			hexString = append(hexString, "\n")
		}
		hexString = append(hexString, fmt.Sprintf("%02X", b))
	}

	return strings.Join(hexString, " ")
}

func GetAssembler(program int) string {
	if program < 0 || program > len(asmList)-1 {
		return "Invalid program number"
	} else if len(memList) == 0 || hadError(program) {
		return fmt.Sprintf("No assembler generated due to %s error", errorMap[program])
	}

	return asmList[program].String()
}
