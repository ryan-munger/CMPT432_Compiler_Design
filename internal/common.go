package internal

import (
	"fmt"
	"html"
	"sync"

	"github.com/fatih/color"
)

var (
	webMode   bool       // Flag to toggle between CLI and Web mode
	logBuffer string     // Stores log output for Web mode
	mu        sync.Mutex // Ensures safe concurrent access
)

// capitalized = export
// lowercase = internal
var Verbose bool

func SetVerbose(toggle bool) {
	Verbose = toggle
}

func SetWebMode(toggle bool) {
	webMode = toggle
}

func appendLog(msg string) {
	mu.Lock()
	logBuffer += msg
	mu.Unlock()
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
	mu.Lock()
	defer mu.Unlock()
	output := logBuffer
	logBuffer = "" // Clear after reading
	return output
}
