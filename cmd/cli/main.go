package main

import (
	"os"

	"github.com/itamaker/go-chrome-ai/internal/app"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "help", "-h", "--help":
			app.Usage(os.Stdout)
			return
		}
	}
	os.Exit(app.RunCLI(args, os.Stdout, os.Stderr))
}
