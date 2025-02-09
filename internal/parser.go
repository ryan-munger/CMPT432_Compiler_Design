package internal

func Parse(tokenStream []Token, programNum int) {
	// Info(fmt.Sprintf("Parsing program %d", programNum), "GOPILER", true)
	defer func() {
		if r := recover(); r != nil {
			CriticalError("parser", r)
		}
	}()

}
