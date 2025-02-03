package internal

import (
	"fmt"

	"github.com/fatih/color"
)

// constrain tokentypes to the 5 values
type TokenType string

const (
	Keyword    TokenType = "keyword"
	Identifier TokenType = "identifier"
	Symbol     TokenType = "symbol"
	Digits     TokenType = "digits"
	Character  TokenType = "character"
)

type Location struct {
	line     int
	startPos int
}

type Token struct {
	tType    TokenType
	location Location
	content  string
}

var Verbose bool

func SetVerbose(toggle bool) {
	Verbose = toggle
}

// capitalized = export
// lowercase = internal
func Log(msg string, component string, space bool) {
	if space {
		fmt.Printf("\n%-5s | %s --> %s\n", "LOG", component, msg)
	} else {
		fmt.Printf("%-5s | %s --> %s\n", "LOG", component, msg)
	}
}

func Error(msg string, component string) {
	if Verbose {
		color.Red(fmt.Sprintf("%-5s | %s --> %s\n", "ERROR", component, msg))
	}
}

func Warn(msg string, component string) {
	color.Yellow(fmt.Sprintf("%-5s | %s --> %s\n", "WARN", component, msg))
}

func Pass(msg string, component string) {
	color.Green(fmt.Sprintf("%-5s | %s --> %s\n", "PASS", component, msg))
}

func Debug(msg string, component string) {
	if Verbose {
		color.Blue(fmt.Sprintf("%-5s | %s --> %s\n", "DEBUG", component, msg))
	}
}

var SymbolMap = map[rune]string{
	'{': "OPEN_BRACE",
	'}': "CLOSE_BRACE",
	'(': "OPEN_PAREN",
	')': "CLOSE_PAREN",
	'$': "EOPS",
	'=': "ASSIGN_OP",
}
