package repl

import (
	"bufio"
	"errors"
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
	"pkreyenhop.com/miracula-go/typecheck"
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
				insertedTab := false
				if len(prefix) > 0 {
					var all []string
					all = append(all, []string{"where", "if", "then", "else", "otherwise", "mod"}...)
					all = append(all, []string{"hd", "tl", "show", "read", "lines", "numval", "length", "reverse"}...)
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
					} else {
						insertedTab = true
					}
				} else {
					insertedTab = true
				}

				if insertedTab {
					buf = append(buf[:cursor], append([]rune{'\t'}, buf[cursor:]...)...)
					cursor++
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

		visualCursor := len(prompt)
		for i := 0; i < cursor && i < len(buf); i++ {
			if buf[i] == '\t' {
				visualCursor += 8 - (visualCursor % 8)
			} else {
				visualCursor++
			}
		}

		fmt.Printf("\r%s%s\x1b[K\r\x1b[%dG", prompt, string(buf), visualCursor+1)
	}
}

func FormatParseError(filename string, fileContent string, pe parser.ParseError) string {
	lines := strings.Split(fileContent, "\n")
	lineNum := pe.Tok.Line
	colNum := pe.Tok.Col

	if lineNum < 1 {
		lineNum = 1
	}
	if lineNum > len(lines) {
		lineNum = len(lines)
	}

	var lineStr string
	if len(lines) > 0 && lineNum-1 < len(lines) {
		lineStr = strings.TrimSuffix(lines[lineNum-1], "\r")
	}

	linePrefix := fmt.Sprintf("  %d | ", lineNum)
	indentPrefix := strings.Repeat(" ", len(fmt.Sprintf("  %d ", lineNum))) + "| "

	var caretSpace strings.Builder
	runes := []rune(lineStr)
	for i := 0; i < colNum-1 && i < len(runes); i++ {
		if runes[i] == '\t' {
			caretSpace.WriteString("\t")
		} else {
			caretSpace.WriteString(" ")
		}
	}

	return fmt.Sprintf("%s:%d:%d: %s\n%s%s\n%s%s^",
		filename, lineNum, colNum, pe.Error(),
		linePrefix, lineStr,
		indentPrefix, caretSpace.String())
}

func FormatTypeError(filename string, fileContent string, te *typecheck.TypeError) string {
	posVal, found := ast.NodePositions.Load(ast.GetNodeKey(te.Node))
	lineNum, colNum := 1, 1
	if found {
		if pos, ok := posVal.(ast.Position); ok {
			lineNum = pos.Line
			colNum = pos.Col
		}
	}

	lines := strings.Split(fileContent, "\n")
	if lineNum < 1 {
		lineNum = 1
	}
	if lineNum > len(lines) {
		lineNum = len(lines)
	}

	var lineStr string
	if len(lines) > 0 && lineNum-1 < len(lines) {
		lineStr = strings.TrimSuffix(lines[lineNum-1], "\r")
	}

	linePrefix := fmt.Sprintf("  %d | ", lineNum)
	indentPrefix := strings.Repeat(" ", len(fmt.Sprintf("  %d ", lineNum))) + "| "

	var caretSpace strings.Builder
	runes := []rune(lineStr)
	for i := 0; i < colNum-1 && i < len(runes); i++ {
		if runes[i] == '\t' {
			caretSpace.WriteString("\t")
		} else {
			caretSpace.WriteString(" ")
		}
	}

	return fmt.Sprintf("%s:%d:%d: Type Error: %s\n%s%s\n%s%s^",
		filename, lineNum, colNum, te.Err.Error(),
		linePrefix, lineStr,
		indentPrefix, caretSpace.String())
}

func loadHistory() []string {
	bytes, err := os.ReadFile("history.m")
	if err != nil {
		return nil
	}
	lines := strings.Split(string(bytes), "\n")
	var history []string
	for _, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		if line != "" {
			history = append(history, line)
		}
	}
	if len(history) > 200 {
		history = history[len(history)-200:]
	}
	return history
}

func saveHistory(history []string) {
	if len(history) > 200 {
		history = history[len(history)-200:]
	}
	content := strings.Join(history, "\n")
	if len(history) > 0 {
		content += "\n"
	}
	_ = os.WriteFile("history.m", []byte(content), 0644)
}

func LoadScriptFile(filename string, env *ast.Env, typeEnv *typecheck.TypeEnv) (*ast.Env, *typecheck.TypeEnv, error) {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		if filename == "stdenv.m" {
			fmt.Println("Standard environment file 'stdenv.m' not found. Skipping.")
			return env, typeEnv, nil
		}
		fmt.Printf("Script file '%s' not found. Starting with empty space.\n", filename)
		return env, typeEnv, nil
	}

	lines := strings.Split(string(bytes), "\n")
	var layoutLines []lexer.LayoutLine

	for lineIdx, line := range lines {
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

		lineToks := lexer.TokenizeWithPos(lineContent, lineIdx+1)
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
					if pe, ok := r.(parser.ParseError); ok {
						err = fmt.Errorf("%s", FormatParseError(filename, string(bytes), pe))
					} else {
						var tokStrs []string
						for _, t := range seg {
							tokStrs = append(tokStrs, lexer.TokenToString(t))
						}
						err = fmt.Errorf("parse error in segment:\n%s\nDetails: %v", strings.Join(tokStrs, " "), r)
					}
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
			return nil, nil, err
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
	accTypeEnv := typeEnv
	tc := typecheck.NewTypeChecker()
	sCurr := typecheck.Substitution(nil)

	for _, name := range order {
		eqList := grouped[name]
		desugaredLambda := parser.DesugarEquations(eqList)

		selfTy := tc.Fresh()
		tcEnv := accTypeEnv.Extend(name, typecheck.Scheme{Vars: nil, Ty: selfTy})
		tB, sNext, err := tc.Infer(tcEnv, desugaredLambda, sCurr)
		if err != nil {
			var te *typecheck.TypeError
			if errors.As(err, &te) {
				return nil, nil, fmt.Errorf("%s", FormatTypeError(filename, string(bytes), te))
			}
			return nil, nil, fmt.Errorf("Type Error in '%s': %w", name, err)
		}
		sNext2, err := sNext.Unify(selfTy, tB)
		if err != nil {
			var te *typecheck.TypeError
			if errors.As(err, &te) {
				return nil, nil, fmt.Errorf("%s", FormatTypeError(filename, string(bytes), te))
			}
			return nil, nil, fmt.Errorf("Type Error in '%s': %w", name, err)
		}
		sCurr = sNext2

		finalTy := sCurr.Apply(selfTy)
		scheme := typecheck.Generalize(sCurr.ApplyEnv(accTypeEnv), finalTy)
		accTypeEnv = accTypeEnv.Extend(name, scheme)
		accEnv = accEnv.Extend(name, desugaredLambda)
	}

	return accEnv, accTypeEnv, nil
}

func RunREPLDirect(env *ast.Env, typeEnv *typecheck.TypeEnv, scriptFile string) {
	interactive := IsTTY()
	var history []string
	if interactive {
		history = loadHistory()
	}
	var scanner *bufio.Scanner
	if !interactive {
		scanner = bufio.NewScanner(os.Stdin)
	}

	for {
		var firstLine string
		var ok bool
		if interactive {
			firstLine, history, ok = readLine("miranda> ", history, env)
			if ok {
				saveHistory(history)
			}
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
			firstLine = scanner.Text()
		}

		lineTrimmed := strings.TrimSpace(firstLine)
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
			editor := "./mica"
			if _, err := os.Stat(editor); err != nil {
				editor = "vi"
			}
			fmt.Printf("Opening %s %s ...\n", editor, scriptFile)
			cmd := exec.Command(editor, scriptFile)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			_ = cmd.Run()
			fmt.Printf("Reloading environment profiles from %s...\n", scriptFile)
			envWithStd, typeEnvWithStd, _ := LoadScriptFile("stdenv.m", ast.NewEnv(), typecheck.DefaultTypeEnv())
			reloadedEnv, reloadedTypeEnv, err := LoadScriptFile(scriptFile, envWithStd, typeEnvWithStd)
			if err != nil {
				fmt.Printf("Error reloading: %v\n", err)
			} else {
				env = reloadedEnv
				typeEnv = reloadedTypeEnv
			}
			continue
		}

		var lines []string
		currentLine := firstLine
		for {
			trimmedRight := strings.TrimRightFunc(currentLine, unicode.IsSpace)
			if strings.HasSuffix(trimmedRight, "\\\\") {
				lineWithoutSlash := strings.TrimSuffix(trimmedRight, "\\\\")
				lines = append(lines, lineWithoutSlash)

				var nextLine string
				var nextOk bool
				promptStr := "miranda> "
				continuationPrompt := strings.Repeat(" ", len(promptStr)-2) + "> "
				if interactive {
					nextLine, history, nextOk = readLine(continuationPrompt, history, env)
					if nextOk {
						saveHistory(history)
					}
					if !nextOk {
						break
					}
				} else {
					fmt.Print(continuationPrompt)
					if !scanner.Scan() {
						break
					}
					nextLine = scanner.Text()
				}
				currentLine = nextLine
			} else {
				lines = append(lines, currentLine)
				break
			}
		}

		fullInput := strings.Join(lines, "\n")
		fullInputTrimmed := strings.TrimSpace(fullInput)
		if fullInputTrimmed == "" {
			continue
		}

		var tokens []lexer.Token
		inputLines := strings.Split(fullInput, "\n")
		var layoutLines []lexer.LayoutLine
		for lineIdx, lineText := range inputLines {
			indent := 0
			for _, r := range lineText {
				if r == ' ' {
					indent++
				} else if r == '\t' {
					indent += 4
				} else {
					break
				}
			}

			lineToks := lexer.TokenizeWithPos(lineText, lineIdx+1)
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
		tokens = lexer.ApplyLayout(layoutLines)

		func() {
			defer func() {
				if r := recover(); r != nil {
					var te *typecheck.TypeError
					if errVal, ok := r.(error); ok && errors.As(errVal, &te) {
						fmt.Println(FormatTypeError("<stdin>", fullInput, te))
					} else if rtErr, ok := r.(ast.RuntimeError); ok {
						if strings.HasPrefix(rtErr.Msg, "Type Error:") {
							fmt.Println(rtErr.Msg)
						} else {
							fmt.Printf("Runtime Error: %s\n", rtErr.Msg)
						}
					} else if bhErr, ok := r.(ast.BlackholeError); ok {
						fmt.Printf("Runtime Error: %s\n", bhErr.Msg)
					} else if pe, ok := r.(parser.ParseError); ok {
						fmt.Println(FormatParseError("<stdin>", fullInput, pe))
					} else {
						fmt.Printf("Error: %v\n", r)
					}
				}
			}()
			
			segments := lexer.SplitTokens(tokens)
			if len(segments) == 0 {
				return
			}

			var bindings []parser.RawBinding
			var evalStmt parser.REPLEvalStmt
			isMultiBind := false

			for _, seg := range segments {
				p := parser.NewParser(seg)
				stmt := p.Parse()
				switch s := stmt.(type) {
				case parser.ScriptBindStmt:
					bindings = append(bindings, s.Binding)
					isMultiBind = true
				case parser.REPLEvalStmt:
					if isMultiBind {
						panic(ast.RuntimeError{Msg: "Cannot mix binding statements and evaluation expressions"})
					}
					evalStmt = s
				}
			}

			if isMultiBind {
				grouped := make(map[string][]parser.RawBinding)
				var order []string
				for _, b := range bindings {
					if _, ok := grouped[b.FName]; !ok {
						order = append(order, b.FName)
					}
					grouped[b.FName] = append(grouped[b.FName], b)
				}

				tc := typecheck.NewTypeChecker()
				sCurr := typecheck.Substitution(nil)
				accTypeEnv := typeEnv
				accEnv := env

				for _, name := range order {
					eqList := grouped[name]
					finalLambda := parser.DesugarEquations(eqList)

					selfTy := tc.Fresh()
					tcEnv := accTypeEnv.Extend(name, typecheck.Scheme{Vars: nil, Ty: selfTy})
					tB, sNext, err := tc.Infer(tcEnv, finalLambda, sCurr)
					if err != nil {
						panic(err)
					}
					sNext2, err := sNext.Unify(selfTy, tB)
					if err != nil {
						panic(err)
					}
					sCurr = sNext2

					finalTy := sCurr.Apply(selfTy)
					scheme := typecheck.Generalize(sCurr.ApplyEnv(accTypeEnv), finalTy)
					accTypeEnv = accTypeEnv.Extend(name, scheme)
					accEnv = accEnv.Extend(name, finalLambda)
				}

				for _, name := range order {
					fmt.Printf("Defined variable: %s\n", name)
				}
				env = accEnv
				typeEnv = accTypeEnv

				// Keep REPL definitions in scriptFile
				f, err := os.OpenFile(scriptFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err == nil {
					toWrite := fullInput
					if !strings.HasSuffix(toWrite, "\n") {
						toWrite += "\n"
					}
					_, _ = f.WriteString(toWrite)
					_ = f.Close()
				}
			} else {
				tc := typecheck.NewTypeChecker()
				_, _, err := tc.Infer(typeEnv, evalStmt.Expr, nil)
				if err != nil {
					panic(err)
				}

				startTime := time.Now()
				result := eval.Whnf(env, evalStmt.Expr)
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
