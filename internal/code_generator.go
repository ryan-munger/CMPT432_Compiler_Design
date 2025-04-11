package internal

import (
	"fmt"
	"strings"
)

var memList []*[256]byte
var curMem *[256]byte // Array of 256 bytes, all init to 0x00
var asmList []*strings.Builder
var curAsm *strings.Builder // constantly adding to it

func initMem() {
	// new memory
	var newMem [256]byte
	memList = append(memList, &newMem)
	curMem = &newMem

	// new assembler
	var newAsm strings.Builder
	asmList = append(asmList, &newAsm)
	curAsm = &newAsm
}

func CodeGeneration(ast *TokenTree, symbolTable *SymbolTableTree, pNum int) {
	defer func() {
		if r := recover(); r != nil {
			CriticalError("code generator", r)
		}
	}()

	Info(fmt.Sprintf("Generating Code for program %d", pNum+1), "CODE GENERATOR", true)

	Debug("Generating Code from AST...", "CODE GENERATOR")
	initMem()
	generateCode()
}

func generateCode() {

}

func GetMachineCode(program int) string {
	if program < 0 || program > len(memList)-1 {
		return "Invalid program number"
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
	}

	return asmList[program].String()
}
