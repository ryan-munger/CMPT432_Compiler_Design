package internal

import (
	"fmt"
	"unicode"
)

func nextProgram(programNum *int) {
	*programNum++ // deref to update it
	Log(fmt.Sprintf("Compiling program %d", *programNum))
}

func isSymbol(candidate rune) bool {
	if meaning, exists := SymbolMap[candidate]; exists {
		fmt.Printf("The symbol '%c' means: %s\n", candidate, meaning)
		return true
	}
	return false
}

func Lex(filedata string) {
	// convert string to array of runes
	// regular strings are indexed by bytes and thus can only handle ASCII
	// since there is a non-0 possibility of unicode, this must be done
	var codeRunes []rune = []rune(filedata)

	var programNum int = 0
	// careful: indexed from 0 but programs from 1
	// this is NOT an array: it is a 'slice' (dynamically allocated)
	//var tokenStream [][]Token
	nextProgram(&programNum)

	var currentPos int = 0
	//var lastPos int = 0
	var line int = 0
	var liveRune rune // char
	// token extraction loop - what in the while syntax...
	for currentPos < len(codeRunes) { // stay within the user's code
		liveRune = codeRunes[currentPos]

		if isSymbol(liveRune) {
			// do stuff
		} else if liveRune == '\n' {
			line++
			fmt.Println("newline")
		} else if unicode.IsSpace(liveRune) {
			fmt.Println("white")
		} else {
			fmt.Println("char")
		}
		currentPos++
	}
}
