package main

import (
	"os"

	"github.com/itamaker/go-chrome-ai/internal/app"
	"github.com/itamaker/go-chrome-ai/internal/guiapp"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "gui":
			guiapp.Run()
			return
		case "help", "-h", "--help":
			app.Usage(os.Stdout)
			return
		case "cli":
			os.Exit(app.RunCLI(args[1:], os.Stdout, os.Stderr))
			return
		}
	}

	os.Exit(app.RunCLI(args, os.Stdout, os.Stderr))
}
