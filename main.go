package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"
	"unsafe"
)

// ==========================================================================
// 1. AST & VALUE NODES DEFINITION
// ==========================================================================

type Node interface {
	isNode()
}

type IntNode struct{ Val int }
type CharNode struct{ Val rune }
type NilNode struct{}
type ConsNode struct{ Head, Tail Node }
type TupleNode struct{ Elems []Node }
type VarNode struct{ Name string }
type LamNode struct {
	Var  string
	Body Node
}
type ClosureNode struct {
	Var  string
	Body Node
	Env  *Env
}
type LetNode struct {
	Bindings []Binding
	Body     Node
}
type ProjNode struct {
	Index int
	Tuple Node
}
type AppNode struct {
	Left  Node
	Right Node
}
type MatchErrorNode struct{}

type ThunkState int

const (
	Unevaluated ThunkState = iota
	Evaluating
	Evaluated
)

type ThunkCell struct {
	State ThunkState
	Expr  Node
	Env   *Env
	Val   Node
}

type ThunkNode struct {
	Cell *ThunkCell
}

type IfZeroNode struct{ Cond, Then, Else Node }
type IfNilNode struct{ Cond, Then, Else Node }
type IfNode struct{ Cond, Then, Else Node }
type AppendNode struct{ Left, Right Node }
type DiffNode struct{ Left, Right Node }
type RangeNode struct{ Start, End Node }

type ZFNode struct {
	Body Node
	Quals []Qualifier
}

type ZFGeneratorNode struct {
	Pat   Pat
	Rest  []Qualifier
	Src   Node
	Body  Node
	ZFEnv *Env
}

type AddNode struct{ Left, Right Node }
type SubNode struct{ Left, Right Node }
type MulNode struct{ Left, Right Node }
type DivNode struct{ Left, Right Node }
type ModNode struct{ Left, Right Node }
type EqNode struct{ Left, Right Node }
type NeNode struct{ Left, Right Node }
type LtNode struct{ Left, Right Node }
type GtNode struct{ Left, Right Node }
type LeNode struct{ Left, Right Node }
type GeNode struct{ Left, Right Node }

func (IntNode) isNode()         {}
func (CharNode) isNode()        {}
func (NilNode) isNode()         {}
func (ConsNode) isNode()        {}
func (TupleNode) isNode()       {}
func (VarNode) isNode()         {}
func (LamNode) isNode()         {}
func (ClosureNode) isNode()     {}
func (LetNode) isNode()         {}
func (ProjNode) isNode()        {}
func (AppNode) isNode()         {}
func (MatchErrorNode) isNode()  {}
func (ThunkNode) isNode()       {}
func (IfZeroNode) isNode()      {}
func (IfNilNode) isNode()       {}
func (IfNode) isNode()          {}
func (AppendNode) isNode()      {}
func (DiffNode) isNode()        {}
func (ZFNode) isNode()          {}
func (ZFGeneratorNode) isNode() {}
func (RangeNode) isNode()       {}
func (AddNode) isNode()         {}
func (SubNode) isNode()         {}
func (MulNode) isNode()         {}
func (DivNode) isNode()         {}
func (ModNode) isNode()         {}
func (EqNode) isNode()          {}
func (NeNode) isNode()          {}
func (LtNode) isNode()          {}
func (GtNode) isNode()          {}
func (LeNode) isNode()          {}
func (GeNode) isNode()          {}

type Binding struct {
	Name string
	Expr Node
}

// ==========================================================================
// 2. PATTERNS & QUALIFIERS
// ==========================================================================

type Pat interface {
	isPat()
}

type PatInt struct{ Val int }
type PatChar struct{ Val rune }
type PatVar struct{ Name string }
type PatNil struct{}
type PatCons struct{ Head, Tail Pat }
type PatTuple struct{ Elems []Pat }

func (PatInt) isPat()   {}
func (PatChar) isPat()  {}
func (PatVar) isPat()   {}
func (PatNil) isPat()   {}
func (PatCons) isPat()  {}
func (PatTuple) isPat() {}

type Qualifier interface {
	isQualifier()
}

type GeneratorQual struct {
	Pat Pat
	Src Node
}

type FilterQual struct {
	Cond Node
}

func (GeneratorQual) isQualifier() {}
func (FilterQual) isQualifier()    {}

// ==========================================================================
// 3. ENVIRONMENT DEFINITION
// ==========================================================================

type Env struct {
	Parent *Env
	Name   string
	Val    Node
}

func (e *Env) Lookup(x string) (Node, bool) {
	for curr := e; curr != nil; curr = curr.Parent {
		if curr.Name == x {
			return curr.Val, true
		}
	}
	return nil, false
}

func (e *Env) Extend(x string, val Node) *Env {
	return &Env{Parent: e, Name: x, Val: val}
}

// ==========================================================================
// 4. EXCEPTIONS / ERRORS
// ==========================================================================

type RuntimeError struct {
	Msg string
}

func (e RuntimeError) Error() string { return e.Msg }

type BlackholeError struct {
	Msg string
}

func (e BlackholeError) Error() string { return e.Msg }

// ==========================================================================
// 5. LEXER IMPLEMENTATION
// ==========================================================================

type TokenType int

const (
	TOK_LAMBDA TokenType = iota
	TOK_DOT
	TOK_DOTDOT
	TOK_ARROW
	TOK_ASSIGN
	TOK_LPAREN
	TOK_RPAREN
	TOK_LBRACK
	TOK_RBRACK
	TOK_COMMA
	TOK_COLON
	TOK_SUB
	TOK_ADD
	TOK_MUL
	TOK_IFZERO
	TOK_THEN
	TOK_ELSE
	TOK_INT
	TOK_VAR
	TOK_EOF
	TOK_PIPE
	TOK_LARROW
	TOK_SEMICOLON
	TOK_EQ
	TOK_NE
	TOK_LT
	TOK_GT
	TOK_LE
	TOK_GE
	TOK_MOD
	TOK_IF
	TOK_CHAR
	TOK_STRING
	TOK_PP
	TOK_WHERE
	TOK_LBRACE
	TOK_RBRACE
	TOK_HASH
	TOK_DIV
	TOK_AND
	TOK_OR
	TOK_DIFF
)

type Token struct {
	Type TokenType
	Int  int
	Str  string
	Char rune
}

func tokenize(str string) []Token {
	runes := []rune(str)
	size := len(runes)
	var acc []Token
	i := 0
	for i < size {
		c := runes[i]
		if unicode.IsSpace(c) {
			i++
			continue
		}
		if c == '\\' {
			if i+1 < size && runes[i+1] == '/' {
				acc = append(acc, Token{Type: TOK_OR})
				i += 2
			} else {
				acc = append(acc, Token{Type: TOK_LAMBDA})
				i++
			}
			continue
		}
		if c == '.' {
			if i+1 < size && runes[i+1] == '.' {
				acc = append(acc, Token{Type: TOK_DOTDOT})
				i += 2
			} else {
				acc = append(acc, Token{Type: TOK_DOT})
				i++
			}
			continue
		}
		if c == '(' {
			acc = append(acc, Token{Type: TOK_LPAREN})
			i++
			continue
		}
		if c == ')' {
			acc = append(acc, Token{Type: TOK_RPAREN})
			i++
			continue
		}
		if c == '[' {
			acc = append(acc, Token{Type: TOK_LBRACK})
			i++
			continue
		}
		if c == ']' {
			acc = append(acc, Token{Type: TOK_RBRACK})
			i++
			continue
		}
		if c == ',' {
			acc = append(acc, Token{Type: TOK_COMMA})
			i++
			continue
		}
		if c == ';' {
			acc = append(acc, Token{Type: TOK_SEMICOLON})
			i++
			continue
		}
		if c == '|' {
			if i+1 < size && runes[i+1] == '|' {
				// Comment! Ignore the rest of the line
				break
			} else {
				acc = append(acc, Token{Type: TOK_PIPE})
				i++
			}
			continue
		}
		if c == '<' {
			if i+1 < size && runes[i+1] == '-' {
				acc = append(acc, Token{Type: TOK_LARROW})
				i += 2
			} else if i+1 < size && runes[i+1] == '=' {
				acc = append(acc, Token{Type: TOK_LE})
				i += 2
			} else {
				acc = append(acc, Token{Type: TOK_LT})
				i++
			}
			continue
		}
		if c == '>' {
			if i+1 < size && runes[i+1] == '=' {
				acc = append(acc, Token{Type: TOK_GE})
				i += 2
			} else {
				acc = append(acc, Token{Type: TOK_GT})
				i++
			}
			continue
		}
		if c == '=' {
			if i+1 < size && runes[i+1] == '=' {
				acc = append(acc, Token{Type: TOK_EQ})
				i += 2
			} else {
				acc = append(acc, Token{Type: TOK_ASSIGN})
				i++
			}
			continue
		}
		if c == '!' {
			if i+1 < size && runes[i+1] == '=' {
				acc = append(acc, Token{Type: TOK_NE})
				i += 2
			} else {
				i++
			}
			continue
		}
		if c == '~' {
			if i+1 < size && runes[i+1] == '=' {
				acc = append(acc, Token{Type: TOK_NE})
				i += 2
			} else {
				i++
			}
			continue
		}
		if c == '/' {
			acc = append(acc, Token{Type: TOK_DIV})
			i++
			continue
		}
		if c == '&' {
			acc = append(acc, Token{Type: TOK_AND})
			i++
			continue
		}
		if c == '*' {
			acc = append(acc, Token{Type: TOK_MUL})
			i++
			continue
		}
		if c == ':' {
			acc = append(acc, Token{Type: TOK_COLON})
			i++
			continue
		}
		if c == '#' {
			acc = append(acc, Token{Type: TOK_HASH})
			i++
			continue
		}
		if c == '+' {
			if i+1 < size && runes[i+1] == '+' {
				acc = append(acc, Token{Type: TOK_PP})
				i += 2
			} else {
				acc = append(acc, Token{Type: TOK_ADD})
				i++
			}
			continue
		}
		if c == '-' {
			if i+1 < size && runes[i+1] == '>' {
				acc = append(acc, Token{Type: TOK_ARROW})
				i += 2
			} else if i+1 < size && runes[i+1] == '-' {
				acc = append(acc, Token{Type: TOK_DIFF})
				i += 2
			} else {
				acc = append(acc, Token{Type: TOK_SUB})
				i++
			}
			continue
		}
		if c == '\'' {
			if i+2 < size && runes[i+1] != '\\' && runes[i+2] == '\'' {
				acc = append(acc, Token{Type: TOK_CHAR, Char: runes[i+1]})
				i += 3
			} else if i+3 < size && runes[i+1] == '\\' && runes[i+3] == '\'' {
				esc := runes[i+2]
				var ch rune
				switch esc {
				case 'n':
					ch = '\n'
				case 't':
					ch = '\t'
				case '\'':
					ch = '\''
				case '\\':
					ch = '\\'
				default:
					ch = esc
				}
				acc = append(acc, Token{Type: TOK_CHAR, Char: ch})
				i += 4
			} else {
				i++
			}
			continue
		}
		if c == '"' {
			j := i + 1
			var sb strings.Builder
			for j < size {
				if runes[j] == '"' {
					j++
					break
				}
				if runes[j] == '\\' && j+1 < size {
					esc := runes[j+1]
					var ch rune
					switch esc {
					case 'n':
						ch = '\n'
					case 't':
						ch = '\t'
					case '"':
						ch = '"'
					case '\\':
						ch = '\\'
					default:
						ch = esc
					}
					sb.WriteRune(ch)
					j += 2
				} else {
					sb.WriteRune(runes[j])
					j++
				}
			}
			acc = append(acc, Token{Type: TOK_STRING, Str: sb.String()})
			i = j
			continue
		}
		if unicode.IsDigit(c) {
			j := i + 1
			for j < size && unicode.IsDigit(runes[j]) {
				j++
			}
			val, _ := strconv.Atoi(string(runes[i:j]))
			acc = append(acc, Token{Type: TOK_INT, Int: val})
			i = j
			continue
		}
		if unicode.IsLetter(c) || c == '_' {
			j := i + 1
			for j < size && (unicode.IsLetter(runes[j]) || unicode.IsDigit(runes[j]) || runes[j] == '_') {
				j++
			}
			s := string(runes[i:j])
			var tokType TokenType
			switch s {
			case "ifzero":
				tokType = TOK_IFZERO
			case "if":
				tokType = TOK_IF
			case "then":
				tokType = TOK_THEN
			case "else":
				tokType = TOK_ELSE
			case "mod":
				tokType = TOK_MOD
			case "where":
				tokType = TOK_WHERE
			default:
				tokType = TOK_VAR
			}
			acc = append(acc, Token{Type: tokType, Str: s})
			i = j
			continue
		}
		i++
	}
	acc = append(acc, Token{Type: TOK_EOF})
	return acc
}

func wrapWhereOnLine(toks []Token) []Token {
	var res []Token
	for i := 0; i < len(toks); i++ {
		if toks[i].Type == TOK_WHERE {
			res = append(res, toks[i])
			if i+1 < len(toks) {
				res = append(res, Token{Type: TOK_LBRACE})
				res = append(res, wrapWhereOnLine(toks[i+1:])...)
				res = append(res, Token{Type: TOK_RBRACE})
				break
			}
		} else {
			res = append(res, toks[i])
		}
	}
	return res
}

type layoutLine struct {
	Indent int
	Toks   []Token
}

func tokenDepthDelta(toks []Token) int {
	delta := 0
	for _, t := range toks {
		switch t.Type {
		case TOK_LPAREN, TOK_LBRACK:
			delta++
		case TOK_RPAREN, TOK_RBRACK:
			delta--
		}
	}
	return delta
}

func hasWhere(toks []Token) bool {
	for _, t := range toks {
		if t.Type == TOK_WHERE {
			return true
		}
	}
	return false
}

func applyLayout(lines []layoutLine) []Token {
	stack := []int{0}
	var acc []Token
	expectLayout := false
	depth := 0

	for _, line := range lines {
		indent := line.Indent
		lineToks := line.Toks

		justPushed := false
		if expectLayout && depth == 0 {
			parentLayout := stack[len(stack)-1]
			if indent > parentLayout {
				stack = append(stack, indent)
				acc = append(acc, Token{Type: TOK_LBRACE})
				expectLayout = false
				justPushed = true
			} else {
				expectLayout = false
			}
		}

		if depth == 0 {
			for len(stack) > 1 && indent < stack[len(stack)-1] {
				stack = stack[:len(stack)-1]
				acc = append(acc, Token{Type: TOK_RBRACE})
			}
		}

		currentLayout := stack[len(stack)-1]
		if depth == 0 && indent == currentLayout && len(acc) > 0 && !justPushed {
			acc = append(acc, Token{Type: TOK_SEMICOLON})
		}

		if depth == 0 {
			expectLayout = hasWhere(lineToks)
		} else {
			expectLayout = false
		}

		acc = append(acc, lineToks...)
		delta := tokenDepthDelta(lineToks)
		depth += delta
		if depth < 0 {
			depth = 0
		}
	}

	for len(stack) > 1 {
		stack = stack[:len(stack)-1]
		acc = append(acc, Token{Type: TOK_RBRACE})
	}
	acc = append(acc, Token{Type: TOK_EOF})
	return acc
}

func splitTokens(tokens []Token) [][]Token {
	var segments [][]Token
	var current []Token
	depth := 0

	for _, t := range tokens {
		if t.Type == TOK_EOF {
			continue
		}
		newDepth := depth
		switch t.Type {
		case TOK_LBRACE, TOK_LPAREN, TOK_LBRACK:
			newDepth++
		case TOK_RBRACE, TOK_RPAREN, TOK_RBRACK:
			newDepth--
		}

		if t.Type == TOK_SEMICOLON && depth == 0 {
			segment := append([]Token(nil), current...)
			segment = append(segment, Token{Type: TOK_EOF})
			segments = append(segments, segment)
			current = nil
		} else {
			current = append(current, t)
		}
		depth = newDepth
	}

	if len(current) > 0 {
		segment := append([]Token(nil), current...)
		segment = append(segment, Token{Type: TOK_EOF})
		segments = append(segments, segment)
	}
	return segments
}

// ==========================================================================
// 6. PARSER IMPLEMENTATION
// ==========================================================================

type RawBinding struct {
	FName string
	Pats  []Pat
	Body  Node
}

type Stmt interface {
	isStmt()
}

type ScriptBindStmt struct {
	Binding RawBinding
}

type REPLEvalStmt struct {
	Expr Node
}

func (ScriptBindStmt) isStmt() {}
func (REPLEvalStmt) isStmt()   {}

type Parser struct {
	tokens []Token
	pos    int
}

func NewParser(tokens []Token) *Parser {
	return &Parser{tokens: tokens, pos: 0}
}

func (p *Parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TOK_EOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) peek2() Token {
	if p.pos+1 >= len(p.tokens) {
		return Token{Type: TOK_EOF}
	}
	return p.tokens[p.pos+1]
}

func (p *Parser) peek3() Token {
	if p.pos+2 >= len(p.tokens) {
		return Token{Type: TOK_EOF}
	}
	return p.tokens[p.pos+2]
}

func (p *Parser) consume() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

func isAssignment(tokens []Token) bool {
	depth := 0
	for _, t := range tokens {
		if t.Type == TOK_SEMICOLON && depth == 0 {
			return false
		}
		if t.Type == TOK_RBRACE && depth == 0 {
			return false
		}
		if t.Type == TOK_ASSIGN && depth == 0 {
			return true
		}
		switch t.Type {
		case TOK_LBRACE, TOK_LPAREN, TOK_LBRACK:
			depth++
		case TOK_RBRACE, TOK_RPAREN, TOK_RBRACK:
			depth--
		}
	}
	return false
}

func (p *Parser) parse() Stmt {
	if isAssignment(p.tokens[p.pos:]) {
		tok := p.peek()
		if tok.Type != TOK_VAR {
			panic(fmt.Errorf("left hand side of binding must start with an identifier"))
		}
		p.consume()
		var pats []Pat
		for p.peek().Type != TOK_ASSIGN {
			pats = append(pats, p.parsePattern())
		}
		p.consume() // '='
		exprBody := p.parseExpr()
		return ScriptBindStmt{Binding: RawBinding{FName: tok.Str, Pats: pats, Body: exprBody}}
	} else {
		e := p.parseExpr()
		if p.peek().Type != TOK_EOF {
			panic(fmt.Errorf("trailing tokens left unparsed: %v", p.peek()))
		}
		return REPLEvalStmt{Expr: e}
	}
}

func (p *Parser) parseExpr() Node {
	var e Node
	tok := p.peek()
	switch tok.Type {
	case TOK_LAMBDA:
		p.consume()
		v := p.peek()
		if v.Type != TOK_VAR {
			panic(fmt.Errorf("expected variable after lambda '\\'"))
		}
		p.consume()
		if p.peek().Type != TOK_DOT {
			panic(fmt.Errorf("expected '.' after lambda variable"))
		}
		p.consume()
		e = LamNode{Var: v.Str, Body: p.parseExpr()}
	case TOK_IFZERO:
		p.consume()
		cond := p.parseExpr()
		if p.peek().Type != TOK_THEN {
			panic(fmt.Errorf("expected 'then'"))
		}
		p.consume()
		tBranch := p.parseExpr()
		if p.peek().Type != TOK_ELSE {
			panic(fmt.Errorf("expected 'else'"))
		}
		p.consume()
		fBranch := p.parseExpr()
		e = IfZeroNode{Cond: cond, Then: tBranch, Else: fBranch}
	case TOK_IF:
		p.consume()
		cond := p.parseExpr()
		if p.peek().Type != TOK_THEN {
			panic(fmt.Errorf("expected 'then'"))
		}
		p.consume()
		tBranch := p.parseExpr()
		if p.peek().Type != TOK_ELSE {
			panic(fmt.Errorf("expected 'else'"))
		}
		p.consume()
		fBranch := p.parseExpr()
		e = IfNode{Cond: cond, Then: tBranch, Else: fBranch}
	default:
		e = p.parseOr()
	}

	if p.peek().Type == TOK_WHERE {
		p.consume()
		if p.peek().Type != TOK_LBRACE {
			panic(fmt.Errorf("expected '{' after 'where'"))
		}
		p.consume()

		var parseBindings func() []RawBinding
		parseBindings = func() []RawBinding {
			if p.peek().Type == TOK_RBRACE {
				p.consume()
				return nil
			}
			if isAssignment(p.tokens[p.pos:]) {
				nameTok := p.peek()
				if nameTok.Type != TOK_VAR {
					panic(fmt.Errorf("left hand side of local binding must start with an identifier"))
				}
				p.consume()
				var pats []Pat
				for p.peek().Type != TOK_ASSIGN {
					pats = append(pats, p.parsePattern())
				}
				p.consume() // '='
				exprBody := p.parseExpr()
				b := RawBinding{FName: nameTok.Str, Pats: pats, Body: exprBody}

				var rest []RawBinding
				if p.peek().Type == TOK_SEMICOLON {
					p.consume()
					rest = parseBindings()
				} else if p.peek().Type == TOK_RBRACE {
					p.consume()
				} else {
					panic(fmt.Errorf("expected ';' or '}' in where bindings"))
				}
				return append([]RawBinding{b}, rest...)
			} else {
				panic(fmt.Errorf("expected local binding in where clause"))
			}
		}

		bs := parseBindings()
		grouped := make(map[string][]RawBinding)
		var order []string
		for _, b := range bs {
			if _, ok := grouped[b.FName]; !ok {
				order = append(order, b.FName)
			}
			grouped[b.FName] = append(grouped[b.FName], b)
		}

		var desugared []Binding
		for _, name := range order {
			eqList := grouped[name]
			desugared = append(desugared, Binding{
				Name: name,
				Expr: desugarEquations(eqList),
			})
		}
		return LetNode{Bindings: desugared, Body: e}
	}
	return e
}

func (p *Parser) parseOr() Node {
	left := p.parseAnd()
	if p.peek().Type == TOK_OR {
		p.consume()
		return IfNode{Cond: left, Then: IntNode{Val: 1}, Else: p.parseOr()}
	}
	return left
}

func (p *Parser) parseAnd() Node {
	left := p.parseCons()
	if p.peek().Type == TOK_AND {
		p.consume()
		return IfNode{Cond: left, Then: p.parseAnd(), Else: IntNode{Val: 0}}
	}
	return left
}

func (p *Parser) parseCons() Node {
	left := p.parsePP()
	if p.peek().Type == TOK_COLON {
		p.consume()
		return ConsNode{Head: left, Tail: p.parseCons()}
	}
	return left
}

func (p *Parser) parsePP() Node {
	left := p.parseComp()
	tok := p.peek()
	if tok.Type == TOK_PP {
		p.consume()
		return AppendNode{Left: left, Right: p.parsePP()}
	} else if tok.Type == TOK_DIFF {
		p.consume()
		return DiffNode{Left: left, Right: p.parsePP()}
	}
	return left
}

func (p *Parser) parseComp() Node {
	left := p.parseAddSub()
	tok := p.peek()
	switch tok.Type {
	case TOK_EQ:
		p.consume()
		return EqNode{Left: left, Right: p.parseAddSub()}
	case TOK_NE:
		p.consume()
		return NeNode{Left: left, Right: p.parseAddSub()}
	case TOK_LT:
		p.consume()
		return LtNode{Left: left, Right: p.parseAddSub()}
	case TOK_GT:
		p.consume()
		return GtNode{Left: left, Right: p.parseAddSub()}
	case TOK_LE:
		p.consume()
		return LeNode{Left: left, Right: p.parseAddSub()}
	case TOK_GE:
		p.consume()
		return GeNode{Left: left, Right: p.parseAddSub()}
	}
	return left
}

func (p *Parser) parseAddSub() Node {
	left := p.parseMod()
	for {
		tok := p.peek()
		if tok.Type == TOK_ADD {
			p.consume()
			left = AddNode{Left: left, Right: p.parseMod()}
		} else if tok.Type == TOK_SUB {
			p.consume()
			left = SubNode{Left: left, Right: p.parseMod()}
		} else {
			break
		}
	}
	return left
}

func (p *Parser) parseMod() Node {
	left := p.parseCompose()
	for {
		tok := p.peek()
		if tok.Type == TOK_MOD {
			p.consume()
			left = ModNode{Left: left, Right: p.parseCompose()}
		} else if tok.Type == TOK_MUL {
			p.consume()
			left = MulNode{Left: left, Right: p.parseCompose()}
		} else if tok.Type == TOK_DIV {
			p.consume()
			left = DivNode{Left: left, Right: p.parseCompose()}
		} else {
			break
		}
	}
	return left
}

func (p *Parser) parseCompose() Node {
	left := p.parseApp()
	if p.peek().Type == TOK_DOT {
		p.consume()
		right := p.parseCompose()
		varName := newVarName("cx")
		return LamNode{
			Var:  varName,
			Body: AppNode{Left: left, Right: AppNode{Left: right, Right: VarNode{Name: varName}}},
		}
	}
	return left
}

func (p *Parser) parseApp() Node {
	left := p.parseAtom()
	for {
		tok := p.peek()
		if tok.Type == TOK_INT || tok.Type == TOK_CHAR || tok.Type == TOK_STRING ||
			tok.Type == TOK_VAR || tok.Type == TOK_LPAREN || tok.Type == TOK_LBRACK {
			left = AppNode{Left: left, Right: p.parseAtom()}
		} else {
			break
		}
	}
	return left
}

func (p *Parser) parseAtom() Node {
	tok := p.peek()
	switch tok.Type {
	case TOK_HASH:
		p.consume()
		return AppNode{Left: VarNode{Name: "length"}, Right: p.parseAtom()}
	case TOK_INT:
		p.consume()
		return IntNode{Val: tok.Int}
	case TOK_CHAR:
		p.consume()
		return CharNode{Val: tok.Char}
	case TOK_STRING:
		p.consume()
		return makeStringNode(tok.Str)
	case TOK_VAR:
		p.consume()
		return VarNode{Name: tok.Str}
	case TOK_SUB:
		p.consume()
		return SubNode{Left: IntNode{Val: 0}, Right: p.parseAtom()}
	case TOK_LBRACK:
		p.consume()
		return p.parseListElements()
	case TOK_LPAREN:
		if p.peek2().Type == TOK_COLON {
			if p.peek3().Type == TOK_RPAREN {
				p.consume() // '('
				p.consume() // ':'
				p.consume() // ')'
				return LamNode{
					Var: "x",
					Body: LamNode{
						Var:  "y",
						Body: ConsNode{Head: VarNode{Name: "x"}, Tail: VarNode{Name: "y"}},
					},
				}
			} else {
				p.consume() // '('
				p.consume() // ':'
				e := p.parseExpr()
				if p.peek().Type != TOK_RPAREN {
					panic(fmt.Errorf("expected ')'"))
				}
				p.consume()
				return LamNode{
					Var:  "x",
					Body: ConsNode{Head: VarNode{Name: "x"}, Tail: e},
				}
			}
		} else if p.peek2().Type == TOK_ADD {
			if p.peek3().Type == TOK_RPAREN {
				p.consume()
				p.consume()
				p.consume()
				return LamNode{
					Var: "x",
					Body: LamNode{
						Var:  "y",
						Body: AddNode{Left: VarNode{Name: "x"}, Right: VarNode{Name: "y"}},
					},
				}
			} else {
				p.consume()
				p.consume()
				e := p.parseExpr()
				if p.peek().Type != TOK_RPAREN {
					panic(fmt.Errorf("expected ')'"))
				}
				p.consume()
				return LamNode{
					Var:  "x",
					Body: AddNode{Left: VarNode{Name: "x"}, Right: e},
				}
			}
		} else if p.peek2().Type == TOK_SUB {
			if p.peek3().Type == TOK_RPAREN {
				p.consume()
				p.consume()
				p.consume()
				return LamNode{
					Var: "x",
					Body: LamNode{
						Var:  "y",
						Body: SubNode{Left: VarNode{Name: "x"}, Right: VarNode{Name: "y"}},
					},
				}
			} else {
				p.consume()
				p.consume()
				e := p.parseExpr()
				if p.peek().Type != TOK_RPAREN {
					panic(fmt.Errorf("expected ')'"))
				}
				p.consume()
				return LamNode{
					Var:  "x",
					Body: SubNode{Left: VarNode{Name: "x"}, Right: e},
				}
			}
		} else {
			p.consume() // '('
			first := p.parseExpr()
			if p.peek().Type == TOK_COMMA {
				p.consume()
				var elms []Node
				elms = append(elms, first)
				for {
					elms = append(elms, p.parseExpr())
					if p.peek().Type == TOK_COMMA {
						p.consume()
					} else if p.peek().Type == TOK_RPAREN {
						p.consume()
						break
					} else {
						panic(fmt.Errorf("expected ',' or ')' inside tuple"))
					}
				}
				return TupleNode{Elems: elms}
			} else {
				if p.peek().Type != TOK_RPAREN {
					panic(fmt.Errorf("expected ')'"))
				}
				p.consume()
				return first
			}
		}
	default:
		panic(fmt.Errorf("unexpected token %v inside atom expression", tok))
	}
}

func (p *Parser) parseListElements() Node {
	if p.peek().Type == TOK_RBRACK {
		p.consume()
		return NilNode{}
	}
	head := p.parseExpr()
	tok := p.peek()
	if tok.Type == TOK_PIPE {
		p.consume()

		hasLArrow := func() bool {
			depth := 0
			for i := p.pos; i < len(p.tokens); i++ {
				t := p.tokens[i]
				if t.Type == TOK_SEMICOLON && depth == 0 {
					return false
				}
				if t.Type == TOK_RBRACK && depth == 0 {
					return false
				}
				if t.Type == TOK_LARROW && depth == 0 {
					return true
				}
				switch t.Type {
				case TOK_LBRACE, TOK_LPAREN, TOK_LBRACK:
					depth++
				case TOK_RBRACE, TOK_RPAREN, TOK_RBRACK:
					depth--
				}
			}
			return false
		}

		var parseQualifiers func() []Qualifier
		parseQualifiers = func() []Qualifier {
			var q Qualifier
			if hasLArrow() {
				pat := p.parsePattern()
				if p.peek().Type != TOK_LARROW {
					panic(fmt.Errorf("expected '<-'"))
				}
				p.consume()
				src := p.parseExpr()
				q = GeneratorQual{Pat: pat, Src: src}
			} else {
				q = FilterQual{Cond: p.parseExpr()}
			}

			next := p.peek()
			if next.Type == TOK_SEMICOLON {
				p.consume()
				return append([]Qualifier{q}, parseQualifiers()...)
			} else if next.Type == TOK_RBRACK {
				p.consume()
				return []Qualifier{q}
			} else {
				panic(fmt.Errorf("expected ';' or ']' in qualifiers"))
			}
		}

		quals := parseQualifiers()
		return ZFNode{Body: head, Quals: quals}
	} else if tok.Type == TOK_DOTDOT {
		p.consume()
		tailExpr := p.parseExpr()
		if p.peek().Type != TOK_RBRACK {
			panic(fmt.Errorf("expected ']' after range expression"))
		}
		p.consume()
		return RangeNode{Start: head, End: tailExpr}
	} else if tok.Type == TOK_COMMA {
		p.consume()
		return ConsNode{Head: head, Tail: p.parseListElements()}
	} else if tok.Type == TOK_RBRACK {
		p.consume()
		return ConsNode{Head: head, Tail: NilNode{}}
	} else {
		panic(fmt.Errorf("expected '|', '..', ',', or ']' in list expression"))
	}
}

func (p *Parser) parsePattern() Pat {
	tok := p.peek()
	switch tok.Type {
	case TOK_INT:
		p.consume()
		return PatInt{Val: tok.Int}
	case TOK_CHAR:
		p.consume()
		return PatChar{Val: tok.Char}
	case TOK_VAR:
		p.consume()
		return PatVar{Name: tok.Str}
	case TOK_LBRACK:
		p.consume()
		if p.peek().Type == TOK_RBRACK {
			p.consume()
			return PatNil{}
		}
		panic(fmt.Errorf("only empty list pattern '[]' is supported directly"))
	case TOK_LPAREN:
		p.consume()
		var parseTuplePats func([]Pat) []Pat
		parseTuplePats = func(acc []Pat) []Pat {
			pCons := p.parsePatternCons()
			next := p.peek()
			if next.Type == TOK_COMMA {
				p.consume()
				return parseTuplePats(append(acc, pCons))
			} else if next.Type == TOK_RPAREN {
				p.consume()
				return append(acc, pCons)
			} else {
				panic(fmt.Errorf("expected ',' or ')' inside tuple pattern"))
			}
		}
		first := p.parsePatternCons()
		if p.peek().Type == TOK_COMMA {
			p.consume()
			return PatTuple{Elems: parseTuplePats([]Pat{first})}
		} else {
			if p.peek().Type != TOK_RPAREN {
				panic(fmt.Errorf("expected ')' in pattern"))
			}
			p.consume()
			return first
		}
	default:
		panic(fmt.Errorf("malformed pattern in equation left hand side: %v", tok))
	}
}

func (p *Parser) parsePatternCons() Pat {
	left := p.parsePattern()
	if p.peek().Type == TOK_COLON {
		p.consume()
		return PatCons{Head: left, Tail: p.parsePatternCons()}
	}
	return left
}

// ==========================================================================
// 7. PATTERN MATCHING DESUGARER
// ==========================================================================

var varCounter int

func newVarName(prefix string) string {
	c := varCounter
	varCounter++
	return fmt.Sprintf("%s_%d", prefix, c)
}

func desugarEquations(eqs []RawBinding) Node {
	if len(eqs) == 0 {
		panic("empty equation sequence")
	}
	if len(eqs) == 1 && len(eqs[0].Pats) == 0 {
		return eqs[0].Body
	}
	if len(eqs) == 1 && len(eqs[0].Pats) == 1 {
		if pv, ok := eqs[0].Pats[0].(PatVar); ok {
			return LamNode{Var: pv.Name, Body: eqs[0].Body}
		}
	}

	firstPats := eqs[0].Pats
	arity := len(firstPats)
	for _, eq := range eqs {
		if len(eq.Pats) != arity {
			panic("equations have mismatched parameter arities")
		}
	}

	var paramNames []string
	for i := 0; i < arity; i++ {
		paramNames = append(paramNames, fmt.Sprintf("p%d", i))
	}

	var buildDecisionTree func([]RawBinding) Node
	buildDecisionTree = func(restEqs []RawBinding) Node {
		if len(restEqs) == 0 {
			return MatchErrorNode{}
		}
		eq := restEqs[0]
		var checkPats func([]string, []Pat, Node) Node
		checkPats = func(params []string, pats []Pat, treeBody Node) Node {
			if len(params) == 0 && len(pats) == 0 {
				return treeBody
			}
			if len(params) == 0 || len(pats) == 0 {
				panic("internal pattern arity violation")
			}
			p := params[0]
			pat := pats[0]
			pRest := params[1:]
			patRest := pats[1:]

			switch pt := pat.(type) {
			case PatInt:
				cond := SubNode{Left: VarNode{Name: p}, Right: IntNode{Val: pt.Val}}
				return IfZeroNode{
					Cond: cond,
					Then: checkPats(pRest, patRest, treeBody),
					Else: buildDecisionTree(restEqs[1:]),
				}
			case PatChar:
				cond := SubNode{
					Left:  EqNode{Left: VarNode{Name: p}, Right: CharNode{Val: pt.Val}},
					Right: IntNode{Val: 1},
				}
				return IfZeroNode{
					Cond: cond,
					Then: checkPats(pRest, patRest, treeBody),
					Else: buildDecisionTree(restEqs[1:]),
				}
			case PatVar:
				substitutedBody := treeBody
				if pt.Name != p {
					substitutedBody = AppNode{
						Left:  LamNode{Var: pt.Name, Body: treeBody},
						Right: VarNode{Name: p},
					}
				}
				return checkPats(pRest, patRest, substitutedBody)
			case PatTuple:
				var elmsVars []string
				for i := 0; i < len(pt.Elems); i++ {
					elmsVars = append(elmsVars, newVarName(fmt.Sprintf("t%d", i)))
				}
				innerBody := checkPats(append(elmsVars, pRest...), append(pt.Elems, patRest...), treeBody)
				var wrapProjs func([]string, int, Node) Node
				wrapProjs = func(vars []string, idx int, body Node) Node {
					if len(vars) == 0 {
						return body
					}
					return AppNode{
						Left:  LamNode{Var: vars[0], Body: wrapProjs(vars[1:], idx+1, body)},
						Right: ProjNode{Index: idx, Tuple: VarNode{Name: p}},
					}
				}
				return wrapProjs(elmsVars, 0, innerBody)
			case PatNil:
				return IfNilNode{
					Cond: VarNode{Name: p},
					Then: checkPats(pRest, patRest, treeBody),
					Else: buildDecisionTree(restEqs[1:]),
				}
			case PatCons:
				hVar := newVarName("h")
				tVar := newVarName("t")
				failureBranch := buildDecisionTree(restEqs[1:])
				innerBody := checkPats(
					append([]string{hVar, tVar}, pRest...),
					append([]Pat{pt.Head, pt.Tail}, patRest...),
					treeBody,
				)
				return IfNilNode{
					Cond: VarNode{Name: p},
					Then: failureBranch,
					Else: AppNode{
						Left: LamNode{
							Var: hVar,
							Body: AppNode{
								Left:  LamNode{Var: tVar, Body: innerBody},
								Right: AppNode{Left: VarNode{Name: "tl"}, Right: VarNode{Name: p}},
							},
						},
						Right: AppNode{Left: VarNode{Name: "hd"}, Right: VarNode{Name: p}},
					},
				}
			}
			panic("unknown pattern type")
		}
		return checkPats(paramNames, eq.Pats, eq.Body)
	}

	decisionTree := buildDecisionTree(eqs)
	res := decisionTree
	for i := len(paramNames) - 1; i >= 0; i-- {
		res = LamNode{Var: paramNames[i], Body: res}
	}
	return res
}

// ==========================================================================
// 8. LAZY GRAPH REDUCTION MODEL & WHNF EVALUATOR
// ==========================================================================

func smlDiv(a, b int) int {
	q := a / b
	r := a % b
	if (r > 0 && b < 0) || (r < 0 && b > 0) {
		q--
	}
	return q
}

func smlMod(a, b int) int {
	r := a % b
	if (r > 0 && b < 0) || (r < 0 && b > 0) {
		r += b
	}
	return r
}

func needsThunkCons(n Node) bool {
	switch n.(type) {
	case IntNode, CharNode, NilNode, ThunkNode, ClosureNode, MatchErrorNode:
		return false
	}
	return true
}

func needsThunkTuple(n Node) bool {
	switch n.(type) {
	case IntNode, CharNode, NilNode, ThunkNode, ClosureNode, MatchErrorNode:
		return false
	}
	return true
}

type MatchBinding struct {
	Name string
	Val  Node
}

func mergeBindings(m1, m2 []MatchBinding) []MatchBinding {
	res := append([]MatchBinding(nil), m1...)
	for _, b2 := range m2 {
		found := false
		for i, b1 := range res {
			if b1.Name == b2.Name {
				res[i].Val = b2.Val
				found = true
				break
			}
		}
		if !found {
			res = append(res, b2)
		}
	}
	return res
}

func matchPattern(env *Env, pat Pat, node Node) ([]MatchBinding, bool) {
	v := whnf(env, node)
	switch p := pat.(type) {
	case PatInt:
		if i, ok := v.(IntNode); ok && p.Val == i.Val {
			return nil, true
		}
		return nil, false
	case PatChar:
		if c, ok := v.(CharNode); ok && p.Val == c.Val {
			return nil, true
		}
		return nil, false
	case PatVar:
		if p.Name == "_" {
			return nil, true
		}
		return []MatchBinding{{Name: p.Name, Val: v}}, true
	case PatNil:
		if _, ok := v.(NilNode); ok {
			return nil, true
		}
		return nil, false
	case PatCons:
		if c, ok := v.(ConsNode); ok {
			m1, ok1 := matchPattern(env, p.Head, c.Head)
			if !ok1 {
				return nil, false
			}
			m2, ok2 := matchPattern(env, p.Tail, c.Tail)
			if !ok2 {
				return nil, false
			}
			return mergeBindings(m1, m2), true
		}
		return nil, false
	case PatTuple:
		if t, ok := v.(TupleNode); ok {
			if len(p.Elems) != len(t.Elems) {
				return nil, false
			}
			var acc []MatchBinding
			for i := range p.Elems {
				m, ok := matchPattern(env, p.Elems[i], t.Elems[i])
				if !ok {
					return nil, false
				}
				acc = mergeBindings(acc, m)
			}
			return acc, true
		}
		return nil, false
	}
	return nil, false
}

func getStringValue(env *Env, node Node) string {
	var collect func(Node, []rune) string
	collect = func(current Node, acc []rune) string {
		switch l := whnf(env, current).(type) {
		case NilNode:
			return string(acc)
		case ConsNode:
			hVal := whnf(env, l.Head)
			if c, ok := hVal.(CharNode); ok {
				return collect(l.Tail, append(acc, c.Val))
			}
			panic(RuntimeError{Msg: "Expected char in string"})
		default:
			panic(RuntimeError{Msg: "Expected string"})
		}
	}
	return collect(node, []rune{})
}

func makeStringNode(s string) Node {
	runes := []rune(s)
	var makeList func([]rune) Node
	makeList = func(rs []rune) Node {
		if len(rs) == 0 {
			return NilNode{}
		}
		return ConsNode{
			Head: CharNode{Val: rs[0]},
			Tail: makeList(rs[1:]),
		}
	}
	return makeList(runes)
}

func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}
	fields := strings.Split(s, "\n")
	if len(fields) > 0 && fields[len(fields)-1] == "" {
		fields = fields[:len(fields)-1]
	}
	return fields
}

func removeOne(env *Env, x Node, listNode Node) Node {
	l := whnf(env, listNode)
	switch xs := l.(type) {
	case NilNode:
		return NilNode{}
	case ConsNode:
		eqH := whnf(env, EqNode{Left: x, Right: xs.Head})
		if iH, ok := eqH.(IntNode); ok && iH.Val == 1 {
			return xs.Tail
		}
		tEval := removeOne(env, x, xs.Tail)
		return ConsNode{Head: xs.Head, Tail: tEval}
	default:
		panic(RuntimeError{Msg: "-- expects lists"})
	}
}

func diff(env *Env, xs, ys Node) Node {
	currY := ys
	currX := xs
	for {
		yVal := whnf(env, currY)
		switch y := yVal.(type) {
		case NilNode:
			return currX
		case ConsNode:
			yEval := whnf(env, y.Head)
			currX = removeOne(env, yEval, currX)
			currY = y.Tail
		default:
			panic(RuntimeError{Msg: "-- expects lists"})
		}
	}
}

func evalZF(env *Env, bodyExpr Node, quals []Qualifier) Node {
	if len(quals) == 0 {
		h := bodyExpr
		if needsThunkCons(bodyExpr) {
			h = ThunkNode{Cell: &ThunkCell{State: Unevaluated, Expr: bodyExpr, Env: env}}
		}
		return ConsNode{Head: h, Tail: NilNode{}}
	}
	q := quals[0]
	switch qual := q.(type) {
	case FilterQual:
		cond := ThunkNode{Cell: &ThunkCell{State: Unevaluated, Expr: qual.Cond, Env: env}}
		return IfNode{Cond: cond, Then: evalZF(env, bodyExpr, quals[1:]), Else: NilNode{}}
	case GeneratorQual:
		return ZFGeneratorNode{
			Pat:   qual.Pat,
			Rest:  quals[1:],
			Src:   qual.Src,
			Body:  bodyExpr,
			ZFEnv: env,
		}
	}
	panic("Unknown qualifier type")
}

func eq(env *Env, v1, v2 Node) bool {
	switch x1 := v1.(type) {
	case IntNode:
		if x2, ok := v2.(IntNode); ok {
			return x1.Val == x2.Val
		}
	case CharNode:
		if x2, ok := v2.(CharNode); ok {
			return x1.Val == x2.Val
		}
	case NilNode:
		switch v2.(type) {
		case NilNode:
			return true
		case ConsNode:
			return false
		}
	case ConsNode:
		switch x2 := v2.(type) {
		case NilNode:
			return false
		case ConsNode:
			eqH := whnf(env, EqNode{Left: x1.Head, Right: x2.Head})
			if iH, okH := eqH.(IntNode); okH && iH.Val == 1 {
				eqT := whnf(env, EqNode{Left: x1.Tail, Right: x2.Tail})
				if iT, okT := eqT.(IntNode); okT && iT.Val == 1 {
					return true
				}
			}
			return false
		}
	case TupleNode:
		if x2, ok := v2.(TupleNode); ok {
			if len(x1.Elems) != len(x2.Elems) {
				return false
			}
			for i := range x1.Elems {
				eqE := whnf(env, EqNode{Left: x1.Elems[i], Right: x2.Elems[i]})
				if iE, okE := eqE.(IntNode); !okE || iE.Val != 1 {
					return false
				}
			}
			return true
		}
	}
	panic(RuntimeError{Msg: fmt.Sprintf("Equality expects integers, characters, lists or tuples, got: %s and %s", printNode(env, v1), printNode(env, v2))})
}

func whnf(env *Env, n Node) Node {
	for {
		switch node := n.(type) {
		case IntNode:
			return node
		case CharNode:
			return node
		case NilNode:
			return node
		case LamNode:
			return ClosureNode{Var: node.Var, Body: node.Body, Env: env}
		case ClosureNode:
			return node
		case LetNode:
			envPrime := env
			cells := make([]*ThunkCell, len(node.Bindings))
			for i, b := range node.Bindings {
				cells[i] = &ThunkCell{State: Unevaluated, Expr: b.Expr}
				envPrime = envPrime.Extend(b.Name, ThunkNode{Cell: cells[i]})
			}
			for _, cell := range cells {
				cell.Env = envPrime
			}
			env = envPrime
			n = node.Body
			continue
		case ConsNode:
			h := node.Head
			t := node.Tail
			if needsThunkCons(h) {
				h = ThunkNode{Cell: &ThunkCell{State: Unevaluated, Expr: h, Env: env}}
			}
			if needsThunkCons(t) {
				t = ThunkNode{Cell: &ThunkCell{State: Unevaluated, Expr: t, Env: env}}
			}
			return ConsNode{Head: h, Tail: t}
		case TupleNode:
			elmsPrime := make([]Node, len(node.Elems))
			for i, e := range node.Elems {
				if needsThunkTuple(e) {
					elmsPrime[i] = ThunkNode{Cell: &ThunkCell{State: Unevaluated, Expr: e, Env: env}}
				} else {
					elmsPrime[i] = e
				}
			}
			return TupleNode{Elems: elmsPrime}
		case VarNode:
			name := node.Name
			if name == "hd" || name == "tl" || name == "show" || name == "read" || name == "lines" || name == "numval" || name == "length" {
				return node
			}
			val, ok := env.Lookup(name)
			if !ok {
				panic(RuntimeError{Msg: "Unbound variable: " + name})
			}
			if th, ok := val.(ThunkNode); ok {
				cell := th.Cell
				switch cell.State {
				case Evaluated:
					n = cell.Val
					continue
				case Evaluating:
					panic(BlackholeError{Msg: "Infinite loop on identifier: " + name})
				case Unevaluated:
					cell.State = Evaluating
					res := whnf(cell.Env, cell.Expr)
					cell.State = Evaluated
					cell.Val = res
					n = res
					continue
				}
			}
			n = val
			continue
		case ThunkNode:
			cell := node.Cell
			switch cell.State {
			case Evaluated:
				n = cell.Val
				continue
			case Evaluating:
				panic(BlackholeError{Msg: "Infinite loop inside generic thunk node"})
			case Unevaluated:
				cell.State = Evaluating
				res := whnf(cell.Env, cell.Expr)
				cell.State = Evaluated
				cell.Val = res
				n = res
				continue
			}
		case IfNode:
			condVal := whnf(env, node.Cond)
			if i, ok := condVal.(IntNode); ok {
				if i.Val != 0 {
					n = node.Then
				} else {
					n = node.Else
				}
				continue
			}
			panic(RuntimeError{Msg: fmt.Sprintf("If condition must be an integer, got: %s", printNode(env, condVal))})
		case IfZeroNode:
			condVal := whnf(env, node.Cond)
			if i, ok := condVal.(IntNode); ok {
				if i.Val == 0 {
					n = node.Then
				} else {
					n = node.Else
				}
				continue
			}
			panic(RuntimeError{Msg: "Condition must resolve to an integer"})
		case IfNilNode:
			condVal := whnf(env, node.Cond)
			switch condVal.(type) {
			case NilNode:
				n = node.Then
				continue
			case ConsNode:
				n = node.Else
				continue
			default:
				panic(RuntimeError{Msg: "Condition must resolve to a list"})
			}
		case AppendNode:
			e1Val := whnf(env, node.Left)
			switch l := e1Val.(type) {
			case NilNode:
				n = node.Right
				continue
			case ConsNode:
				tPrime := ThunkNode{Cell: &ThunkCell{
					State: Unevaluated,
					Expr:  AppendNode{Left: l.Tail, Right: node.Right},
					Env:   env,
				}}
				return ConsNode{Head: l.Head, Tail: tPrime}
			default:
				panic(RuntimeError{Msg: "Append expects lists"})
			}
		case ZFNode:
			n = evalZF(env, node.Body, node.Quals)
			continue
		case ZFGeneratorNode:
			srcVal := whnf(node.ZFEnv, node.Src)
			switch s := srcVal.(type) {
			case NilNode:
				return NilNode{}
			case ConsNode:
				matchRes, matchOk := matchPattern(node.ZFEnv, node.Pat, s.Head)
				nextGen := ZFGeneratorNode{
					Pat:   node.Pat,
					Rest:  node.Rest,
					Src:   s.Tail,
					Body:  node.Body,
					ZFEnv: node.ZFEnv,
				}
				if matchOk {
					extendedEnv := node.ZFEnv
					for _, b := range matchRes {
						extendedEnv = extendedEnv.Extend(b.Name, b.Val)
					}
					firstList := evalZF(extendedEnv, node.Body, node.Rest)
					n = AppendNode{Left: firstList, Right: nextGen}
					continue
				} else {
					n = nextGen
					continue
				}
			default:
				panic(RuntimeError{Msg: "Generator source must be a list"})
			}
		case RangeNode:
			v1 := whnf(env, node.Start)
			v2 := whnf(env, node.End)
			i1, ok1 := v1.(IntNode)
			i2, ok2 := v2.(IntNode)
			if !ok1 || !ok2 {
				panic(RuntimeError{Msg: "Range bounds must evaluate to integers"})
			}
			if i1.Val > i2.Val {
				return NilNode{}
			}
			nextRange := RangeNode{Start: IntNode{Val: i1.Val + 1}, End: v2}
			tPrime := ThunkNode{Cell: &ThunkCell{
				State: Unevaluated,
				Expr:  nextRange,
				Env:   env,
			}}
			return ConsNode{Head: v1, Tail: tPrime}
		case ProjNode:
			tplVal := whnf(env, node.Tuple)
			if t, ok := tplVal.(TupleNode); ok {
				n = t.Elems[node.Index]
				continue
			}
			panic(RuntimeError{Msg: "Proj expects a tuple"})
		case MatchErrorNode:
			panic(RuntimeError{Msg: "Pattern matching exhausted"})
		case AddNode:
			v1 := whnf(env, node.Left)
			v2 := whnf(env, node.Right)
			i1, ok1 := v1.(IntNode)
			i2, ok2 := v2.(IntNode)
			if !ok1 || !ok2 {
				panic(RuntimeError{Msg: "Addition expects integers"})
			}
			return IntNode{Val: i1.Val + i2.Val}
		case SubNode:
			v1 := whnf(env, node.Left)
			v2 := whnf(env, node.Right)
			i1, ok1 := v1.(IntNode)
			i2, ok2 := v2.(IntNode)
			if !ok1 || !ok2 {
				panic(RuntimeError{Msg: "Subtraction expects integers"})
			}
			return IntNode{Val: i1.Val - i2.Val}
		case MulNode:
			v1 := whnf(env, node.Left)
			v2 := whnf(env, node.Right)
			i1, ok1 := v1.(IntNode)
			i2, ok2 := v2.(IntNode)
			if !ok1 || !ok2 {
				panic(RuntimeError{Msg: "Multiplication expects integers"})
			}
			return IntNode{Val: i1.Val * i2.Val}
		case DivNode:
			v1 := whnf(env, node.Left)
			v2 := whnf(env, node.Right)
			i1, ok1 := v1.(IntNode)
			i2, ok2 := v2.(IntNode)
			if !ok1 || !ok2 {
				panic(RuntimeError{Msg: "Division expects integers"})
			}
			if i2.Val == 0 {
				panic(RuntimeError{Msg: "Division by zero"})
			}
			return IntNode{Val: smlDiv(i1.Val, i2.Val)}
		case ModNode:
			v1 := whnf(env, node.Left)
			v2 := whnf(env, node.Right)
			i1, ok1 := v1.(IntNode)
			i2, ok2 := v2.(IntNode)
			if !ok1 || !ok2 {
				panic(RuntimeError{Msg: "Modulo expects integers"})
			}
			if i2.Val == 0 {
				panic(RuntimeError{Msg: "Division by zero"})
			}
			return IntNode{Val: smlMod(i1.Val, i2.Val)}
		case EqNode:
			v1 := whnf(env, node.Left)
			v2 := whnf(env, node.Right)
			if eq(env, v1, v2) {
				return IntNode{Val: 1}
			}
			return IntNode{Val: 0}
		case NeNode:
			v1 := whnf(env, node.Left)
			v2 := whnf(env, node.Right)
			if eq(env, v1, v2) {
				return IntNode{Val: 0}
			}
			return IntNode{Val: 1}
		case LtNode:
			v1 := whnf(env, node.Left)
			v2 := whnf(env, node.Right)
			i1, ok1 := v1.(IntNode)
			i2, ok2 := v2.(IntNode)
			if !ok1 || !ok2 {
				panic(RuntimeError{Msg: "Less-than expects integers"})
			}
			if i1.Val < i2.Val {
				return IntNode{Val: 1}
			}
			return IntNode{Val: 0}
		case GtNode:
			v1 := whnf(env, node.Left)
			v2 := whnf(env, node.Right)
			i1, ok1 := v1.(IntNode)
			i2, ok2 := v2.(IntNode)
			if !ok1 || !ok2 {
				panic(RuntimeError{Msg: "Greater-than expects integers"})
			}
			if i1.Val > i2.Val {
				return IntNode{Val: 1}
			}
			return IntNode{Val: 0}
		case LeNode:
			v1 := whnf(env, node.Left)
			v2 := whnf(env, node.Right)
			i1, ok1 := v1.(IntNode)
			i2, ok2 := v2.(IntNode)
			if !ok1 || !ok2 {
				panic(RuntimeError{Msg: "Less-than-or-equal expects integers"})
			}
			if i1.Val <= i2.Val {
				return IntNode{Val: 1}
			}
			return IntNode{Val: 0}
		case GeNode:
			v1 := whnf(env, node.Left)
			v2 := whnf(env, node.Right)
			i1, ok1 := v1.(IntNode)
			i2, ok2 := v2.(IntNode)
			if !ok1 || !ok2 {
				panic(RuntimeError{Msg: "Greater-than-or-equal expects integers"})
			}
			if i1.Val >= i2.Val {
				return IntNode{Val: 1}
			}
			return IntNode{Val: 0}
		case DiffNode:
			xs := whnf(env, node.Left)
			ys := whnf(env, node.Right)
			n = diff(env, xs, ys)
			continue
		case AppNode:
			fVal := whnf(env, node.Left)
			switch f := fVal.(type) {
			case VarNode:
				switch f.Name {
				case "hd":
					e2Val := whnf(env, node.Right)
					if c, ok := e2Val.(ConsNode); ok {
						n = c.Head
						continue
					}
					if _, ok := e2Val.(NilNode); ok {
						panic(RuntimeError{Msg: "hd applied to empty list"})
					}
					panic(RuntimeError{Msg: "hd expects a list"})
				case "tl":
					e2Val := whnf(env, node.Right)
					if c, ok := e2Val.(ConsNode); ok {
						n = c.Tail
						continue
					}
					if _, ok := e2Val.(NilNode); ok {
						panic(RuntimeError{Msg: "tl applied to empty list"})
					}
					panic(RuntimeError{Msg: "tl expects a list"})
				case "read":
					filename := getStringValue(env, node.Right)
					content, err := os.ReadFile(filename)
					if err != nil {
						panic(RuntimeError{Msg: fmt.Sprintf("Failed to read file: %s", filename)})
					}
					return makeStringNode(string(content))
				case "lines":
					content := getStringValue(env, node.Right)
					strList := splitLines(content)
					var makeNodeList func(strs []string) Node
					makeNodeList = func(strs []string) Node {
						if len(strs) == 0 {
							return NilNode{}
						}
						return ConsNode{
							Head: makeStringNode(strs[0]),
							Tail: makeNodeList(strs[1:]),
						}
					}
					return makeNodeList(strList)
				case "numval":
					s := getStringValue(env, node.Right)
					sTrimmed := strings.Map(func(r rune) rune {
						if unicode.IsSpace(r) {
							return -1
						}
						return r
					}, s)
					v, err := strconv.Atoi(sTrimmed)
					if err != nil {
						panic(RuntimeError{Msg: "numval: invalid integer: " + s})
					}
					return IntNode{Val: v}
				case "show":
					evaluatedNode := whnf(env, node.Right)
					s := printNode(env, evaluatedNode)
					return makeStringNode(s)
				case "length":
					length := 0
					curr := node.Right
					for {
						lVal := whnf(env, curr)
						if cons, ok := lVal.(ConsNode); ok {
							length++
							curr = cons.Tail
						} else if _, ok := lVal.(NilNode); ok {
							break
						} else {
							panic(RuntimeError{Msg: "length expects a list"})
						}
					}
					return IntNode{Val: length}
				default:
					panic(RuntimeError{Msg: "Unimplemented built-in: " + f.Name})
				}
			case ClosureNode:
				sharedThunk := ThunkNode{Cell: &ThunkCell{State: Unevaluated, Expr: node.Right, Env: env}}
				env = f.Env.Extend(f.Var, sharedThunk)
				n = f.Body
				continue
			case LamNode:
				sharedThunk := ThunkNode{Cell: &ThunkCell{State: Unevaluated, Expr: node.Right, Env: env}}
				env = env.Extend(f.Var, sharedThunk)
				n = f.Body
				continue
			default:
				panic(RuntimeError{Msg: "Non-functional application"})
			}
		}
		panic(fmt.Sprintf("Internal error: unhandled node type in whnf: %T", n))
	}
}

// ==========================================================================
// 9. STRING FORMATTING & ESCAPING FOR NODES
// ==========================================================================

func escapeChar(r rune) string {
	switch r {
	case '\n':
		return "\\n"
	case '\t':
		return "\\t"
	case '\'':
		return "\\'"
	case '\\':
		return "\\\\"
	default:
		return string(r)
	}
}

func escapeString(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case '\n':
			sb.WriteString("\\n")
		case '\t':
			sb.WriteString("\\t")
		case '"':
			sb.WriteString("\\\"")
		case '\\':
			sb.WriteString("\\\\")
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func printNode(env *Env, n Node) string {
	switch node := n.(type) {
	case IntNode:
		return strconv.Itoa(node.Val)
	case CharNode:
		return "'" + escapeChar(node.Val) + "'"
	case NilNode:
		return "[]"
	case LamNode:
		return "\\" + node.Var + ". <closure>"
	case ClosureNode:
		return "\\" + node.Var + ". <closure>"
	case LetNode:
		return "<let>"
	case VarNode:
		return node.Name
	case AppNode:
		return "(" + printNode(env, node.Left) + " " + printNode(env, node.Right) + ")"
	case SubNode:
		return "(" + printNode(env, node.Left) + " - " + printNode(env, node.Right) + ")"
	case AddNode:
		return "(" + printNode(env, node.Left) + " + " + printNode(env, node.Right) + ")"
	case MulNode:
		return "(" + printNode(env, node.Left) + " * " + printNode(env, node.Right) + ")"
	case DivNode:
		return "(" + printNode(env, node.Left) + " / " + printNode(env, node.Right) + ")"
	case DiffNode:
		return "(" + printNode(env, node.Left) + " -- " + printNode(env, node.Right) + ")"
	case EqNode:
		return "(" + printNode(env, node.Left) + " == " + printNode(env, node.Right) + ")"
	case NeNode:
		return "(" + printNode(env, node.Left) + " != " + printNode(env, node.Right) + ")"
	case LtNode:
		return "(" + printNode(env, node.Left) + " < " + printNode(env, node.Right) + ")"
	case GtNode:
		return "(" + printNode(env, node.Left) + " > " + printNode(env, node.Right) + ")"
	case LeNode:
		return "(" + printNode(env, node.Left) + " <= " + printNode(env, node.Right) + ")"
	case GeNode:
		return "(" + printNode(env, node.Left) + " >= " + printNode(env, node.Right) + ")"
	case ModNode:
		return "(" + printNode(env, node.Left) + " mod " + printNode(env, node.Right) + ")"
	case TupleNode:
		var elms []string
		for _, e := range node.Elems {
			elms = append(elms, printNode(env, whnf(env, e)))
		}
		return "(" + strings.Join(elms, ",") + ")"
	case IfZeroNode, IfNode:
		return "<conditional>"
	case IfNilNode:
		return "<conditional-nil>"
	case AppendNode:
		return "<append>"
	case ZFNode:
		return "<zf-comprehension>"
	case ZFGeneratorNode:
		return "<zf-generator>"
	case MatchErrorNode:
		return "<match-error>"
	case ThunkNode:
		return "<thunk>"
	case RangeNode:
		return "[" + printNode(env, node.Start) + ".." + printNode(env, node.End) + "]"
	case ConsNode:
		if s, isStr := isString(env, node); isStr {
			if s == "" {
				return "[]"
			}
			return "\"" + escapeString(s) + "\""
		}
		var elms []string
		curr := Node(node)
		for {
			currVal := whnf(env, curr)
			if cons, ok := currVal.(ConsNode); ok {
				elms = append(elms, printNode(env, whnf(env, cons.Head)))
				curr = cons.Tail
			} else if _, ok := currVal.(NilNode); ok {
				break
			} else {
				elms = append(elms, printNode(env, currVal))
				break
			}
		}
		return "[" + strings.Join(elms, ",") + "]"
	case ProjNode:
		return "<projection-" + strconv.Itoa(node.Index) + ">"
	}
	return "<unknown>"
}

func isString(env *Env, node Node) (string, bool) {
	var sb strings.Builder
	curr := node
	for {
		v := whnf(env, curr)
		switch val := v.(type) {
		case NilNode:
			return sb.String(), true
		case ConsNode:
			h := whnf(env, val.Head)
			if c, ok := h.(CharNode); ok {
				sb.WriteRune(c.Val)
				curr = val.Tail
			} else {
				return "", false
			}
		default:
			return "", false
		}
	}
}

func debugPrintNode(n Node) string {
	if n == nil {
		return "nil"
	}
	switch node := n.(type) {
	case IntNode:
		return fmt.Sprintf("Int(%d)", node.Val)
	case CharNode:
		return fmt.Sprintf("Char(%q)", node.Val)
	case NilNode:
		return "Nil"
	case VarNode:
		return fmt.Sprintf("Var(%s)", node.Name)
	case LamNode:
		return fmt.Sprintf("Lam(%s, %s)", node.Var, debugPrintNode(node.Body))
	case AppNode:
		return fmt.Sprintf("App(%s, %s)", debugPrintNode(node.Left), debugPrintNode(node.Right))
	case ConsNode:
		return fmt.Sprintf("Cons(%s, %s)", debugPrintNode(node.Head), debugPrintNode(node.Tail))
	case TupleNode:
		var elms []string
		for _, e := range node.Elems {
			elms = append(elms, debugPrintNode(e))
		}
		return fmt.Sprintf("Tuple(%s)", strings.Join(elms, ", "))
	case LetNode:
		var binds []string
		for _, b := range node.Bindings {
			binds = append(binds, fmt.Sprintf("%s=%s", b.Name, debugPrintNode(b.Expr)))
		}
		return fmt.Sprintf("Let([%s], %s)", strings.Join(binds, "; "), debugPrintNode(node.Body))
	case IfNode:
		return fmt.Sprintf("If(%s, %s, %s)", debugPrintNode(node.Cond), debugPrintNode(node.Then), debugPrintNode(node.Else))
	case IfZeroNode:
		return fmt.Sprintf("IfZero(%s, %s, %s)", debugPrintNode(node.Cond), debugPrintNode(node.Then), debugPrintNode(node.Else))
	case IfNilNode:
		return fmt.Sprintf("IfNil(%s, %s, %s)", debugPrintNode(node.Cond), debugPrintNode(node.Then), debugPrintNode(node.Else))
	case AddNode:
		return fmt.Sprintf("Add(%s, %s)", debugPrintNode(node.Left), debugPrintNode(node.Right))
	case SubNode:
		return fmt.Sprintf("Sub(%s, %s)", debugPrintNode(node.Left), debugPrintNode(node.Right))
	case MulNode:
		return fmt.Sprintf("Mul(%s, %s)", debugPrintNode(node.Left), debugPrintNode(node.Right))
	case DivNode:
		return fmt.Sprintf("Div(%s, %s)", debugPrintNode(node.Left), debugPrintNode(node.Right))
	case ModNode:
		return fmt.Sprintf("Mod(%s, %s)", debugPrintNode(node.Left), debugPrintNode(node.Right))
	case EqNode:
		return fmt.Sprintf("Eq(%s, %s)", debugPrintNode(node.Left), debugPrintNode(node.Right))
	case NeNode:
		return fmt.Sprintf("Ne(%s, %s)", debugPrintNode(node.Left), debugPrintNode(node.Right))
	case LtNode:
		return fmt.Sprintf("Lt(%s, %s)", debugPrintNode(node.Left), debugPrintNode(node.Right))
	case GtNode:
		return fmt.Sprintf("Gt(%s, %s)", debugPrintNode(node.Left), debugPrintNode(node.Right))
	case LeNode:
		return fmt.Sprintf("Le(%s, %s)", debugPrintNode(node.Left), debugPrintNode(node.Right))
	case GeNode:
		return fmt.Sprintf("Ge(%s, %s)", debugPrintNode(node.Left), debugPrintNode(node.Right))
	case ProjNode:
		return fmt.Sprintf("Proj(%d, %s)", node.Index, debugPrintNode(node.Tuple))
	case MatchErrorNode:
		return "MatchError"
	default:
		return fmt.Sprintf("%T", n)
	}
}

// ==========================================================================
// 10. LOAD SCRIPT FILE IMPLEMENTATION
// ==========================================================================

func loadScriptFile(filename string, env *Env) (*Env, error) {
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
	var layoutLines []layoutLine

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

		lineToks := tokenize(lineContent)
		var filtered []Token
		for _, t := range lineToks {
			if t.Type != TOK_EOF {
				filtered = append(filtered, t)
			}
		}

		wrapped := wrapWhereOnLine(filtered)
		if len(wrapped) > 0 {
			layoutLines = append(layoutLines, layoutLine{Indent: indent, Toks: wrapped})
		}
	}

	fileTokens := applyLayout(layoutLines)
	segments := splitTokens(fileTokens)

	var bindings []RawBinding
	for _, seg := range segments {
		err := func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					var tokStrs []string
					for _, t := range seg {
						tokStrs = append(tokStrs, tokenToString(t))
					}
					err = fmt.Errorf("parse error in segment:\n%s\nDetails: %v", strings.Join(tokStrs, " "), r)
				}
			}()
			parser := NewParser(seg)
			stmt := parser.parse()
			if bind, ok := stmt.(ScriptBindStmt); ok {
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

	grouped := make(map[string][]RawBinding)
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
		desugaredLambda := desugarEquations(eqList)
		accEnv = accEnv.Extend(name, desugaredLambda)
	}

	return accEnv, nil
}

func tokenToString(t Token) string {
	switch t.Type {
	case TOK_LAMBDA:
		return "\\"
	case TOK_DOT:
		return "."
	case TOK_DOTDOT:
		return ".."
	case TOK_ARROW:
		return "->"
	case TOK_ASSIGN:
		return "="
	case TOK_LPAREN:
		return "("
	case TOK_RPAREN:
		return ")"
	case TOK_LBRACK:
		return "["
	case TOK_RBRACK:
		return "]"
	case TOK_COMMA:
		return ","
	case TOK_COLON:
		return ":"
	case TOK_SUB:
		return "-"
	case TOK_ADD:
		return "+"
	case TOK_MUL:
		return "*"
	case TOK_DIV:
		return "/"
	case TOK_IFZERO:
		return "ifzero"
	case TOK_THEN:
		return "then"
	case TOK_ELSE:
		return "else"
	case TOK_INT:
		return strconv.Itoa(t.Int)
	case TOK_VAR:
		return t.Str
	case TOK_EOF:
		return "<EOF>"
	case TOK_PIPE:
		return "|"
	case TOK_LARROW:
		return "<-"
	case TOK_SEMICOLON:
		return ";"
	case TOK_EQ:
		return "=="
	case TOK_NE:
		return "~="
	case TOK_LT:
		return "<"
	case TOK_GT:
		return ">"
	case TOK_LE:
		return "<="
	case TOK_GE:
		return ">="
	case TOK_MOD:
		return "mod"
	case TOK_IF:
		return "if"
	case TOK_CHAR:
		return "'" + escapeChar(t.Char) + "'"
	case TOK_STRING:
		return "\"" + t.Str + "\""
	case TOK_PP:
		return "++"
	case TOK_WHERE:
		return "where"
	case TOK_LBRACE:
		return "{"
	case TOK_RBRACE:
		return "}"
	case TOK_HASH:
		return "#"
	case TOK_AND:
		return "&"
	case TOK_OR:
		return "\\/"
	case TOK_DIFF:
		return "--"
	}
	return ""
}

// ==========================================================================
// 11. MAIN ENTRYPOINT
// ==========================================================================

func isTTY() bool {
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

func readLine(prompt string, history []string) (string, []string, bool) {
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

	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)

	for {
		r, _, err := reader.ReadRune()
		if err != nil {
			return "", history, false
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

func runREPLDirect(env *Env, scriptFile string) {
	interactive := isTTY()
	var history []string
	var scanner *bufio.Scanner
	if !interactive {
		scanner = bufio.NewScanner(os.Stdin)
	}

	for {
		var line string
		if interactive {
			var ok bool
			line, history, ok = readLine("miranda> ", history)
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
			envWithStd, _ := loadScriptFile("stdenv.m", &Env{})
			reloadedEnv, err := loadScriptFile(scriptFile, envWithStd)
			if err != nil {
				fmt.Printf("Error reloading: %v\n", err)
			} else {
				env = reloadedEnv
			}
			continue
		}

		tokens := tokenize(lineTrimmed)
		func() {
			defer func() {
				if r := recover(); r != nil {
					if rtErr, ok := r.(RuntimeError); ok {
						fmt.Printf("Runtime Error: %s\n", rtErr.Msg)
					} else if bhErr, ok := r.(BlackholeError); ok {
						fmt.Printf("Runtime Error: %s\n", bhErr.Msg)
					} else {
						fmt.Printf("Error: %v\n", r)
					}
				}
			}()
			parser := NewParser(tokens)
			stmt := parser.parse()
			switch s := stmt.(type) {
			case ScriptBindStmt:
				finalLambda := desugarEquations([]RawBinding{s.Binding})
				env = env.Extend(s.Binding.FName, finalLambda)
				fmt.Printf("Defined variable: %s\n", s.Binding.FName)
			case REPLEvalStmt:
				startTime := time.Now()
				result := whnf(env, s.Expr)
				duration := time.Since(startTime).Milliseconds()

				sVal, isStr := isString(env, result)
				if isStr {
					fmt.Printf("Result:\n%s", sVal)
					if len(sVal) > 0 && sVal[len(sVal)-1] == '\n' {
						// no extra newline
					} else {
						fmt.Println()
					}
				} else {
					fmt.Printf("Result: %s\n", printNode(env, result))
				}
				fmt.Printf("Evaluation time: %d ms\n", duration)
			}
		}()
	}
}

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

	env := &Env{}
	var err error
	env, err = loadScriptFile("stdenv.m", env)
	if err != nil {
		fmt.Printf("Error loading stdenv.m: %v\n", err)
		os.Exit(1)
	}

	env, err = loadScriptFile(scriptFile, env)
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

	runREPLDirect(env, scriptFile)
}
