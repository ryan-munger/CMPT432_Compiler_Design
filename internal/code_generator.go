package internal

import (
	"fmt"
	"strconv"
	"strings"
)

var (
	memList       []*[256]byte
	curMem        *[256]byte // Array of 256 bytes, all init to 0x00
	asmList       []*[]byte
	curAsm        []byte // so we can update it from indices later
	curBytePtr    int    = 0
	placeholders  []*placeholder
	curScope      *SymbolTable
	genErrors     int             = 0
	genWarns      int             = 0
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
			return
		}
	}
	// placeholder was not found
	// this happens SPECIFICALLY when var is redecl in a scope,
	// but is being assigned before that new decl. Scope table knows, we don't!
	Warn(fmt.Sprintf("!!! The usage of symbol %s in scope %s at (%d:%d) is referencing the redeclaration in this scope even before the redeclaration statement !!!",
		symbol.name, curScope.scopeID, node.Token.location.line, node.Token.location.startPos), "CODE GENERATOR")
	genWarns++
	var newPlaceholder *placeholder = newPlaceholder(node)
	placeholders = append(placeholders, newPlaceholder)
	addPlaceholderLocation(node, loc, asmLoc)
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
		curAsm = []byte{}
		curAsm = append(curAsm, "6502 Assembly:\n\t"...)
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
		Pass(fmt.Sprintf("Successfully generated machine code and assembly for program %d with 0 errors and %d warning(s).",
			pNum+1, genWarns), "CODE GENERATOR")
		Info(fmt.Sprintf("Program %d Assembly:\n%s\n%s", pNum+1, strings.Repeat("-", 75),
			string(curAsm)), "GOPILER", true)
		Info(fmt.Sprintf("Program %d 6502 Machine Code:\n%s\n%s", pNum+1, strings.Repeat("-", 75),
			GetMachineCode(pNum, true)), "GOPILER", true)
	} else {
		Fail(fmt.Sprintf("Code Generation for program %d failed with %d error(s) and %d warning(s).",
			pNum+1, genErrors, genWarns), "CODE GENERATOR")
		errorMap[pNum] = "code generation"
		asmList[pNum] = &[]byte{}
		Info(fmt.Sprintf("Compilation of program %d aborted due to code generation error(s).",
			pNum+1), "GOPILER", false)
	}
	// reset for next program
	genErrors = 0
	genWarns = 0
	endStackPtr = 0
	topHeapPtr = 255
	curBytePtr = 0
	placeholders = []*placeholder{}
	curScope = nil
	storedStrings = make(map[string]int)
	usedScopes = make(map[string]bool)
	firstTime = true
}

func generateCode(node *Node) {
	if genErrors != 0 {
		return
	}
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
	if curBytePtr+len(newMem) >= topHeapPtr {
		curBytePtr -= len(newMem) // just overwrite the end of existing so not out of bounds
		if genErrors == 0 {
			Error("Memory size exceeded (256 Bytes)", "CODE GENERATOR")
			genErrors++
		}
	}
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
	if node.Children[0].Token.content == "I_TYPE" || node.Children[0].Token.content == "B_TYPE" {
		// load 0 to accum for init
		addBytes([]byte{0xA9, 0x00})
		addAsm("LDA #$00")
	} else {
		// init strings to instant break
		// load last string heap addr (always a padded brk statement)
		addBytes([]byte{0xA9, 0xFE})
		addAsm("LDA #FE")
	}
	// store init value to address (temp 00s for now)
	temp.locations = append(temp.locations, curBytePtr+1)
	addBytes([]byte{0x8D, 0x00, 0x00})
	temp.asmLocations = append(temp.asmLocations, len(curAsm)+4)
	addAsm("STA _TEMP")
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
		} else if node.Token.content == "KEYW_TRUE" || node.Token.content == "KEYW_FALSE" {
			generateComparison(node)
		}

	case "<Addition>":
		generateAdd(node)

	case "<Equality>", "<Inequality>":
		generateComparison(node)
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

	case "<Addition>", "<Equality>", "<Inequality>": // results are in accum
		if toPrint.Type == "<Addition>" {
			generateAdd(node.Children[0])
		} else {
			generateComparison(node.Children[0])
		}

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
		endStackPtr++
		if endStackPtr >= topHeapPtr {
			if genErrors == 0 {
				Error("Memory size exceeded (256 Bytes)", "CODE GENERATOR")
				genErrors++
			}
		} else {
			for _, loc := range p.locations {
				// little endian
				curMem[loc] = p.realAddr[1]
				curMem[loc+1] = p.realAddr[0]
			}

			for _, loc := range p.asmLocations {
				copy(curAsm[loc:], fmt.Sprintf("$%02X%02X", p.realAddr[0], p.realAddr[1]))
			}
		}
	}
	copyAsm := make([]byte, len(curAsm))
	copy(copyAsm, curAsm)
	asmList = append(asmList, &copyAsm)
}

func generateIfWhile(node *Node) {
	var whileReturn int = curBytePtr

	var condition *Node = node.Children[0]
	var block *Node = node.Children[1]
	generateComparison(condition)
	addBytes([]byte{0x8D, boolMemAddr[0], boolMemAddr[1]}) // move result of boolexpr to bool addr
	addAsm("STA $00FF")
	addBytes([]byte{0xA2, 0x01}) // load X with 1 (true)
	addAsm("LDX #$01")
	addBytes([]byte{0xEC, boolMemAddr[0], boolMemAddr[1]}) // compare X and booladdr to set Z
	addAsm("CPX $00FF")

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
		zFlagZero()

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

// sets the z flag to 0
func zFlagZero() {
	addBytes([]byte{0xA9, 0x01}) // load accum 1 (true)
	addAsm("LDA #$01")
	addBytes([]byte{0x8D, boolMemAddr[0], boolMemAddr[1]}) // store in reserved bool mem loc
	addAsm("STA $00FF")
	addBytes([]byte{0xA2, 0x00}) // load X with 0
	addAsm("LDX #$00")
	addBytes([]byte{0xEC, boolMemAddr[0], boolMemAddr[1]})
	addAsm("CPX $00FF")
}

func generateComparison(node *Node) {
	if node.Type == "Token" {
		if node.Token.content == "KEYW_TRUE" {
			addBytes([]byte{0xA9, 0x01}) // load 1 to accum
			addAsm("LDA #$01")

		} else if node.Token.content == "KEYW_FALSE" {
			addBytes([]byte{0xA9, 0x00}) // load 0 to accum
			addAsm("LDA #$00")

		} else if node.Token.tType == Digit {
			var b byte = strIntToByte(node.Token.trueContent)
			addBytes([]byte{0xA9, b})
			addAsm(fmt.Sprintf("LDA #$%02X", b))

		} else if node.Token.content == "STRING" {
			var strHeapLoc byte = addToHeap(node.Token.trueContent)
			addBytes([]byte{0xA9, strHeapLoc})
			addAsm(fmt.Sprintf("LDA $#%02X", strHeapLoc))

		} else {
			// user var
			addPlaceholderLocation(node, curBytePtr+1, len(curAsm)+4)
			addBytes([]byte{0xAD, 0x00, 0x00}) // load accum from mem
			addAsm("LDA _TEMP")
		}
		return
	}

	if node.Type == "<Addition>" {
		// result goes in accum
		generateAdd(node)
	} else if node.Type == "<Equality>" || node.Type == "<Inequality>" {
		var compLeft *Node = node.Children[0]
		var compRight *Node = node.Children[1]

		// generate left and store result
		generateComparison(compLeft)
		var leftPlaceholder *placeholder = &placeholder{[]int{}, []int{}, nil, [2]byte{}}
		placeholders = append(placeholders, leftPlaceholder)
		leftPlaceholder.locations = append(leftPlaceholder.locations, curBytePtr+1)
		addBytes([]byte{0x8D, 0x00, 0x00}) // store accum to temp
		leftPlaceholder.asmLocations = append(leftPlaceholder.asmLocations, len(curAsm)+4)
		addAsm("STA _TEMP")

		// generate right and load into X (store in reserved bool spot first)
		generateComparison(compRight)
		addBytes([]byte{0x8D, boolMemAddr[0], boolMemAddr[1]}) // store in reserved bool mem loc
		addAsm("STA $00FF")
		addBytes([]byte{0xAE, boolMemAddr[0], boolMemAddr[1]}) /// move bool mem addr to X
		addAsm("LDX $00FF")

		// compare X to leftAddr to set Z
		leftPlaceholder.locations = append(leftPlaceholder.locations, curBytePtr+1)
		addBytes([]byte{0xEC, 0x00, 0x00})
		leftPlaceholder.asmLocations = append(leftPlaceholder.asmLocations, len(curAsm)+4)
		addAsm("CPX _TEMP")

		var positiveOutcome int = 1
		var negativeOutcome int = 0
		if node.Type == "<Inequality>" { // comparison succeeds - we failed
			positiveOutcome = 0
			negativeOutcome = 1
		}

		// branch if comparison is false to negative outcome
		addBytes([]byte{0xD0, 0x0E}) // branch past the loading of positive outcome
		addAsm("BNE $0E")

		// positive outcome
		zFlagZero() // so we always branch
		// did Z flag first as to not overwrite result
		addBytes([]byte{0xA9, byte(positiveOutcome)})
		addAsm(fmt.Sprintf("LDA #$%02X", uint8(positiveOutcome)))
		addBytes([]byte{0xD0, 0x02}) // skip the negative outcome
		addAsm("BNE $02")

		// negative outcome
		addBytes([]byte{0xA9, byte(negativeOutcome)})
		addAsm(fmt.Sprintf("LDA #$%02X", uint8(negativeOutcome)))
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

	if genErrors == 0 && topHeapPtr <= curBytePtr {
		Error("Memory size exceeded (256 Bytes)", "CODE GENERATOR")
		genErrors++
	}

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

	return string(*asmList[program])
}
