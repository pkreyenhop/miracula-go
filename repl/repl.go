package repl

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unicode"
	"unsafe"
	"pkreyenhop.com/miracula-go/ast"
	"pkreyenhop.com/miracula-go/lexer"
	"pkreyenhop.com/miracula-go/parser"
	"pkreyenhop.com/miracula-go/eval"
)

func IsTTY() bool {
	cmd := exec.Command("stty", "-g")
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	return err == nil
}

func pendingBytes() int {
	var limit int
	_, _, err := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(0), // stdin fd
		uintptr(0x541b), // FIONREAD / TIOCINQ ioctl code on Linux
		uintptr(unsafe.Pointer(&limit)),
	)
	if err != 0 {
		return 0
	}
	return limit
}

func hasMore(r *bufio.Reader) bool {
	if r.Buffered() > 0 {
		return true
	}
	return pendingBytes() > 0
}

func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func readLine(prompt string, history []string, env *ast.Env) (string, []string, bool) {
	cmd := exec.Command("stty", "raw", "-echo")
	cmd.Stdin = os.Stdin
	_ = cmd.Run()

	defer func() {
		restoreCmd := exec.Command("stty", "-raw", "echo")
		restoreCmd.Stdin = os.Stdin
		_ = restoreCmd.Run()
	}()

	var buf []rune
	cursor := 0
	historyIdx := len(history)
	var draft []rune

	var lastTabCandidates []string
	var lastTabIdx int
	var lastTabStart int

	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)

	for {
		r, _, err := reader.ReadRune()
		if err != nil {
			return "", history, false
		}

		if r != 9 {
			lastTabCandidates = nil
		}

		switch r {
		case 3: // Ctrl-C
			fmt.Print("\r\n")
			return "", history, true
		case 4: // Ctrl-D
			if len(buf) == 0 {
				fmt.Print("\r\n")
				return "", history, false
			}
			if cursor < len(buf) {
				buf = append(buf[:cursor], buf[cursor+1:]...)
			}
		case 1: // Ctrl-A
			cursor = 0
		case 5: // Ctrl-E
			cursor = len(buf)
		case 11: // Ctrl-K
			buf = buf[:cursor]
		case 9: // Tab Completion
			if len(lastTabCandidates) > 0 {
				lastTabIdx = (lastTabIdx + 1) % len(lastTabCandidates)
				cand := lastTabCandidates[lastTabIdx]
				buf = append(buf[:lastTabStart], append([]rune(cand), buf[cursor:]...)...)
				cursor = lastTabStart + len(cand)
			} else {
				start := cursor
				for start > 0 && isWordChar(buf[start-1]) {
					start--
				}
				prefix := string(buf[start:cursor])
				if len(prefix) > 0 {
					var all []string
					all = append(all, []string{"where", "if", "then", "else", "otherwise", "mod"}...)
					all = append(all, []string{"hd", "tl", "show", "read", "lines", "numval", "length"}...)
					if env != nil {
						all = append(all, env.GetNames()...)
					}

					seen := make(map[string]bool)
					var candidates []string
					for _, item := range all {
						if strings.HasPrefix(item, prefix) && !seen[item] {
							seen[item] = true
							candidates = append(candidates, item)
						}
					}

					if len(candidates) > 0 {
						lastTabCandidates = candidates
						lastTabIdx = 0
						lastTabStart = start
						cand := candidates[0]
						buf = append(buf[:start], append([]rune(cand), buf[cursor:]...)...)
						cursor = start + len(cand)
					}
				}
			}
		case 13, 10: // Enter
			fmt.Print("\r\n")
			line := string(buf)
			if len(line) > 0 && (len(history) == 0 || history[len(history)-1] != line) {
				history = append(history, line)
			}
			return line, history, true
		case 8, 127: // Backspace
			if cursor > 0 {
				buf = append(buf[:cursor-1], buf[cursor:]...)
				cursor--
			}
		case 27: // Escape
			if hasMore(reader) {
				r2, _, _ := reader.ReadRune()
				if r2 == '[' {
					r3, _, _ := reader.ReadRune()
					switch r3 {
					case 'A': // Up Arrow
						if historyIdx > 0 {
							if historyIdx == len(history) {
								draft = append([]rune(nil), buf...)
							}
							historyIdx--
							buf = []rune(history[historyIdx])
							cursor = len(buf)
						}
					case 'B': // Down Arrow
						if historyIdx < len(history) {
							historyIdx++
							if historyIdx == len(history) {
								buf = append([]rune(nil), draft...)
							} else {
								buf = []rune(history[historyIdx])
							}
							cursor = len(buf)
						}
					case 'C': // Right Arrow
						if cursor < len(buf) {
							cursor++
						}
					case 'D': // Left Arrow
						if cursor > 0 {
							cursor--
						}
					case 'H': // Home
						cursor = 0
					case 'F': // End
						cursor = len(buf)
					case '1', '2', '3', '4', '5', '6', '7', '8', '9':
						r4, _, _ := reader.ReadRune()
						if r4 == '~' {
							if r3 == '3' { // Delete
								if cursor < len(buf) {
									buf = append(buf[:cursor], buf[cursor+1:]...)
								}
							}
						}
					}
				} else if r2 == 'O' {
					r3, _, _ := reader.ReadRune()
					switch r3 {
					case 'H': // Home
						cursor = 0
					case 'F': // End
						cursor = len(buf)
					}
				}
			}
		default:
			if r >= 32 {
				buf = append(buf[:cursor], append([]rune{r}, buf[cursor:]...)...)
				cursor++
			}
		}

		fmt.Printf("\r%s%s\x1b[K\r\x1b[%dG", prompt, string(buf), len(prompt)+cursor+1)
	}
}

func LoadScriptFile(filename string, env *ast.Env) (*ast.Env, error) {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		if filename == "stdenv.m" {
			fmt.Println("Standard environment file 'stdenv.m' not found. Skipping.")
			return env, nil
		}
		fmt.Printf("Script file '%s' not found. Starting with empty space.\n", filename)
		return env, nil
	}

	lines := strings.Split(string(bytes), "\n")
	var layoutLines []lexer.LayoutLine

	for _, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "||") {
			continue
		}

		indent := 0
		for _, r := range line {
			if r == ' ' {
				indent++
			} else if r == '\t' {
				indent += 4
			} else {
				break
			}
		}

		runes := []rune(line)
		dropCount := 0
		tempIndent := 0
		for dropCount < len(runes) {
			r := runes[dropCount]
			if r == ' ' {
				tempIndent++
				dropCount++
			} else if r == '\t' {
				tempIndent += 4
				dropCount++
			} else {
				break
			}
		}
		lineContent := string(runes[dropCount:])

		lineToks := lexer.Tokenize(lineContent)
		var filtered []lexer.Token
		for _, t := range lineToks {
			if t.Type != lexer.TOK_EOF {
				filtered = append(filtered, t)
			}
		}

		wrapped := lexer.WrapWhereOnLine(filtered)
		if len(wrapped) > 0 {
			layoutLines = append(layoutLines, lexer.LayoutLine{Indent: indent, Toks: wrapped})
		}
	}

	fileTokens := lexer.ApplyLayout(layoutLines)
	segments := lexer.SplitTokens(fileTokens)

	var bindings []parser.RawBinding
	for _, seg := range segments {
		err := func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					var tokStrs []string
					for _, t := range seg {
						tokStrs = append(tokStrs, lexer.TokenToString(t))
					}
					err = fmt.Errorf("parse error in segment:\n%s\nDetails: %v", strings.Join(tokStrs, " "), r)
				}
			}()
			p := parser.NewParser(seg)
			stmt := p.Parse()
			if bind, ok := stmt.(parser.ScriptBindStmt); ok {
				bindings = append(bindings, bind.Binding)
			} else {
				panic("invalid expression structure in script file")
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}

	grouped := make(map[string][]parser.RawBinding)
	var order []string
	for _, b := range bindings {
		if _, ok := grouped[b.FName]; !ok {
			order = append(order, b.FName)
		}
		grouped[b.FName] = append(grouped[b.FName], b)
	}

	accEnv := env
	for _, name := range order {
		eqList := grouped[name]
		desugaredLambda := parser.DesugarEquations(eqList)
		accEnv = accEnv.Extend(name, desugaredLambda)
	}

	return accEnv, nil
}

func RunREPLDirect(env *ast.Env, scriptFile string) {
	interactive := IsTTY()
	var history []string
	var scanner *bufio.Scanner
	if !interactive {
		scanner = bufio.NewScanner(os.Stdin)
	}

	for {
		var line string
		if interactive {
			var ok bool
			line, history, ok = readLine("miranda> ", history, env)
			if !ok {
				fmt.Println("Goodbye.")
				break
			}
		} else {
			fmt.Print("miranda> ")
			if !scanner.Scan() {
				fmt.Println("Goodbye.")
				break
			}
			line = scanner.Text()
		}

		lineTrimmed := strings.TrimSpace(line)
		if lineTrimmed == "" {
			continue
		}
		if lineTrimmed == "/q" || lineTrimmed == "exit" || lineTrimmed == "quit" {
			if interactive {
				// Goodbye already printed by readLine / Enter / EOF loop
			} else {
				fmt.Println("Goodbye.")
			}
			break
		}
		if lineTrimmed == "/e" {
			fmt.Printf("Opening vi %s ...\n", scriptFile)
			cmd := exec.Command("vi", scriptFile)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			_ = cmd.Run()
			fmt.Printf("Reloading environment profiles from %s...\n", scriptFile)
			envWithStd, _ := LoadScriptFile("stdenv.m", &ast.Env{})
			reloadedEnv, err := LoadScriptFile(scriptFile, envWithStd)
			if err != nil {
				fmt.Printf("Error reloading: %v\n", err)
			} else {
				env = reloadedEnv
			}
			continue
		}

		tokens := lexer.Tokenize(lineTrimmed)
		func() {
			defer func() {
				if r := recover(); r != nil {
					if rtErr, ok := r.(ast.RuntimeError); ok {
						fmt.Printf("Runtime Error: %s\n", rtErr.Msg)
					} else if bhErr, ok := r.(ast.BlackholeError); ok {
						fmt.Printf("Runtime Error: %s\n", bhErr.Msg)
					} else {
						fmt.Printf("Error: %v\n", r)
					}
				}
			}()
			p := parser.NewParser(tokens)
			stmt := p.Parse()
			switch s := stmt.(type) {
			case parser.ScriptBindStmt:
				finalLambda := parser.DesugarEquations([]parser.RawBinding{s.Binding})
				env = env.Extend(s.Binding.FName, finalLambda)
				fmt.Printf("Defined variable: %s\n", s.Binding.FName)
			case parser.REPLEvalStmt:
				startTime := time.Now()
				result := eval.Whnf(env, s.Expr)
				duration := time.Since(startTime).Milliseconds()

				sVal, isStr := eval.IsString(env, result)
				if isStr {
					fmt.Printf("Result:\n%s", sVal)
					if len(sVal) > 0 && sVal[len(sVal)-1] == '\n' {
						// no extra newline
					} else {
						fmt.Println()
					}
				} else {
					fmt.Printf("Result: %s\n", eval.PrintNode(env, result))
				}
				fmt.Printf("Evaluation time: %d ms\n", duration)
			}
		}()
	}
}
