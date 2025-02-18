package internal

func SemanticAnalysis(cst TokenTree, tokenStream []Token, programNum int) {
	// recover from error, pass it up to parser, lexer, main
	defer func() {
		if r := recover(); r != nil {
			CriticalError("semantic analyzer", r)
		}
	}()

	// Info(fmt.Sprintf("Semantically Analyzing program %d", programNum+1), "GOPILER", true)
}
