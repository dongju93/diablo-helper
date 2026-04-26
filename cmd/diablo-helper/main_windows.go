//go:build windows

package main

import (
	"os"

	"github.com/dongju93/diablo-helper/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		app.ShowFatalError(err)
		os.Exit(1)
	}
}
