package internal

import (
	"fmt"
	"strings"
)

var (
	memList   []*[256]byte
	curMem    *[256]byte // Array of 256 bytes, all init to 0x00
	asmList   []*strings.Builder
	curAsm    *strings.Builder // constantly adding to it
	genErrors int              = 0
)

func initMem(pNum int) {
	for len(cstList) <= pNum {
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

func CodeGeneration(ast *TokenTree, symbolTable *SymbolTableTree, pNum int) {
	defer func() {
		if r := recover(); r != nil {
			CriticalError("code generator", r)
		}
	}()

	Info(fmt.Sprintf("Generating Code for program %d", pNum+1), "CODE GENERATOR", true)

	Debug("Generating Code from AST...", "CODE GENERATOR")
	initMem(pNum)
	generateCode()

	if genErrors == 0 {
		Pass(fmt.Sprintf("Successfully generated code and assembler for program %d with 0 errors.",
			pNum+1), "CODE GENERATOR")
		Info(fmt.Sprintf("Program %d Assembler:\n%s\n%s", pNum+1, strings.Repeat("-", 75),
			GetAssembler(pNum)), "GOPILER", true)
		Info(fmt.Sprintf("Program %d 6502 Machine Code:\n%s\n%s", pNum+1, strings.Repeat("-", 75),
			GetMachineCode(pNum)), "GOPILER", true)
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

func generateCode() {

}

func GetMachineCode(program int) string {
	if program < 0 || program > len(memList)-1 {
		return "Invalid program number"
	} else if len(memList) == 0 || hadError(program) {
		return fmt.Sprintf("Program %d\n%s\nNo code generated due to %s error\n\n",
			program, strings.Repeat("-", 75), errorMap[0])
	}

	var hexString []string
	for _, b := range memList[program] {
		hexString = append(hexString, fmt.Sprintf("%02x", b))
	}

	return strings.Join(hexString, " ")
}

func GetAssembler(program int) string {
	if program < 0 || program > len(asmList)-1 {
		return "Invalid program number"
	} else if len(memList) == 0 || hadError(program) {
		return fmt.Sprintf("Program %d\n%s\nNo code generated due to %s error\n\n",
			program, strings.Repeat("-", 75), errorMap[0])
	}

	return asmList[program].String()
}
