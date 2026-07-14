package repl

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"pkreyenhop.com/miracula-go/ast"
	"pkreyenhop.com/miracula-go/eval"
	"pkreyenhop.com/miracula-go/lexer"
	"pkreyenhop.com/miracula-go/parser"
	"pkreyenhop.com/miracula-go/typecheck"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"
	"unsafe"
)

var (
	lastErrorLine int
	lastErrorCol  int
)

// ExpandHome resolves a Unix home directory prefix (`~`) in the given path,
// returning the absolute path or the original path if the home directory
// cannot be determined or if there is no prefix.
func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			if len(path) > 1 && (path[1] == '/' || path[1] == '\\') {
				return filepath.Join(home, path[2:])
			}
		}
	}
	return path
}

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
		uintptr(0),      // stdin fd
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
	lastErrorLine = 0
	lastErrorCol = 0
	bytes, err := os.ReadFile(ExpandHome(filename))
	if err != nil {
		if filename == "stdenv.m" {
			fmt.Println("Standard environment file 'stdenv.m' not found. Skipping.")
			return env, typeEnv, nil
		}
		if filename == "~/.script.m" {
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
						lastErrorLine = pe.Tok.Line
						lastErrorCol = pe.Tok.Col
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
			p := parser.NewParser(seg).WithFilename(filename)
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
		if len(eqList) > 0 {
			bodyKey := ast.GetNodeKey(eqList[0].Body)
			if posVal, found := ast.NodePositions.Load(bodyKey); found {
				ast.NodePositions.Store(ast.GetNodeKey(desugaredLambda), posVal)
			}
		}

		selfTy := tc.Fresh()
		tcEnv := accTypeEnv.Extend(name, typecheck.Scheme{Vars: nil, Ty: selfTy})
		tB, sNext, err := tc.Infer(tcEnv, desugaredLambda, sCurr)
		if err != nil {
			var te *typecheck.TypeError
			if errors.As(err, &te) {
				posVal, found := ast.NodePositions.Load(ast.GetNodeKey(te.Node))
				if found {
					if pos, ok := posVal.(ast.Position); ok {
						lastErrorLine = pos.Line
						lastErrorCol = pos.Col
					}
				}
				return nil, nil, fmt.Errorf("%s", FormatTypeError(filename, string(bytes), te))
			}
			return nil, nil, fmt.Errorf("Type Error in '%s': %w", name, err)
		}
		sNext2, err := sNext.Unify(selfTy, tB)
		if err != nil {
			var te *typecheck.TypeError
			if errors.As(err, &te) {
				posVal, found := ast.NodePositions.Load(ast.GetNodeKey(te.Node))
				if found {
					if pos, ok := posVal.(ast.Position); ok {
						lastErrorLine = pos.Line
						lastErrorCol = pos.Col
					}
				}
				return nil, nil, fmt.Errorf("%s", FormatTypeError(filename, string(bytes), te))
			}
			return nil, nil, fmt.Errorf("Type Error in '%s': %w", name, err)
		}
		sCurr = sNext2

		finalTy := sCurr.Apply(selfTy)
		scheme := typecheck.Generalize(sCurr.ApplyEnv(accTypeEnv), finalTy)
		accTypeEnv = accTypeEnv.Extend(name, scheme)
		resolved := eval.Resolve(desugaredLambda)
		// store the global as a memoized cell (CAF): constants evaluate
		// once per session and self-recursive globals blackhole cleanly
		rootEnv := accEnv.Root
		if rootEnv == nil {
			rootEnv = accEnv
		}
		global := ast.ThunkNode{Cell: &ast.ThunkCell{State: ast.Unevaluated, Expr: resolved, Env: rootEnv}}
		if posVal, found := ast.NodePositions.Load(ast.GetNodeKey(desugaredLambda)); found {
			ast.NodePositions.Store(ast.GetNodeKey(resolved), posVal)
			ast.NodePositions.Store(ast.GetNodeKey(global), posVal)
		}
		accEnv = accEnv.ExtendGlobal(name, global)
	}

	return accEnv, accTypeEnv, nil
}

func RunREPLDirect(env *ast.Env, typeEnv *typecheck.TypeEnv, scriptFile string) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)
	doneChan := make(chan struct{})
	go func() {
		for {
			select {
			case <-sigChan:
				eval.SetInterrupted(true)
			case <-doneChan:
				return
			}
		}
	}()
	defer func() {
		close(doneChan)
		signal.Stop(sigChan)
	}()

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
		if lineTrimmed == "/?" || lineTrimmed == "/h" || lineTrimmed == "help" {
			fmt.Println("Available commands:")
			fmt.Println("  /q, exit, quit    Exit the REPL")
			fmt.Println("  /e [file.m]       Edit script file (default ~/.script.m)")
			fmt.Println("  /m                Open language manual using 'more'")
			fmt.Println("  !COMMAND          Execute COMMAND in the Unix shell")
			fmt.Println("  ?FUNCTION         Show the first line of a function's definition")
			fmt.Println("  ??FUNCTION        Open the file defining FUNCTION in editor at the definition line")
			fmt.Println("  /? or /h          Show this help menu")
			continue
		}
		if strings.HasPrefix(lineTrimmed, "??") {
			funcName := strings.TrimSpace(strings.TrimPrefix(lineTrimmed, "??"))
			handleShowDefinition(env, funcName, true)
			continue
		} else if strings.HasPrefix(lineTrimmed, "?") {
			funcName := strings.TrimSpace(strings.TrimPrefix(lineTrimmed, "?"))
			handleShowDefinition(env, funcName, false)
			continue
		}
		if strings.HasPrefix(lineTrimmed, "!") {
			handleShellCommand(lineTrimmed)
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
		if lineTrimmed == "/m" {
			handleOpenManual()
			continue
		}
		if strings.HasPrefix(lineTrimmed, "/e") {
			targetFile := scriptFile
			parts := strings.Fields(lineTrimmed)
			if len(parts) > 1 {
				target := parts[1]
				if !strings.HasSuffix(target, ".m") {
					fmt.Println("Error: Filename must have a .m extension indicating a Miracula file.")
					continue
				}
				targetFile = target
				scriptFile = targetFile
			}

			editor := "./mica"
			if _, err := os.Stat(editor); err != nil {
				editor = "vi"
			}
			var cmd *exec.Cmd
			resolvedTargetFile := ExpandHome(targetFile)
			if lastErrorLine > 0 {
				fmt.Printf("Opening %s %s at line %d, col %d ...\n", editor, targetFile, lastErrorLine, lastErrorCol)
				if editor == "./mica" {
					cmd = exec.Command(editor, resolvedTargetFile, strconv.Itoa(lastErrorLine), strconv.Itoa(lastErrorCol))
				} else {
					cmd = exec.Command(editor, fmt.Sprintf("+call cursor(%d,%d)", lastErrorLine, lastErrorCol), resolvedTargetFile)
				}
				// Reset error tracking after opening
				lastErrorLine = 0
				lastErrorCol = 0
			} else {
				fmt.Printf("Opening %s %s ...\n", editor, targetFile)
				cmd = exec.Command(editor, resolvedTargetFile)
			}
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			_ = cmd.Run()
			fmt.Printf("Reloading environment profiles from %s...\n", targetFile)
			envWithStd, typeEnvWithStd, _ := LoadScriptFile("stdenv.m", ast.NewEnv(), typecheck.DefaultTypeEnv())

			if envWithHome, typeEnvWithHome, err := LoadScriptFile("~/.script.m", envWithStd, typeEnvWithStd); err == nil {
				envWithStd = envWithHome
				typeEnvWithStd = typeEnvWithHome
			}

			var reloadedEnv *ast.Env
			var reloadedTypeEnv *typecheck.TypeEnv
			var err error

			if ExpandHome(targetFile) != ExpandHome("~/.script.m") {
				reloadedEnv, reloadedTypeEnv, err = LoadScriptFile(targetFile, envWithStd, typeEnvWithStd)
			} else {
				reloadedEnv = envWithStd
				reloadedTypeEnv = typeEnvWithStd
			}

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

		eval.SetInterrupted(false)
		func() {
			defer func() {
				if r := recover(); r != nil {
					if _, ok := r.(eval.InterruptedException); ok {
						fmt.Println("\nEvaluation Interrupted.")
					} else {
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
				p := parser.NewParser(seg).WithFilename("~/.script.m")
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
					if len(eqList) > 0 {
						bodyKey := ast.GetNodeKey(eqList[0].Body)
						if posVal, found := ast.NodePositions.Load(bodyKey); found {
							ast.NodePositions.Store(ast.GetNodeKey(finalLambda), posVal)
						}
					}

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
					resolved := eval.Resolve(finalLambda)
					rootEnv := accEnv.Root
					if rootEnv == nil {
						rootEnv = accEnv
					}
					global := ast.ThunkNode{Cell: &ast.ThunkCell{State: ast.Unevaluated, Expr: resolved, Env: rootEnv}}
					if posVal, found := ast.NodePositions.Load(ast.GetNodeKey(finalLambda)); found {
						ast.NodePositions.Store(ast.GetNodeKey(resolved), posVal)
						ast.NodePositions.Store(ast.GetNodeKey(global), posVal)
					}
					accEnv = accEnv.ExtendGlobal(name, global)
				}

				for _, name := range order {
					fmt.Printf("Defined variable: %s\n", name)
				}
				env = accEnv
				typeEnv = accTypeEnv

				lineOffset := countLinesInFile(ExpandHome("~/.script.m"))

				// Keep REPL definitions in ~/.script.m
				f, err := os.OpenFile(ExpandHome("~/.script.m"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err == nil {
					toWrite := fullInput
					if !strings.HasSuffix(toWrite, "\n") {
						toWrite += "\n"
					}
					_, _ = f.WriteString(toWrite)
					_ = f.Close()
				}

				// Update positions to reference ~/.script.m with the correct line offset
				for _, name := range order {
					val, ok := env.Lookup(name)
					if ok {
						key := ast.GetNodeKey(val)
						if posVal, found := ast.NodePositions.Load(key); found {
							if pos, ok := posVal.(ast.Position); ok {
								ast.NodePositions.Store(key, ast.Position{
									Filename: "~/.script.m",
									Line:     pos.Line + lineOffset,
									Col:      pos.Col,
								})
							}
						} else {
							ast.NodePositions.Store(key, ast.Position{
								Filename: "~/.script.m",
								Line:     lineOffset + 1,
								Col:      1,
							})
						}
					}
				}
			} else {
				tc := typecheck.NewTypeChecker()
				_, _, err := tc.Infer(typeEnv, evalStmt.Expr, nil)
				if err != nil {
					panic(err)
				}

				startTime := time.Now()
				result := eval.Whnf(env, eval.Resolve(evalStmt.Expr))
				sVal, isStr := eval.IsString(env, result)
				var output string
				if isStr {
					output = sVal
				} else {
					output = eval.PrintNode(env, result)
				}
				duration := time.Since(startTime).Milliseconds()

				if isStr {
					fmt.Printf("Result:\n%s", output)
					if len(output) > 0 && output[len(output)-1] == '\n' {
						// no extra newline
					} else {
						fmt.Println()
					}
				} else {
					fmt.Printf("Result: %s\n", output)
				}
				fmt.Printf("Evaluation time: %d ms\n", duration)
			}
		}()
	}
}

// countLinesInFile reads a file and counts the number of lines in it.
func countLinesInFile(filepath string) int {
	bytes, err := os.ReadFile(filepath)
	if err != nil {
		return 0
	}
	return countLines(string(bytes))
}

// countLines counts the number of lines in a given string content, treating
// non-empty lines without a trailing newline as a line as well.
func countLines(content string) int {
	if len(content) == 0 {
		return 0
	}
	n := strings.Count(content, "\n")
	if !strings.HasSuffix(content, "\n") {
		n++
	}
	return n
}

// handleShowDefinition looks up a function in the environment and either prints
// its definition's first line or opens the corresponding file in an editor at the definition line.
func handleShowDefinition(env *ast.Env, funcName string, openInEditor bool) {
	if funcName == "" {
		fmt.Println("Usage: ?FUNCTION to show definition, ??FUNCTION to open in editor")
		return
	}

	val, ok := env.Lookup(funcName)
	if !ok {
		switch funcName {
		case "hd", "tl", "show", "read", "lines", "numval", "length", "reverse":
			fmt.Printf("Built-in function: %s\n", funcName)
			return
		}
		fmt.Printf("Function '%s' not found.\n", funcName)
		return
	}

	key := ast.GetNodeKey(val)
	posVal, found := ast.NodePositions.Load(key)
	if !found {
		fmt.Printf("Position info for '%s' not found.\n", funcName)
		return
	}

	pos, ok := posVal.(ast.Position)
	if !ok {
		fmt.Printf("Invalid position info for '%s'.\n", funcName)
		return
	}

	if pos.Filename == "" {
		fmt.Printf("Function '%s' is defined at line %d, but the source file is unknown.\n", funcName, pos.Line)
		return
	}

	resolvedFile := ExpandHome(pos.Filename)

	if !openInEditor {
		bytes, err := os.ReadFile(resolvedFile)
		if err != nil {
			fmt.Printf("Error reading source file %s: %v\n", pos.Filename, err)
			return
		}
		lines := strings.Split(string(bytes), "\n")
		if pos.Line < 1 || pos.Line > len(lines) {
			fmt.Printf("Function '%s' is defined at %s:%d, but file has only %d lines.\n", funcName, pos.Filename, pos.Line, len(lines))
			return
		}
		lineContent := strings.TrimRightFunc(lines[pos.Line-1], unicode.IsSpace)
		fmt.Printf("%s:%d: %s\n", pos.Filename, pos.Line, lineContent)
		return
	}

	editor := "./mica"
	if _, err := os.Stat(editor); err != nil {
		editor = "vi"
	}
	var cmd *exec.Cmd
	fmt.Printf("Opening %s %s at line %d ...\n", editor, pos.Filename, pos.Line)
	if editor == "./mica" {
		cmd = exec.Command(editor, resolvedFile, strconv.Itoa(pos.Line), "1")
	} else {
		cmd = exec.Command(editor, fmt.Sprintf("+%d", pos.Line), resolvedFile)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

// getManualContent reads the project's Miracula manual file miracula-man.md.
func getManualContent() string {
	manBytes, err := os.ReadFile("miracula-man.md")
	if err != nil {
		manBytes, err = os.ReadFile("/Users/pkreyenhop/src/miracula-go/miracula-man.md")
	}

	if err != nil {
		return "miracula-man.md not found.\n"
	}
	return string(manBytes)
}

// handleOpenManual displays the unified manual content using the Unix 'more' command.
func handleOpenManual() {
	content := getManualContent()

	tempFile := filepath.Join(os.TempDir(), "miracula_manual.txt")
	err := os.WriteFile(tempFile, []byte(content), 0644)
	if err != nil {
		fmt.Printf("Error creating manual file: %v\n", err)
		return
	}
	defer os.Remove(tempFile)

	cmd := exec.Command("more", tempFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

// handleShellCommand executes a command line directly in the Unix shell (sh -c).
func handleShellCommand(line string) {
	cmdText := strings.TrimSpace(strings.TrimPrefix(line, "!"))
	if cmdText == "" {
		fmt.Println("Usage: !command to execute a shell command")
		return
	}

	cmd := exec.Command("sh", "-c", cmdText)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

// EvaluateAndExit evaluates a FILE or a COMMAND and exits.
// OnExit, when set, runs before EvaluateAndExit terminates the process —
// main uses it to flush the CPU profile, which os.Exit would otherwise skip.
var OnExit func()

func exitWith(code int) {
	if OnExit != nil {
		OnExit()
	}
	os.Exit(code)
}

func EvaluateAndExit(env *ast.Env, typeEnv *typecheck.TypeEnv, parameter string, showResult, showTiming bool) {
	defer func() {
		if r := recover(); r != nil {
			if rtErr, ok := r.(ast.RuntimeError); ok {
				if strings.HasPrefix(rtErr.Msg, "Type Error:") {
					fmt.Println(rtErr.Msg)
				} else {
					fmt.Printf("Runtime Error: %s\n", rtErr.Msg)
				}
			} else if bhErr, ok := r.(ast.BlackholeError); ok {
				fmt.Printf("Runtime Error: %s\n", bhErr.Msg)
			} else if pe, ok := r.(parser.ParseError); ok {
				fmt.Println(FormatParseError("<stdin>", parameter, pe))
			} else {
				fmt.Printf("Error: %v\n", r)
			}
			exitWith(1)
		}
	}()

	info, err := os.Stat(parameter)
	isFile := err == nil && !info.IsDir()

	if isFile {
		nextEnv, nextTypeEnv, err := LoadScriptFile(parameter, env, typeEnv)
		if err != nil {
			fmt.Printf("Error loading %s: %v\n", parameter, err)
			exitWith(1)
		}
		env = nextEnv
		typeEnv = nextTypeEnv

		mainVal, ok := env.Lookup("main")
		if !ok {
			fmt.Printf("Error: 'main' is not defined in %s\n", parameter)
			exitWith(1)
		}

		startTime := time.Now()
		result := eval.Whnf(env, mainVal)
		sVal, isStr := eval.IsString(env, result)
		var output string
		if isStr {
			output = sVal
		} else {
			output = eval.PrintNode(env, result)
		}
		duration := time.Since(startTime).Milliseconds()

		if showResult {
			if isStr {
				fmt.Printf("Result:\n%s", output)
				if len(output) > 0 && output[len(output)-1] == '\n' {
					// no extra newline
				} else {
					fmt.Println()
				}
			} else {
				fmt.Printf("Result: %s\n", output)
			}
		}
		if showTiming {
			fmt.Printf("Evaluation time: %d ms\n", duration)
		}
		exitWith(0)
	} else {
		tokens := lexer.TokenizeWithPos(parameter, 1)
		var filtered []lexer.Token
		for _, t := range tokens {
			if t.Type != lexer.TOK_EOF {
				filtered = append(filtered, t)
			}
		}

		var layoutLines []lexer.LayoutLine
		layoutLines = append(layoutLines, lexer.LayoutLine{Indent: 0, Toks: lexer.WrapWhereOnLine(filtered)})
		fileTokens := lexer.ApplyLayout(layoutLines)
		segments := lexer.SplitTokens(fileTokens)
		if len(segments) == 0 {
			exitWith(0)
		}

		p := parser.NewParser(segments[0]).WithFilename("<stdin>")
		stmt := p.Parse()

		switch s := stmt.(type) {
		case parser.REPLEvalStmt:
			tc := typecheck.NewTypeChecker()
			_, _, err := tc.Infer(typeEnv, s.Expr, nil)
			if err != nil {
				fmt.Printf("Type Error: %v\n", err)
				exitWith(1)
			}

			startTime := time.Now()
			result := eval.Whnf(env, eval.Resolve(s.Expr))
			sVal, isStr := eval.IsString(env, result)
			var output string
			if isStr {
				output = sVal
			} else {
				output = eval.PrintNode(env, result)
			}
			duration := time.Since(startTime).Milliseconds()

			if showResult {
				if isStr {
					fmt.Printf("Result:\n%s", output)
					if len(output) > 0 && output[len(output)-1] == '\n' {
						// no extra newline
					} else {
						fmt.Println()
					}
				} else {
					fmt.Printf("Result: %s\n", output)
				}
			}
			if showTiming {
				fmt.Printf("Evaluation time: %d ms\n", duration)
			}
			exitWith(0)
		default:
			fmt.Printf("Error: not an evaluation expression\n")
			exitWith(1)
		}
	}
}
