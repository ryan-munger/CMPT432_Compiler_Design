package internal

import (
	"fmt"
	"strings"
)

var memory [256]byte // Array of 256 bytes, all init to 0x00

func CodeGeneration(ast *TokenTree, symbolTable *SymbolTableTree, pNum int) {
	// Info("Code Gen", "CODE GENERATOR", true)
}

func GetMachineCode() string {
	var hexStrings []string

	for _, b := range memory {
		hexStrings = append(hexStrings, fmt.Sprintf("%02x", b))
	}

	return strings.Join(hexStrings, " ")
}
