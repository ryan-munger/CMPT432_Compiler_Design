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
	// +1 for human indexing starting at 1
	Info(fmt.Sprintf("Lexing program %d", *programNum+1), "GOPILER", true)

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
	return exists
}

func handleNewLine(line *int, lastPos *int, deadPos *int) {
	*line++
	*lastPos++
	*deadPos = *lastPos
}

func evaluateTokenBuffer(tokenBuffer []rune) string {
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

	case "boolean":
		tokenType = Keyword
		Debug(fmt.Sprintf("B_TYPE [ %s ] found at (%d:%d)", capture, line, pos), "LEXER")

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
			// char or identifier is based off quotes
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
	defer func() { // so that any errors do not explode the compiler
		if r := recover(); r != nil {
			CriticalError("lexer", r)
		}
	}()

	// convert string to array of runes
	// regular strings are indexed by bytes and thus can only handle ASCII
	// since there is a non-0 possibility of unicode, this must be done
	var codeRunes []rune = []rune(filedata)

	var programNum int = -1 // since we increment it to 0 on start
	// careful: indexed from 0 but programs from 1
	// this is NOT an array: it is a 'slice' (dynamically allocated)
	var tokenStream [][]Token
	var currentPos int = 0
	var lastPos int = 0
	var line int = 1  // start at 1 as its a value for user only
	var liveRune rune // char
	var tokenBuffer []rune
	var greedyCapture string
	var newToken Token
	var deadPos int = 0 // count chars from past lines
	var warningCount int = 0
	var errorCount int = 0
	var quoteFlag bool = false
	var commentFlag bool = false
	var evaluateBuffer = false

	// kick us off - but trust NOBODY - could be file of just whitespace
	if nextProgramExists(codeRunes, 0) {
		nextProgram(&programNum, &tokenStream, &errorCount, &warningCount)
	} else {
		Warn("\nCode provided is only whitespace! No tokens generated.", "LEXER")
		return
	}

	// extract tokens
	for lastPos < len(codeRunes) {
		liveRune = codeRunes[currentPos]
		// fmt.Println(string(liveRune))

		if quoteFlag && liveRune != '"' {
			var currentCol = lastPos - deadPos + 1
			if !(unicode.IsLower(liveRune) || liveRune == ' ') {
				if liveRune == '\n' {
					Error(fmt.Sprintf("Invalid character [ \\n ] found in quote at (%d:%d); "+
						"Multiline strings are not permitted.", line, currentCol), "LEXER")
					lastPos-- // bc newline function and this block both add to it
					handleNewLine(&line, &lastPos, &deadPos)
				} else if liveRune == '$' {
					Error(fmt.Sprintf("Invalid character [ %c ] found in quote at (%d:%d); "+
						"Perhaps your string is unterminated.", liveRune, line, currentCol), "LEXER")
				} else if unicode.IsUpper(liveRune) {
					Error(fmt.Sprintf("Invalid character [ %c ] found in quote at (%d:%d); "+
						"Hint: Capital letters are not permitted.", liveRune, line, currentCol), "LEXER")
				} else if unicode.IsDigit(liveRune) {
					Error(fmt.Sprintf("Invalid character [ %c ] found in quote at (%d:%d); "+
						"Hint: Digits are not permitted.", liveRune, line, currentCol), "LEXER")
				} else {
					Error(fmt.Sprintf("Invalid character [ %c ] found in quote at (%d:%d)", liveRune, line, currentCol), "LEXER")
				}
				errorCount++

			} else { // valid quote chars get their own tokens
				newToken = tokenize(string(liveRune), line, currentCol, quoteFlag)
				tokenStream[programNum] = append(tokenStream[programNum], newToken)
			}
			lastPos++ // we added a char

		} else if commentFlag {
			// can't ignore close comment in comment
			// lookahead for the / after *
			if liveRune == '*' && currentPos < len(codeRunes)-1 && codeRunes[currentPos+1] == '/' {
				commentFlag = false
				lastPos += 2 // close comment is 2 characters
				currentPos++

			} else if liveRune == '\n' {
				// we still need to keep track of line despite comment
				handleNewLine(&line, &lastPos, &deadPos)
			} else {
				// fmt.Println("Threw away " + string(liveRune))
				lastPos++ // throw it away
			}

		} else if isSymbol(liveRune) {
			if len(tokenBuffer) == 0 { // found a symbol to tokenize directly
				// check for == with lookahead
				if liveRune == '=' && currentPos < len(codeRunes)-1 && codeRunes[currentPos+1] == '=' {
					newToken = tokenize(string(liveRune)+string(codeRunes[currentPos+1]), line, lastPos-deadPos+1, quoteFlag)
					lastPos += 2             // 2 rune symbol
					currentPos = lastPos - 1 // incremented at end of loop
					tokenStream[programNum] = append(tokenStream[programNum], newToken)

				} else {
					// tokenize single symbol
					newToken = tokenize(string(liveRune), line, lastPos-deadPos+1, quoteFlag)
					lastPos++
					currentPos = lastPos - 1
					tokenStream[programNum] = append(tokenStream[programNum], newToken)

					// special cases
					if liveRune == '$' { // EOP
						if errorCount == 0 {
							Pass(fmt.Sprintf("Lexer processed program %d with %d warnings(s), producing %d tokens.",
								programNum+1, warningCount, len(tokenStream[programNum])), "LEXER")
							Parse(tokenStream[programNum], programNum)
						} else {
							Fail(fmt.Sprintf("Lexer failed with %d error(s) and %d warning(s).", errorCount, warningCount), "LEXER")
						}

						if nextProgramExists(codeRunes, currentPos) {
							nextProgram(&programNum, &tokenStream, &errorCount, &warningCount)
						}
					} else if liveRune == '"' {
						quoteFlag = !quoteFlag // flip it
					}
				}
			} else { // use symbol as delimiter
				evaluateBuffer = true
			}

		} else if unicode.IsSpace(liveRune) {
			// use space as delimiter
			if len(tokenBuffer) > 0 { // buffer not empty
				evaluateBuffer = true
			} else if liveRune == '\n' {
				handleNewLine(&line, &lastPos, &deadPos)
			} else {
				lastPos++ // move past whitespace
				currentPos = lastPos - 1
			}

			// EOF delimiter
			// } else if currentPos >= len(codeRunes)-1 {
			// 	// fmt.Println("This is the end...")
			// 	// add last char to buffer
			// 	tokenBuffer = append(tokenBuffer, liveRune)

			// 	if len(tokenBuffer) > 0 { // no action if buffer empty
			// 		evaluateBuffer = true
			// 	}

		} else { // didn't find a delimiter
			// check for !=
			if liveRune == '!' && currentPos < len(codeRunes)-1 && codeRunes[currentPos+1] == '=' {
				if len(tokenBuffer) == 0 {
					newToken = tokenize(string(liveRune)+string(codeRunes[currentPos+1]), line, lastPos-deadPos+1, quoteFlag)
					lastPos += 2 // 2 rune symbol
					currentPos = lastPos - 1
					tokenStream[programNum] = append(tokenStream[programNum], newToken)
				} else {
					// we can allow ! to enter the buffer as long as it is followed by an =
					// we just cannot tokenize != until it is its turn or tokens will be out of order
					tokenBuffer = append(tokenBuffer, liveRune)
				}

				// open comment symbol - we want to keep buffer unaffected
			} else if liveRune == '/' && currentPos < len(codeRunes)-1 && codeRunes[currentPos+1] == '*' {
				commentFlag = true
				lastPos += 2 // open comment is 2 chars
				currentPos = lastPos - 1

			} else if unicode.IsLower(liveRune) || unicode.IsDigit(liveRune) {
				tokenBuffer = append(tokenBuffer, liveRune) // add to back

				// we need to use invalid char as delimiter or i9nt would produce int
			} else {
				if len(tokenBuffer) > 0 { // hold off erroring
					evaluateBuffer = true

				} else { // error the invalid
					if unicode.IsUpper(liveRune) {
						Error(fmt.Sprintf("Invalid token [ %c ] found at (%d:%d); "+
							"Hint: Capital letters are not permitted.", liveRune, line, lastPos-deadPos+1), "LEXER")
					} else {
						Error(fmt.Sprintf("Invalid token [ %c ] found at (%d:%d)", liveRune, line, lastPos-deadPos+1), "LEXER")
					}
					lastPos++
					errorCount++
				}
			}

			// EOF delimiter
			if currentPos >= len(codeRunes)-1 {
				// fmt.Println("This is the end...")
				if len(tokenBuffer) > 0 { // no action if buffer empty
					evaluateBuffer = true
				}
			}

		}

		// get what we can from the buffer, tokenize it, jump forward to end of that token, and clean buffer
		if evaluateBuffer {
			greedyCapture = evaluateTokenBuffer(tokenBuffer) // check what we have
			newToken = tokenize(greedyCapture, line, lastPos-deadPos+1, quoteFlag)
			lastPos += len(newToken.content) // find the offset based on chars taken
			currentPos = lastPos - 1         // incremented at end of loop

			if newToken.content == "/*" { // open block comment
				commentFlag = true
			} else {
				tokenStream[programNum] = append(tokenStream[programNum], newToken)
			}

			tokenBuffer = []rune{} // release old contents

			evaluateBuffer = false
		}

		currentPos++
		if currentPos < lastPos { // ensure we never fall behind - this should never happen
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
		tokenStream[programNum] = append(tokenStream[programNum], newToken)

		if errorCount == 0 {
			Pass(fmt.Sprintf("Lexer processed program %d with %d warning(s), producing %d tokens.",
				programNum+1, warningCount, len(tokenStream[programNum])), "LEXER")
			Parse(tokenStream[programNum], programNum)
		} else {
			Fail(fmt.Sprintf("Lexer failed with %d error(s) and %d warning(s).", errorCount, warningCount), "LEXER")
		}
	}
}
