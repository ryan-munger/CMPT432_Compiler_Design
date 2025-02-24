package internal

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var tokenRe = regexp.MustCompile(`^(boolean|string|print|while|false|true|int|if|[a-z]|\d)\S*$`)

func nextProgram(programNum *int, tokenStream *[][]Token, errors *int, warns *int, alreadyFailed *bool) {
	*programNum++ // deref to update it
	// add another array for the next program's tokens
	*tokenStream = append(*tokenStream, []Token{})
	// +1 for human indexing starting at 1
	Info(fmt.Sprintf("Lexing program %d", *programNum+1), "GOPILER", true)

	// reset
	*errors = 0
	*warns = 0
	*alreadyFailed = false
}

// do we have another program after $? or just some whitespace or comment
func nextProgramExists(codeRunes []rune, pos int, untermEndComment *bool) bool {
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
	*untermEndComment = inComment
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
	match := tokenRe.FindStringSubmatch(string(tokenBuffer))
	if match != nil {
		return match[1]
	}
	return "NOMATCH"
}

func passFailProgram(programNum int, errorCount int, warningCount int, tokenStream [][]Token, alreadyFailed *bool) {
	if errorCount == 0 {
		Pass(fmt.Sprintf("Lexer processed program %d with %d warnings(s), producing %d tokens.",
			programNum+1, warningCount, len(tokenStream[programNum])), "LEXER")
		Parse(tokenStream[programNum], programNum)
	} else {
		Fail(fmt.Sprintf("Lexer failed with %d error(s) and %d warning(s).", errorCount, warningCount), "LEXER")
		Info(fmt.Sprintf("Compilation of program %d aborted due to lexer error.", programNum+1), "GOPILER", false)
		tokenStream[programNum] = []Token{} // release memory as tokens will never be used
		*alreadyFailed = true
	}
}

// for != and ==
func nextRune(target rune, codeRunes []rune, currentPos int) int {
	// -1 if next rune is not target
	// position of target rune after comment if that ever happens
	if currentPos >= len(codeRunes)-1 { // cannot look ahead
		return -1
	}
	if codeRunes[currentPos+1] == target {
		return currentPos + 1
	}
	// CRAZY INSANE scenario where =/*comment*/= should net a ==
	if currentPos+2 < len(codeRunes)-1 && codeRunes[currentPos+1] == '/' && codeRunes[currentPos+2] == '*' {
		for i := currentPos + 2; i < len(codeRunes)-3; i++ {
			if codeRunes[i] == '*' && codeRunes[i+1] == '/' && codeRunes[i+2] == target {
				return i + 2
			}
		}
	}
	return -1
}

func tokenize(capture string, line int, pos int, quoteFlag bool) Token {
	var tokenType TokenType
	var formalName string
	switch capture {
	case "print", "while", "false", "true", "if":
		tokenType = Keyword
		formalName = "KEYW_" + strings.ToUpper(capture)

	case "string", "int", "boolean":
		tokenType = Keyword
		formalName = TypeMap[capture]

	case "==":
		tokenType = Symbol
		formalName = "EQUAL_OP"

	case "!=":
		tokenType = Symbol
		formalName = "N-EQUAL_OP"

	case " ":
		tokenType = Character
		formalName = "CHAR"

	default:
		if len(capture) == 1 && isSymbol(rune(capture[0])) {
			tokenType = Symbol
			formalName = SymbolMap[rune(capture[0])]
		} else if unicode.IsDigit(rune(capture[0])) {
			tokenType = Digit
			formalName = "DIGIT"
		} else {
			// char or identifier is based off quotes
			if quoteFlag {
				tokenType = Character
				formalName = "CHAR"
			} else {
				tokenType = Identifier
				formalName = "ID"
			}
		}
	}

	// no ternary '?' in go :()
	if capture == " " {
		Debug(fmt.Sprintf("%s [ (space) ] found at (%d:%d)", formalName, line, pos), "LEXER")
	} else {
		Debug(fmt.Sprintf("%s [ %s ] found at (%d:%d)", formalName, capture, line, pos), "LEXER")
	}

	token := Token{
		tType: tokenType,
		location: Location{
			line:     line,
			startPos: pos,
		},
		content:     formalName,
		trueContent: capture,
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
	var lastCommentStart int = 0
	var inTokenCommentPos int = 0
	// if comment after EOP but before EOF is unterminated throw err for last program
	var untermEndComment bool = false
	var alreadyErrUntermComment bool = false // so we don't do it twice
	var alreadyFailed bool = false           // when we clear tokens on fail its not empty file issue

	nextProgram(&programNum, &tokenStream, &errorCount, &warningCount, &alreadyFailed)

	// extract tokens
	for lastPos < len(codeRunes) {
		liveRune = codeRunes[currentPos]

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
						"Hint: Capital letters are not permitted in strings.", liveRune, line, currentCol), "LEXER")
				} else if unicode.IsDigit(liveRune) {
					Error(fmt.Sprintf("Invalid character [ %c ] found in quote at (%d:%d); "+
						"Hint: Digits are not permitted in strings.", liveRune, line, currentCol), "LEXER")
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
				if len(tokenBuffer) == 0 { // can't update last pos if stuff exists pre-comment
					lastPos += 2 // close comment is 2 characters
					currentPos = lastPos - 1
				} else {
					currentPos++
					inTokenCommentPos += 2
				}

			} else if liveRune == '\n' {
				// we still need to keep track of line despite comment
				handleNewLine(&line, &lastPos, &deadPos)
			} else {
				if len(tokenBuffer) == 0 {
					lastPos++ // throw it away
				} else {
					inTokenCommentPos++
				}
			}

		} else if isSymbol(liveRune) {
			if len(tokenBuffer) == 0 { // found a symbol to tokenize directly
				// check for == with lookahead
				// we do this crazy logic to ensure =/*COMMENT*/= registers as ==
				var secondEqPos int = nextRune('=', codeRunes, currentPos)
				if liveRune == '=' && secondEqPos != -1 {
					newToken = tokenize(string(liveRune)+string(codeRunes[secondEqPos]), line, lastPos-deadPos+1, quoteFlag)
					lastPos = secondEqPos + 1 // 2 rune symbol
					currentPos = lastPos - 1  // incremented at end of loop
					tokenStream[programNum] = append(tokenStream[programNum], newToken)

				} else {
					// tokenize single symbol
					newToken = tokenize(string(liveRune), line, lastPos-deadPos+1, quoteFlag)
					lastPos++
					currentPos = lastPos - 1
					tokenStream[programNum] = append(tokenStream[programNum], newToken)

					// special cases
					if liveRune == '$' { // EOP
						if nextProgramExists(codeRunes, currentPos, &untermEndComment) {
							passFailProgram(programNum, errorCount, warningCount, tokenStream, &alreadyFailed)
							nextProgram(&programNum, &tokenStream, &errorCount, &warningCount, &alreadyFailed)
						} else if untermEndComment {
							Error("Unterminated comment after EOP.", "LEXER")
							errorCount++
							untermEndComment = false
							alreadyErrUntermComment = true
							passFailProgram(programNum, errorCount, warningCount, tokenStream, &alreadyFailed)
						} else {
							passFailProgram(programNum, errorCount, warningCount, tokenStream, &alreadyFailed)
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

		} else { // didn't find a delimiter
			// check for !=
			// we do this crazy logic to ensure !/*COMMENT*/= registers as !=
			var followingEqPos int = nextRune('=', codeRunes, currentPos)
			if liveRune == '!' && followingEqPos != -1 {
				if len(tokenBuffer) == 0 {
					newToken = tokenize(string(liveRune)+string(codeRunes[followingEqPos]), line, lastPos-deadPos+1, quoteFlag)
					lastPos = followingEqPos + 1 // 2 rune symbol
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
				lastCommentStart = currentPos
				if len(tokenBuffer) == 0 {
					lastPos += 2 // open comment is 2 chars
					currentPos = lastPos - 1
				} else {
					currentPos++
				}
				inTokenCommentPos += 2

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
					} else if liveRune == '!' {
						Error(fmt.Sprintf("Invalid token [ %c ] found at (%d:%d); "+
							"Hint: possible malformed N-EQUAL_OP [ != ]", liveRune, line, lastPos-deadPos+1), "LEXER")
					} else if liveRune == '/' || liveRune == '*' {
						Error(fmt.Sprintf("Invalid token [ %c ] found at (%d:%d); "+
							"Hint: possible malformed comment.", liveRune, line, lastPos-deadPos+1), "LEXER")
					} else {
						Error(fmt.Sprintf("Invalid token [ %c ] found at (%d:%d)", liveRune, line, lastPos-deadPos+1), "LEXER")
					}
					lastPos++
					errorCount++
				}
			}

			// EOF delimiter
			if currentPos >= len(codeRunes)-1 {
				if len(tokenBuffer) > 0 { // no action if buffer empty
					evaluateBuffer = true
				}
			}

		}

		// get what we can from the buffer, tokenize it, jump forward to end of that token, and clean buffer
		if evaluateBuffer {

			greedyCapture = evaluateTokenBuffer(tokenBuffer) // check what we have
			newToken = tokenize(greedyCapture, line, lastPos-deadPos+1, quoteFlag)

			// buffer spans a comment
			if len(tokenBuffer) < currentPos-lastPos {
				// token spans a comment
				if lastPos+len(newToken.trueContent) > lastCommentStart {
					lastPos += len(newToken.trueContent) + inTokenCommentPos
					inTokenCommentPos = 0
				} else {
					lastPos += len(newToken.trueContent) // find the offset based on chars taken
				}
			} else {
				lastPos += len(newToken.trueContent)
			}

			currentPos = lastPos - 1 // incremented at end of loop

			tokenStream[programNum] = append(tokenStream[programNum], newToken)
			tokenBuffer = []rune{} // release old contents
			evaluateBuffer = false
		}

		currentPos++
		if currentPos < lastPos { // ensure we never fall behind - this should never happen
			currentPos = lastPos
		}
	}

	if quoteFlag {
		Error("EOF reached while inside string; Strings must be terminated.", "LEXER")
		errorCount++
	} else if commentFlag && !alreadyErrUntermComment {
		Error("EOF reached while inside comment; Comments must be terminated.", "LEXER")
		errorCount++
	}

	// ensure we have a program with tokens and that it terminated with $
	if len(tokenStream) > 0 && len(tokenStream[len(tokenStream)-1]) > 0 &&
		tokenStream[len(tokenStream)-1][len(tokenStream[len(tokenStream)-1])-1].content != "EOP" {

		Warn("EOF reached before EOP [ $ ]; EOP token was automatically inserted.", "LEXER")
		warningCount++

		// artificially add EOP at end of last line - user will be told where
		newToken = tokenize("$", line, lastPos-deadPos+1, quoteFlag)
		tokenStream[programNum] = append(tokenStream[programNum], newToken)

		passFailProgram(programNum, errorCount, warningCount, tokenStream, &alreadyFailed)

	} else if !alreadyFailed && len(tokenStream[len(tokenStream)-1]) == 0 {
		Warn("Code provided is only whitespace and/or comments! No tokens generated.", "LEXER")
		warningCount++
		Warn("EOF reached before EOP [ $ ]; EOP token was automatically inserted.", "LEXER")
		warningCount++

		// artificially add EOP at end of last line - user will be told where
		newToken = tokenize("$", line, lastPos-deadPos+1, quoteFlag)
		tokenStream[programNum] = append(tokenStream[programNum], newToken)

		passFailProgram(programNum, errorCount, warningCount, tokenStream, &alreadyFailed)
	}
}
