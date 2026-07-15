package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"runtime/pprof"
	"syscall"
	"pkreyenhop.com/miracula-go/ast"
	"pkreyenhop.com/miracula-go/repl"
	"pkreyenhop.com/miracula-go/typecheck"
)

func main() {
	// Evaluation allocates heavily (thunks, cons cells, environment frames)
	// and runs briefly, so the default GC target (100%) spends most of its
	// time collecting short-lived garbage. A higher target roughly halves the
	// time on allocation-bound workloads at the cost of ~4x peak heap. Respect
	// an explicit GOGC if the user set one.
	if os.Getenv("GOGC") == "" {
		debug.SetGCPercent(400)
	}

	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to `file`")
	execX := flag.String("x", "", "evaluate FILE or COMMAND and exit")
	execT := flag.String("t", "", "evaluate FILE or COMMAND and exit")
	execXT := flag.String("xt", "", "evaluate FILE or COMMAND and exit with timing info")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of miracula:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  miracula [flags] [script_file]\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Flags:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  -cpuprofile string\n")
		fmt.Fprintf(flag.CommandLine.Output(), "    \twrite cpu profile to file\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  -x string\n")
		fmt.Fprintf(flag.CommandLine.Output(), "    \tevaluate FILE or COMMAND and print result only\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  -t string\n")
		fmt.Fprintf(flag.CommandLine.Output(), "    \tevaluate FILE or COMMAND and print evaluation time only\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  -xt string\n")
		fmt.Fprintf(flag.CommandLine.Output(), "    \tevaluate FILE or COMMAND and print both result and evaluation time\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  -h, -?, --help\n")
		fmt.Fprintf(flag.CommandLine.Output(), "    \tshow this help message\n")
	}

	for _, arg := range os.Args[1:] {
		if arg == "-?" || arg == "-h" || arg == "--help" {
			flag.Usage()
			os.Exit(0)
		}
	}

	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fmt.Printf("Error creating CPU profile: %v\n", err)
			os.Exit(1)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Printf("Error starting CPU profile: %v\n", err)
			f.Close()
			os.Exit(1)
		}
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			pprof.StopCPUProfile()
			f.Close()
			os.Exit(1)
		}()
		defer pprof.StopCPUProfile()
		defer f.Close()
		// EvaluateAndExit terminates via os.Exit, which skips the defers
		// above — flush the profile through the REPL exit hook instead.
		repl.OnExit = func() {
			pprof.StopCPUProfile()
			f.Close()
		}
	}

	args := flag.Args()
	scriptFile := "~/.script.m"
	if len(args) == 1 {
		scriptFile = args[0]
	} else if len(args) > 1 {
		fmt.Println("Usage: miracula [-cpuprofile=file] [-x|-t|-xt file/command] [script_file]")
		os.Exit(1)
	}

	isReplMode := len(args) == 0

	env := ast.NewEnv()
	typeEnv := typecheck.DefaultTypeEnv()

	if nextEnv, nextTypeEnv, err := repl.LoadScriptFile("stdenv.m", env, typeEnv); err != nil {
		fmt.Printf("Error loading stdenv.m: %v\n", err)
		os.Exit(1)
	} else {
		env = nextEnv
		typeEnv = nextTypeEnv
	}

	if nextEnv, nextTypeEnv, err := repl.LoadScriptFile("~/.script.m", env, typeEnv); err != nil {
		fmt.Printf("Error loading ~/.script.m: %v\n", err)
	} else {
		env = nextEnv
		typeEnv = nextTypeEnv
	}

	if *execXT != "" {
		repl.EvaluateAndExit(env, typeEnv, *execXT, true, true)
	}
	if *execX != "" {
		repl.EvaluateAndExit(env, typeEnv, *execX, true, false)
	}
	if *execT != "" {
		repl.EvaluateAndExit(env, typeEnv, *execT, false, true)
	}

	if repl.ExpandHome(scriptFile) != repl.ExpandHome("~/.script.m") {
		if nextEnv, nextTypeEnv, err := repl.LoadScriptFile(scriptFile, env, typeEnv); err != nil {
			fmt.Printf("Error loading %s: %v\n", scriptFile, err)
		} else {
			env = nextEnv
			typeEnv = nextTypeEnv
		}
	}

	if isReplMode {
		fmt.Println("==================================================")
		fmt.Println(" Environment-Sharing Go REPL                     ")
		fmt.Println(" Use '/e' to edit ~/.script.m, '/q' to exit       ")
		fmt.Println("==================================================")
	} else {
		fmt.Println("==================================================")
		fmt.Printf(" Loaded file: %s                  \n", scriptFile)
		fmt.Printf(" Use '/e' to edit %s, '/q' to exit\n", scriptFile)
		fmt.Println("==================================================")
	}

	repl.RunREPLDirect(env, typeEnv, scriptFile)
}
