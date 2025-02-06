package internal

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var tokenRe = regexp.MustCompile(`^(boolean|string|print|while|false|true|int|if|[a-z]|\d)\S*$`)

func nextProgram(programNum *int, tokenStream *[][]Token, errors *int, warns *int) {
	*programNum++ // deref to update it
	// add another array for the next program's tokens
	*tokenStream = append(*tokenStream, []Token{})
	Info(fmt.Sprintf("Lexing program %d", *programNum), "GOPILER", true)

	// reset
	*errors = 0
	*warns = 0
}

// do we have another program after $? or just some whitespace or comment
func nextProgramExists(codeRunes []rune, pos int) bool {
	var inComment bool = false
	for i := pos + 1; i < len(codeRunes); i++ {
		// spaces at end don't count as another program
		if !unicode.IsSpace(codeRunes[i]) {
			// if there is just a comment left not a new program
			if codeRunes[i] == '/' && i < len(codeRunes)-1 && codeRunes[i+1] == '*' {
				i++
				inComment = true
			} else if codeRunes[i] == '*' && i < len(codeRunes)-1 && codeRunes[i+1] == '/' {
				i++
				inComment = false
			} else if !inComment {
				return true
			}
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

func evaluatetokenBuffer(tokenBuffer []rune) string {
	// fmt.Println(string(tokenBuffer))
	match := tokenRe.FindStringSubmatch(string(tokenBuffer))
	if match != nil {
		return match[1]
	}
	return "NOMATCH"
}

func tokenize(capture string, line int, pos int, quoteFlag bool) Token {
	var tokenType TokenType
	switch capture {
	case "print", "while", "false", "true", "if":
		tokenType = Keyword
		Debug(fmt.Sprintf("%s [ %s ] found at (%d:%d)", strings.ToUpper(capture), capture, line, pos), "LEXER")

	case "string":
		tokenType = Keyword
		Debug(fmt.Sprintf("S_TYPE [ %s ] found at (%d:%d)", capture, line, pos), "LEXER")

	case "int":
		tokenType = Keyword
		Debug(fmt.Sprintf("I_TYPE [ %s ] found at (%d:%d)", capture, line, pos), "LEXER")

	case "==":
		tokenType = Symbol
		Debug(fmt.Sprintf("EQUAL_OP [ %s ] found at (%d:%d)", capture, line, pos), "LEXER")

	case "!=":
		tokenType = Symbol
		Debug(fmt.Sprintf("N-EQUAL_OP [ %s ] found at (%d:%d)", capture, line, pos), "LEXER")

	case " ":
		tokenType = Character
		Debug(fmt.Sprintf("CHAR [ (space) ] found at (%d:%d)", line, pos), "LEXER")

	default:
		if len(capture) == 1 && isSymbol(rune(capture[0])) {
			tokenType = Symbol
			Debug(fmt.Sprintf("%s [ %s ] found at (%d:%d)", SymbolMap[rune(capture[0])], capture, line, pos), "LEXER")
		} else if unicode.IsDigit(rune(capture[0])) {
			tokenType = Digit
			Debug(fmt.Sprintf("DIGIT [ %s ] found at (%d:%d)", capture, line, pos), "LEXER")
		} else {
			if quoteFlag {
				tokenType = Character
				Debug(fmt.Sprintf("CHAR [ %s ] found at (%d:%d)", capture, line, pos), "LEXER")
			} else {
				tokenType = Identifier
				Debug(fmt.Sprintf("ID [ %s ] found at (%d:%d)", capture, line, pos), "LEXER")
			}
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
	// careful: indexed from 0 but programs from 1
	// this is NOT an array: it is a 'slice' (dynamically allocated)
	var tokenStream [][]Token
	var currentPos int = 0
	var lastPos int = 0
	var line int = 1  // start at 1
	var liveRune rune // char
	var tokenBuffer []rune
	var greedyCapture string
	var newToken Token
	var deadPos int = 0 // count chars from past lines
	var warningCount int = 0
	var errorCount int = 0
	var quoteFlag bool = false
	var commentFlag bool = false

	// kick us off
	nextProgram(&programNum, &tokenStream, &errorCount, &warningCount)

	// extract tokens
	for lastPos < len(codeRunes) {
		liveRune = codeRunes[currentPos]
		// fmt.Println(string(liveRune))

		if quoteFlag && liveRune != '"' {
			if !(unicode.IsLower(liveRune) || liveRune == ' ') {
				if liveRune == '\n' {
					Error(fmt.Sprintf("Invalid character [ \\n ] found in quote at (%d:%d); "+
						"Multiline strings are not permitted.", line, lastPos), "LEXER")
					line++
					deadPos = lastPos + 1 // increment it directly later
				} else if liveRune == '$' {
					Error(fmt.Sprintf("Invalid character [ %c ] found in quote at (%d:%d); "+
						"Perhaps your string is unterminated.", liveRune, line, lastPos), "LEXER")
				} else {
					Error(fmt.Sprintf("Invalid character [ %c ] found in quote at (%d:%d)", liveRune, line, lastPos), "LEXER")
				}
				errorCount++

			} else {
				newToken = tokenize(string(liveRune), line, lastPos-deadPos+1, quoteFlag)
				tokenStream[programNum-1] = append(tokenStream[programNum-1], newToken)
			}
			lastPos++

		} else if commentFlag {
			// can't ignore close comment
			if liveRune == '*' && currentPos < len(codeRunes)-1 && codeRunes[currentPos+1] == '/' {
				commentFlag = false
				lastPos += 2 // close comment is 2 characters
				currentPos++

			} else if liveRune == '\n' {
				line++
				lastPos++
				deadPos = lastPos
			} else {
				// fmt.Println("Threw away " + string(liveRune))
				lastPos++ // throw it away
			}

		} else if isSymbol(liveRune) {
			if len(tokenBuffer) == 0 { // found a symbol to tokenize directly
				// check for ==
				if liveRune == '=' && currentPos < len(codeRunes)-1 && codeRunes[currentPos+1] == '=' {
					newToken = tokenize(string(liveRune)+string(codeRunes[currentPos+1]), line, lastPos-deadPos+1, quoteFlag)
					lastPos += 2 // 2 rune symbol
					tokenStream[programNum-1] = append(tokenStream[programNum-1], newToken)

				} else {
					newToken = tokenize(string(liveRune), line, lastPos-deadPos+1, quoteFlag)
					lastPos++
					tokenStream[programNum-1] = append(tokenStream[programNum-1], newToken)

					if liveRune == '$' {
						if errorCount == 0 {
							Pass(fmt.Sprintf("Lexer processed program %d with %d warnings, producing %d tokens.",
								programNum, warningCount, len(tokenStream[programNum-1])), "LEXER")
							Parse(tokenStream[programNum-1], programNum)
						} else {
							Fail(fmt.Sprintf("Lexer failed with %d errors and %d warning(s).", errorCount, warningCount), "LEXER")
						}

						if nextProgramExists(codeRunes, currentPos) {
							nextProgram(&programNum, &tokenStream, &errorCount, &warningCount)
						}
					} else if liveRune == '"' {
						quoteFlag = !quoteFlag // flip it
					}
				}
			} else {
				greedyCapture = evaluatetokenBuffer(tokenBuffer) // hit a symbol, check what we have
				newToken = tokenize(greedyCapture, line, lastPos-deadPos+1, quoteFlag)
				lastPos += len(newToken.content) // find the offset based on chars taken

				if newToken.content == "/*" { // open block comment
					commentFlag = true
				} else {
					tokenStream[programNum-1] = append(tokenStream[programNum-1], newToken)
				}

				tokenBuffer = []rune{} // release old contents
			}
			currentPos = lastPos - 1 // we increment it later

		} else if unicode.IsSpace(liveRune) {
			if len(tokenBuffer) > 0 { // no action if buffer empty
				greedyCapture = evaluatetokenBuffer(tokenBuffer)
				newToken = tokenize(greedyCapture, line, lastPos-deadPos+1, quoteFlag)
				lastPos += len(newToken.content) // find the offset based on chars taken

				if newToken.content == "/*" { // open block comment
					commentFlag = true
				} else {
					tokenStream[programNum-1] = append(tokenStream[programNum-1], newToken)
				}

				tokenBuffer = []rune{}
				currentPos = lastPos - 1
			} else if liveRune == '\n' {
				line++
				lastPos++
				deadPos = lastPos
			} else {
				lastPos++ // move past whitespace
			}

		} else if currentPos >= len(codeRunes)-1 {
			// end of chars - use it like symbol or whitespace
			fmt.Println("This is the end...")
			if len(tokenBuffer) > 0 { // no action if buffer empty
				greedyCapture = evaluatetokenBuffer(tokenBuffer)
				newToken = tokenize(greedyCapture, line, lastPos-deadPos+1, quoteFlag)
				lastPos += len(newToken.content)
				tokenStream[programNum-1] = append(tokenStream[programNum-1], newToken)
			}

		} else {
			// check for !=
			if liveRune == '!' && len(tokenBuffer) == 0 && currentPos < len(codeRunes)-1 && codeRunes[currentPos+1] == '=' {
				newToken = tokenize(string(liveRune)+string(codeRunes[currentPos+1]), line, lastPos-deadPos+1, quoteFlag)
				lastPos += 2 // 2 rune symbol
				currentPos++
				tokenStream[programNum-1] = append(tokenStream[programNum-1], newToken)

				// open comment symbol - we want to keep buffer unaffected
			} else if liveRune == '/' && currentPos < len(codeRunes)-1 && codeRunes[currentPos+1] == '*' {
				commentFlag = true
				lastPos += 2 // open comment is 2 chars
				currentPos++
			} else if unicode.IsLower(liveRune) || unicode.IsDigit(liveRune) {
				tokenBuffer = append(tokenBuffer, liveRune) // add to back
			} else {
				Error(fmt.Sprintf("Invalid token [ %c ] found at (%d:%d)", liveRune, line, lastPos), "LEXER")
				errorCount++
			}
		}

		currentPos++
		if currentPos < lastPos { // ensure we never fall behind - this should not happen
			currentPos = lastPos
		}
	}
	if quoteFlag {
		Warn("EOF reached while inside string; Perhaps string is unterminated.", "LEXER")
		warningCount++
	} else if commentFlag {
		Warn("EOF reached while inside comment; Perhaps comment is unterminated.", "LEXER")
		warningCount++
	}

	// ensure we have a program with tokens and that it terminated with $
	if len(tokenStream) > 0 && len(tokenStream[len(tokenStream)-1]) > 0 &&
		tokenStream[len(tokenStream)-1][len(tokenStream[len(tokenStream)-1])-1].content != "$" {

		Warn("EOF reached before EOP [ $ ]; EOP token was automatically inserted.", "LEXER")
		warningCount++

		// artificially add EOP at end of last line - user will be told where
		newToken = tokenize("$", line, lastPos-deadPos+1, quoteFlag)
		tokenStream[programNum-1] = append(tokenStream[programNum-1], newToken)

		if errorCount == 0 {
			Pass(fmt.Sprintf("Lexer processed program %d with %d warnings, producing %d tokens.",
				programNum, warningCount, len(tokenStream[programNum-1])), "LEXER")
			Parse(tokenStream[programNum-1], programNum)
		} else {
			Fail(fmt.Sprintf("Lexer failed with %d errors and %d warning(s).", errorCount, warningCount), "LEXER")
		}
	}
}
