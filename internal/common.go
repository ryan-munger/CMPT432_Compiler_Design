package internal

import (
	"fmt"
	"html"

	"github.com/fatih/color"
)

var (
	webMode   bool // Flag to toggle between CLI and Web mode
	Verbose   bool
	logBuffer string                                // Stores log output for Web mode
	errorMap  map[int]string = make(map[int]string) // remember if a program had to halt and where
)

func hadError(candidate int) bool {
	_, exists := errorMap[candidate]
	return exists
}

func SetVerbose(toggle bool) {
	Verbose = toggle
}

func SetWebMode(toggle bool) {
	webMode = toggle
}

func appendLog(msg string) {
	logBuffer += msg
}

// web mode needs html to render
func Debug(msg string, component string) {
	logMsg := fmt.Sprintf("%-5s | %s --> %s", "DEBUG", component, msg)
	if webMode && Verbose {
		appendLog(fmt.Sprintf(`<span class="text-blue-400">%s</span><br>`, html.EscapeString(logMsg)))
	} else if Verbose {
		fmt.Print(color.BlueString(logMsg + "\n"))
	}
}

func Error(msg string, component string) {
	logMsg := fmt.Sprintf("%-5s | %s --> %s", "ERROR", component, msg)
	if webMode {
		appendLog(fmt.Sprintf(`<span class="text-red-400">%s</span><br>`, html.EscapeString(logMsg)))
	} else {
		color.Red(logMsg)
	}
}

func Warn(msg string, component string) {
	logMsg := fmt.Sprintf("%-5s | %s --> %s", "WARN", component, msg)
	if webMode {
		appendLog(fmt.Sprintf(`<span class="text-yellow-400">%s</span><br>`, html.EscapeString(logMsg)))
	} else {
		color.Yellow(logMsg)
	}
}

func Pass(msg string, component string) {
	logMsg := fmt.Sprintf("%-5s | %s --> %s", "PASS", component, msg)
	if webMode {
		appendLog(fmt.Sprintf(`<span class="text-green-400">%s</span><br>`, html.EscapeString(logMsg)))
	} else {
		color.Green(logMsg)
	}
}

func Fail(msg string, component string) {
	logMsg := fmt.Sprintf("%-5s | %s --> %s", "FAIL", component, msg)
	if webMode {
		appendLog(fmt.Sprintf(`<span class="text-red-500">%s</span><br>`, html.EscapeString(logMsg)))
	} else {
		color.Red(logMsg)
	}
}

func Info(msg string, component string, space bool) {
	logMsg := fmt.Sprintf("%-5s | %s --> %s", "INFO", component, msg)
	if space {
		logMsg = "\n" + logMsg
	}

	if webMode {
		appendLog(fmt.Sprintf(`<span class="text-white">%s</span><br>`, html.EscapeString(logMsg)))
	} else {
		fmt.Print(logMsg + "\n")
	}
}

func CriticalError(location string, err interface{}) {
	errorMsg := fmt.Sprintf("DEFEAT | GOPILER --> Congratulations grand wizard. You have truly bested me.\n"+
		"\tYour code caused a critical error in the %s: %v\n", location, err)
	if webMode {
		appendLog(fmt.Sprintf(`<span class="text-white">%s</span><br>`, errorMsg))
	} else {
		color.Red(errorMsg)
	}
}

// Retrieve log output for web responses
func GetLogOutput() string {
	output := logBuffer
	logBuffer = "" // Clear after reading
	return output
}

// erase everything between compiles in webapp just in case
func ResetAll() {
	logBuffer = ""
	tokens = nil
	liveTokenIdx = 0
	liveToken = Token{}
	parseError = false
	alternateWarning = ""
	pNum = 0
	currentParent = nil
	cstList = nil

	astList = nil
	astStrings = nil
	curAst = nil
	curParent = nil
	parentStack = nil
	stringBuffer = nil
	symbolTableTreeList = nil
	curSymbolTableTree = nil
	curSymbolTable = nil

	errorCount = 0
	inAssign = false
	assignParent = nil
	assignParentScope = ""
	propagateUsed = make(map[*SymbolEntry][]*SymbolUsage)
	// used for re-init before use in case self used (earlier deps no longer unused!)
	dependencyArtifact = nil
	warnCount = 0
	scopeDepth = 0
	scopePopulation = make(map[int]int)
	errorMap = make(map[int]string)

	printTreeBuffer = ""
}

func CreateFailedProgramVars(pNum int, failPoint string) {
	switch failPoint {
	case "lexer":
		startCst(pNum)
		fallthrough
	case "parser":
		initAst(pNum)
		astStrings = append(astStrings, "Error")
		fallthrough
	case "semantic":
		initMem(pNum)
	}
}
