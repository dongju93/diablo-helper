//go:build windows

package main

import (
	"fmt"
	"os"

	"github.com/dongju93/diablo-helper/internal/app"
)

func main() {
	defer func() {
		if value := recover(); value != nil {
			app.ShowFatalError(fmt.Errorf("fatal startup panic: %v", value))
			os.Exit(1)
		}
	}()

	if err := app.Run(); err != nil {
		app.ShowFatalError(err)
		os.Exit(1)
	}
}
