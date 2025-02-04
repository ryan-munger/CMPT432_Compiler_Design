package internal

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var keywordRe = regexp.MustCompile(`^(boolean|string|print|while|false|true|int|if|[a-z]|\d)\S*$`)

func nextProgram(programNum *int, tokenStream *[][]Token) {
	*programNum++ // deref to update it
	// add another array for the next program's tokens
	*tokenStream = append(*tokenStream, []Token{})
	Info(fmt.Sprintf("Lexing program %d", *programNum), "LEXER", true)
}

// do we have another program after $? or just some whitespace
func nextProgramExists(codeRunes []rune, pos int) bool {
	for i := pos + 1; i < len(codeRunes); i++ {
		if !unicode.IsSpace(codeRunes[i]) {
			return true
		}
	}
	return false
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
	// fmt.Println(string(collection))
	return "NOMATCH"
}

func tokenize(capture string, line int, pos int) Token {
	var tokenType TokenType
	switch capture {
	case "print", "while", "false", "true", "if":
		tokenType = Keyword
		Debug(fmt.Sprintf("%s [ %s ] found at (%d:%d)", strings.ToUpper(capture), capture, line, pos), "LEXER")

	case "string":
		tokenType = Identifier
		Debug(fmt.Sprintf("S_TYPE [ %s ] found at (%d:%d)", capture, line, pos), "LEXER")

	case "int":
		tokenType = Identifier
		Debug(fmt.Sprintf("I_TYPE [ %s ] found at (%d:%d)", capture, line, pos), "LEXER")

	default:
		if len(capture) == 1 && isSymbol(rune(capture[0])) {
			tokenType = Symbol
			Debug(fmt.Sprintf("%s [ %s ] found at (%d:%d)", SymbolMap[rune(capture[0])], capture, line, pos), "LEXER")
		} else if unicode.IsDigit(rune(capture[0])) {
			tokenType = Digit
			Debug(fmt.Sprintf("DIGIT [ %s ] found at (%d:%d)", capture, line, pos), "LEXER")
		} else {
			tokenType = Character
			Debug(fmt.Sprintf("ID [ %s ] found at (%d:%d)", capture, line, pos), "LEXER")
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
	nextProgram(&programNum, &tokenStream)

	var currentPos int = 0
	var lastPos int = 0
	var line int = 1  // start at 1
	var liveRune rune // char
	var collection []rune
	var greedyCapture string
	var newToken Token
	var deadPos int = 0 // count chars from past lines
	var warningCount int = 0

	// extract tokens
	for lastPos < len(codeRunes) {
		liveRune = codeRunes[currentPos]

		if isSymbol(liveRune) {
			if len(collection) == 0 { // found a symbol to tokenize directly
				newToken = tokenize(string(liveRune), line, lastPos-deadPos+1)
			} else {
				greedyCapture = evaluateCollection(collection) // hit a symbol, check what we have
				newToken = tokenize(greedyCapture, line, lastPos-deadPos)
			}

			lastPos += len(newToken.content) // find the offset based on chars taken
			tokenStream[programNum-1] = append(tokenStream[programNum-1], newToken)

			if liveRune == '$' {
				Info(fmt.Sprintf("Lexer processed program %d with %d warnings, producing %d tokens.",
					programNum, warningCount, len(tokenStream[programNum-1])), "LEXER", false)

				if nextProgramExists(codeRunes, currentPos) {
					nextProgram(&programNum, &tokenStream)
				}
			}

			collection = []rune{}    // release old contents
			currentPos = lastPos - 1 // we increment later

		} else if unicode.IsSpace(liveRune) {
			if len(collection) > 0 { // handle repeating spaces
				greedyCapture = evaluateCollection(collection)
				newToken = tokenize(greedyCapture, line, lastPos-deadPos+1)
				lastPos += len(newToken.content) // find the offset based on chars taken
				tokenStream[programNum-1] = append(tokenStream[programNum-1], newToken)

				collection = []rune{}
				currentPos = lastPos - 1
			} else if liveRune == '\n' {
				line++
				lastPos++
				deadPos = lastPos
			} else {
				lastPos++ // move past whitespace
			}

		} else if currentPos >= len(codeRunes) {
			fmt.Printf("hit the back")
			break
		} else {
			collection = append(collection, liveRune) // add to back
		}

		currentPos++
	}
}
