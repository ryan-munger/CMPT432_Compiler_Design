package internal

import (
	"fmt"

	"github.com/fatih/color"
)

type Token struct {
	location string
}

// capitalized = export
// lowercase = internal
func Log(msg string) {
	fmt.Println(msg)
}

func Error(msg string) {
	color.Red(msg)
}

func Warn(msg string) {
	color.Yellow(msg)
}

func Success(msg string) {
	color.Green(msg)
}

func Debug(msg string) {
	color.Blue(msg)
}
