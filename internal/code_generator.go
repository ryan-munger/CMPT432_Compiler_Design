package internal

import (
	"fmt"
	"strconv"
	"strings"
)

var (
	memList       []*[256]byte
	curMem        *[256]byte // Array of 256 bytes, all init to 0x00
	asmList       [][]byte
	curAsm        []byte // so we can update it from indices later
	curBytePtr    int    = 0
	placeholders  []*placeholder
	curScope      *SymbolTable
	genErrors     int             = 0
	endStackPtr   int             = 0
	topHeapPtr    int             = 255                 // its really 254 as subtracts before access
	boolMemAddr   [2]byte         = [2]byte{0xFF, 0x00} // we will use the last memory segment for comparison results
	storedStrings map[string]int  = make(map[string]int)
	usedScopes    map[string]bool = make(map[string]bool) // map just bc high lookups
	firstTime     bool            = true                  // don't move down scope for block 0
)

type placeholder struct {
	locations    []int // where it appears in code
	asmLocations []int
	symbol       *SymbolEntry // so we know which var its for - scope dependent
	realAddr     [2]byte      // actual location after backpatching
}

func newPlaceholder(node *Node) *placeholder {
	var symbol = lookupSymbol(node.Token.trueContent)
	return &placeholder{locations: []int{}, asmLocations: []int{}, symbol: symbol, realAddr: [2]byte{}}
}

// takes in an ID
func addPlaceholderLocation(node *Node, loc int, asmLoc int) {
	var symbol *SymbolEntry = lookupSymbol(node.Token.trueContent)
	for _, p := range placeholders {
		if p.symbol == symbol {
			p.locations = append(p.locations, loc)
			p.asmLocations = append(p.asmLocations, asmLoc)
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

		// new assembly
		var newAsm []byte
		newAsm = append(newAsm, "6502 Assembly:\n\t"...)
		curAsm = newAsm
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
	addAsm("BRK")
	backpatch()

	if genErrors == 0 {
		Pass(fmt.Sprintf("Successfully generated code and assembly for program %d with 0 errors.",
			pNum+1), "CODE GENERATOR")
		Info(fmt.Sprintf("Program %d Assembly:\n%s\n%s", pNum+1, strings.Repeat("-", 75),
			GetAssembly(pNum)), "GOPILER", true)
		Info(fmt.Sprintf("Program %d 6502 Machine Code:\n%s\n%s", pNum+1, strings.Repeat("-", 75),
			GetMachineCode(pNum, true)), "GOPILER", true)
	} else {
		Fail(fmt.Sprintf("Code Generation for program %d failed with %d error(s).",
			pNum+1, errorCount), "CODE GENERATOR")
		errorMap[pNum] = "code generation"
		asmList[pNum] = []byte{}
		Info(fmt.Sprintf("Compilation of program %d aborted due to code generation error(s).",
			pNum+1), "GOPILER", false)
	}
	// reset for next program
	genErrors = 0
}

func generateCode(node *Node) {
	switch node.Type {
	case "<Block>":
		if firstTime {
			firstTime = false
		} else {
			scopeDown()
		}

		for _, child := range node.Children {
			generateCode(child)
		}

		scopeUp()

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

func scopeDown() {
	// once we go 'down' into a scope and back up from it,
	// we can never go back 'down' into it
	for _, downCandidate := range curScope.subTables {
		// see if it has been used
		if _, exists := usedScopes[downCandidate.scopeID]; !exists {
			// if not used it is our new scope
			// we do all this to determine which child to move down into
			curScope = downCandidate
		}
	}
}

func scopeUp() {
	// don't go up if we are the highest we can go
	if curScope.scopeID != "0" {
		curScope = curScope.parentTable
	}
}

func addBytes(newMem []byte) {
	for _, newByte := range newMem {
		curMem[curBytePtr] = newByte
		curBytePtr++
	}
}

func addAsm(newAsm string) {
	curAsm = append(curAsm, []byte(newAsm+" \n\t")...)
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
		addAsm("LDA #$00")

		// store init value to address (temp 00s for now)
		temp.locations = append(temp.locations, curBytePtr+1)
		addBytes([]byte{0x8D, 0x00, 0x00})
		temp.asmLocations = append(temp.asmLocations, len(curAsm)+4)
		addAsm("STA _TEMP")
	}
}

// id, expr
func generateAssign(node *Node) {
	// edge case for incrementing an ID by 1
	if node.Children[1].Type == "<Addition>" && node.Children[1].Children[0].Token.trueContent == "1" &&
		node.Children[1].Children[1].Type == "Token" && node.Children[1].Children[1].Token.tType == Identifier {

		addPlaceholderLocation(node.Children[1].Children[1], curBytePtr+1, len(curAsm)+4)
		addBytes([]byte{0xEE, 0x00, 0x00}) // increment it!
		addAsm("INC _TEMP")
	} else {
		// load up whatever expr it was
		generateExpr(node.Children[1])
		// store it
		addPlaceholderLocation(node.Children[0], curBytePtr+1, len(curAsm)+4)
		addBytes([]byte{0x8D, 0x00, 0x00})
		addAsm("STA _TEMP")
	}
}

func generateExpr(node *Node) {
	switch node.Type {
	case "Token":
		if node.Token.tType == Digit {
			var b byte = strIntToByte(node.Token.trueContent)
			addBytes([]byte{0xA9, b})
			addAsm(fmt.Sprintf("LDA #$%02X", b))
		} else if node.Token.tType == Identifier {
			addPlaceholderLocation(node, curBytePtr+1, len(curAsm)+4)
			addBytes([]byte{0xAD, 0x00, 0x00}) // load accum from mem
			addAsm("LDA _TEMP")
		} else if node.Token.content == "STRING" {
			// string, heap
			// we store the heap addr in a var
			var strHeapLoc byte = addToHeap(node.Token.trueContent)
			addBytes([]byte{0xA9, strHeapLoc})
			addAsm(fmt.Sprintf("LDA $#%02X", strHeapLoc))
		} else if node.Token.content == "KEYW_TRUE" {
			addBytes([]byte{0xA9, 0x01}) // load true to accum
			addAsm("LDA #$01")
		} else if node.Token.content == "KEYW_FALSE" {
			addBytes([]byte{0xA9, 0x00}) // load false to accum
			addAsm("LDA #$00")
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
	addAsm(fmt.Sprintf("LDA #$%02X", byte(digitTotal)))
	if len(idAddParams) != 0 { // if we don't have IDs no adding needed
		for _, id := range idAddParams {
			// add them up!
			addPlaceholderLocation(id, curBytePtr+1, len(curAsm)+4)
			addBytes([]byte{0x6D, 0x00, 0x00})
			addAsm("ADC _TEMP")
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
			addBytes([]byte{0xA0, b}) // load Y with const
			addAsm(fmt.Sprintf("LDY #$%02X", b))
			addBytes([]byte{0xA2, 0x01}) // load X with 1 for Y printing
			addAsm("LDX #$01")

		} else if toPrint.Token.tType == Identifier {
			var sym *SymbolEntry = lookupSymbol(toPrint.Token.trueContent)
			if sym.dataType == "int" || sym.dataType == "boolean" {
				addPlaceholderLocation(node.Children[0], curBytePtr+1, len(curAsm)+4)
				addBytes([]byte{0xAC, 0x00, 0x00}) // load Y from mem
				addAsm("LDY _TEMP")
				addBytes([]byte{0xA2, 0x01}) // load X with 1 for Y printing
				addAsm("LDX #$01")

			} else { // string ID
				addPlaceholderLocation(node.Children[0], curBytePtr+1, len(curAsm)+4)
				addBytes([]byte{0xAC, 0x00, 0x00}) // load Y w heap addr
				addAsm("LDY _TEMP")
				addBytes([]byte{0xA2, 0x02}) // load X with 2 for addr Y printing
				addAsm("LDX #$02")
			}
		} else if toPrint.Token.content == "STRING" {
			var strHeapLoc byte = addToHeap(toPrint.Token.trueContent)
			addBytes([]byte{0xA0, strHeapLoc}) // load Y with heap addr
			addAsm(fmt.Sprintf("LDY #$%02X", strHeapLoc))
			addBytes([]byte{0xA2, 0x02}) // load X with 2 for addr Y printing
			addAsm("LDX #$02")

		} else if toPrint.Token.content == "KEYW_TRUE" {
			addBytes([]byte{0xA0, 0x01}) // load Y with true
			addAsm("LDY #$01")
			addBytes([]byte{0xA2, 0x01}) // load X with 1 for Y printing
			addAsm("LDX #$01")

		} else if toPrint.Token.content == "KEYW_FALSE" {
			addBytes([]byte{0xA0, 0x00}) // load Y with false
			addAsm("LDY #$00")
			addBytes([]byte{0xA2, 0x01}) // load X with 1 for Y printing
			addAsm("LDX #$01")
		}

	case "<Addition>":
		generateAdd(node.Children[0]) // result is in accum
		// we need to store it, no symbol ref to it though
		var headlessPlaceholder *placeholder = &placeholder{[]int{}, []int{}, nil, [2]byte{}}
		placeholders = append(placeholders, headlessPlaceholder)

		headlessPlaceholder.locations = append(headlessPlaceholder.locations, curBytePtr+1)
		addBytes([]byte{0x8D, 0x00, 0x00}) // store add result
		headlessPlaceholder.asmLocations = append(headlessPlaceholder.asmLocations, len(curAsm)+4)
		addAsm("STA _TEMP")

		headlessPlaceholder.locations = append(headlessPlaceholder.locations, curBytePtr+1)
		addBytes([]byte{0xAC, 0x00, 0x00}) // load stored result to Y
		headlessPlaceholder.asmLocations = append(headlessPlaceholder.asmLocations, len(curAsm)+4)
		addAsm("LDY _TEMP")
		addBytes([]byte{0xA2, 0x01}) // load X with 1 for Y printing
		addAsm("LDX #$01")
	}
	addBytes([]byte{0xFF}) // print sys call
	addAsm("SYS")
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

		for _, loc := range p.asmLocations {
			copy(curAsm[loc:], fmt.Sprintf("$%02X%02X", p.realAddr[0], p.realAddr[1]))
		}
	}
	copyAsm := make([]byte, len(curAsm))
	copy(copyAsm, curAsm)
	asmList = append(asmList, copyAsm)
}

func generateIfWhile(node *Node) {
	var whileReturn int = curBytePtr

	var condition *Node = node.Children[0]
	var block *Node = node.Children[1]
	generateComparison(condition)

	// prep jump
	var jumpPlacehold int = curBytePtr + 1
	addBytes([]byte{0xD0, 0x00})
	var asmJumpFill int = len(curAsm) + 5
	addAsm("BNE $_J")
	var beforeBytePos int = curBytePtr

	generateCode(block)

	// whiles need to go back up
	if node.Type == "<WhileStatement>" {
		// we need the Z to be 0 so we always branch back
		generateComparison(&Node{Type: "Token", Token: &Token{content: "KEYW_FALSE"}})

		var jumpDist byte = byte((curBytePtr + 2) - whileReturn) // (count the D0 and val coming)
		var jumpVal byte = 0xFF - jumpDist + 1                   // 2's comp
		addBytes([]byte{0xD0, jumpVal})
		addAsm(fmt.Sprintf("BNE $%02X", jumpVal))
	}

	// calculate original jump to skip block and backfill
	var afterBytePos int = curBytePtr
	curMem[jumpPlacehold] = byte(afterBytePos - beforeBytePos)
	copy(curAsm[asmJumpFill:], fmt.Sprintf("%02X", byte(afterBytePos-beforeBytePos)))
}

func generateComparison(node *Node) {
	if node.Type == "Token" { // T or F keyword
		addBytes([]byte{0xA9, 0x01}) // load accum 1 (true)
		addAsm("LDA #$01")
		addBytes([]byte{0x8D, boolMemAddr[0], boolMemAddr[1]}) // store in reserved bool mem loc
		addAsm("STA $00FF")

		if node.Token.content == "KEYW_TRUE" {
			addBytes([]byte{0xA2, 0x01}) // load X with 1
			addAsm("LDX #$01")
		} else if node.Token.content == "KEYW_FALSE" {
			addBytes([]byte{0xA2, 0x00}) // load X with 0
			addAsm("LDX #$00")
		}
		addBytes([]byte{0xEC, boolMemAddr[0], boolMemAddr[1]}) // compare X and booladdr to set Z
		addAsm("CPX $00FF")
	}
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

func GetAssembly(program int) string {
	if program < 0 || program > len(asmList)-1 {
		return "Invalid program number"
	} else if len(memList) == 0 || hadError(program) {
		return fmt.Sprintf("No assembly generated due to %s error", errorMap[program])
	}

	return string(asmList[0])
}
