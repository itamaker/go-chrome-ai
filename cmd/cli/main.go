package main

import (
	"os"

	"github.com/itamaker/go-chrome-ai/internal/app"
)

func main() {
	os.Exit(app.RunCLI(os.Args[1:], os.Stderr))
}
