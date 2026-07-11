package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
	"pkreyenhop.com/miracula-go/ast"
	"pkreyenhop.com/miracula-go/repl"
)

func main() {
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to `file`")
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fmt.Printf("Error creating CPU profile: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Printf("Error starting CPU profile: %v\n", err)
			os.Exit(1)
		}
		defer pprof.StopCPUProfile()
	}

	args := flag.Args()
	scriptFile := "script.m"
	if len(args) == 1 {
		scriptFile = args[0]
	} else if len(args) > 1 {
		fmt.Println("Usage: miracula [-cpuprofile=file] [script_file]")
		os.Exit(1)
	}

	isReplMode := scriptFile == "script.m"

	env := ast.NewEnv()
	var err error
	stdenvEnv, err := repl.LoadScriptFile("stdenv.m", env)
	if err != nil {
		fmt.Printf("Error loading stdenv.m: %v\n", err)
		os.Exit(1)
	}
	env = stdenvEnv

	scriptEnv, err := repl.LoadScriptFile(scriptFile, env)
	if err != nil {
		fmt.Printf("Error loading %s: %v\n", scriptFile, err)
	} else {
		env = scriptEnv
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
