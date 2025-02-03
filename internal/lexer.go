package internal

import (
	"fmt"
	"regexp"
	"unicode"
)

func isNumber(s string) bool {
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

var keywordRe = regexp.MustCompile(`^(boolean|string|print|while|false|true|int|if|[a-z]{1}|\d+)\S*$`)

func nextProgram(programNum *int) {
	*programNum++ // deref to update it
	Log(fmt.Sprintf("Lexing program %d", *programNum), "LEXER", true)
}

func isSymbol(candidate rune) bool {
	_, exists := SymbolMap[candidate]
	if exists {
		return true
	}
	if meaning, exists := SymbolMap[candidate]; exists {
		fmt.Printf("The symbol '%c' means: %s\n", candidate, meaning)
		return true
	}
	return false
}

func evaluateCollection(collection []rune) string {
	match := keywordRe.FindStringSubmatch(string(collection))
	if match != nil {
		return match[1]
	}
	return "NOMATCH"
}

func tokenize(capture string, line int, pos int) Token {
	var tokenType TokenType
	switch capture {
	case "print", "while", "false", "true", "if":
		tokenType = Keyword
	case "string", "int":
		tokenType = Identifier
	default:
		if len(capture) == 1 && isSymbol(rune(capture[0])) {
			tokenType = Symbol
		} else if isNumber(capture) {
			tokenType = Digits
		} else {
			tokenType = Character
		}
	}

	token := Token{
		tType: tokenType,
		location: Location{
			line:     line,
			startPos: pos,
		},
		content: capture,
	}
	return token
}

func Lex(filedata string) {
	// convert string to array of runes
	// regular strings are indexed by bytes and thus can only handle ASCII
	// since there is a non-0 possibility of unicode, this must be done
	var codeRunes []rune = []rune(filedata)

	var programNum int = 0
	var tokenStream [][]Token
	// careful: indexed from 0 but programs from 1
	// this is NOT an array: it is a 'slice' (dynamically allocated)
	nextProgram(&programNum)

	var currentPos int = 0
	var lastPos int = 0
	var line int = 1  // start at 1
	var liveRune rune // char
	var collection []rune
	var greedyCapture string

	// extract tokens
	for currentPos < len(codeRunes) {
		liveRune = codeRunes[currentPos]

		if isSymbol(liveRune) {
			if len(collection) == 0 { // found a symbol to tokenize directly
				Debug(fmt.Sprintf("%s [ %c ] found at (%d:%d)", SymbolMap[liveRune], liveRune, line, currentPos+1), "LEXER")
				tokenStream[programNum-1] = append(tokenStream[programNum-1], tokenize(greedyCapture, line, lastPos))

			}

			if liveRune == '$' {
				nextProgram(&programNum)
			}
			greedyCapture = evaluateCollection(collection)
			tokenStream[programNum-1] = append(tokenStream[programNum-1], tokenize(greedyCapture, line, lastPos))
		} else if unicode.IsSpace(liveRune) {
			greedyCapture = evaluateCollection(collection)
			tokenStream[programNum-1] = append(tokenStream[programNum-1], tokenize(greedyCapture, line, lastPos))
		} else if liveRune == '\n' {
			line++
		} else {
			collection = append(collection, liveRune) // add to back
		}
		currentPos++
	}
}
