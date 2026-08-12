package main

import (
	"fmt"
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
	fmt.Println("  -dry-run              Show what would change without modifying files")
	fmt.Println("  -no-restart           Do not restart Chrome after patching")
	fmt.Println("  -disable-ai-download  Block on-device Gemini Nano download by disabling all")
	fmt.Println("                        known chrome://flags entries plus the OS policy (default true)")
	fmt.Println("  -disable-flag         Individual chrome://flags entry to disable (repeatable);")
	fmt.Println("                        only takes effect when -disable-ai-download=false")
	fmt.Println("  -disable-ai-policy    Write the GenAILocalFoundationalModelSettings Enterprise")
	fmt.Println("                        policy (chrome://policy), independent of -disable-flag;")
	fmt.Println("                        only takes effect when -disable-ai-download=false (default true)")
	fmt.Println("")
	fmt.Println("Every run syncs to the current selection: a deselected flag or policy is")
	fmt.Println("actively reverted (chrome://flags reset to default, chrome://policy entry")
	fmt.Println("removed) if this tool previously set it.")
}
