package main

import (
	"os"

	"go-chrome-ai/internal/app"
)

func main() {
	os.Exit(app.RunCLI(os.Args[1:], os.Stderr))
}
