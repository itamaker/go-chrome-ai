package main

import (
	"fmt"
	"os"

	"github.com/itamaker/go-chrome-ai/internal/app"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "gui":
			app.RunGUI()
			return
		case "help", "-h", "--help":
			printUsage()
			return
		case "cli":
			os.Exit(app.RunCLI(args[1:], os.Stderr))
		}
	}

	os.Exit(app.RunCLI(args, os.Stderr))
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  go-chrome-ai [flags]")
	fmt.Println("  go-chrome-ai gui")
	fmt.Println("  go-chrome-ai cli [flags]")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  -dry-run     Show what would change without modifying files")
	fmt.Println("  -no-restart  Do not restart Chrome after patching")
}
