package main

import (
	"fmt"
	"os"
	"pkreyenhop.com/miracula-go/ast"
	"pkreyenhop.com/miracula-go/repl"
)

func main() {
	args := os.Args[1:]
	scriptFile := "script.m"
	if len(args) == 1 {
		scriptFile = args[0]
	} else if len(args) > 1 {
		fmt.Println("Usage: miracula [script_file]")
		os.Exit(1)
	}

	isReplMode := scriptFile == "script.m"

	env := &ast.Env{}
	var err error
	env, err = repl.LoadScriptFile("stdenv.m", env)
	if err != nil {
		fmt.Printf("Error loading stdenv.m: %v\n", err)
		os.Exit(1)
	}

	env, err = repl.LoadScriptFile(scriptFile, env)
	if err != nil {
		fmt.Printf("Error loading %s: %v\n", scriptFile, err)
		os.Exit(1)
	}

	if isReplMode {
		fmt.Println("==================================================")
		fmt.Println(" Environment-Sharing Go REPL                     ")
		fmt.Println(" Use '/e' to edit script.m, '/q' to exit          ")
		fmt.Println("==================================================")
	} else {
		fmt.Println("==================================================")
		fmt.Printf(" Loaded file: %s                  \n", scriptFile)
		fmt.Printf(" Use '/e' to edit %s, '/q' to exit\n", scriptFile)
		fmt.Println("==================================================")
	}

	repl.RunREPLDirect(env, scriptFile)
}
