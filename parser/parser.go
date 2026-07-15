package parser

import (
	"fmt"
	"pkreyenhop.com/miracula-go/ast"
	"pkreyenhop.com/miracula-go/lexer"
)

var varCounter int

// newVarName returns a synthetic binding name for desugaring. The leading
// '$' can never appear in a user identifier (the lexer rejects it), so these
// names cannot collide with user variables — which would otherwise let a
// user parameter named e.g. "p1" capture a desugarer-introduced reference.
func newVarName(prefix string) string {
	c := varCounter
	varCounter++
	return fmt.Sprintf("$%s_%d", prefix, c)
}

type RawBinding struct {
	FName string
	Pats  []ast.Pat
	Body  ast.Node
}

type Stmt interface {
	isStmt()
}

type ScriptBindStmt struct {
	Binding RawBinding
}

type REPLEvalStmt struct {
	Expr ast.Node
}

func (ScriptBindStmt) isStmt() {}
func (REPLEvalStmt) isStmt()   {}

type ParseError struct {
	Msg string
	Tok lexer.Token
}

func (e ParseError) Error() string {
	return e.Msg
}

type Parser struct {
	tokens   []lexer.Token
	pos      int
	filename string
}

func (p *Parser) errorf(format string, args ...interface{}) {
	panic(ParseError{
		Msg: fmt.Sprintf(format, args...),
		Tok: p.peek(),
	})
}

func NewParser(tokens []lexer.Token) *Parser {
	return &Parser{tokens: tokens, pos: 0}
}

func (p *Parser) WithFilename(filename string) *Parser {
	p.filename = filename
	return p
}

func (p *Parser) mark(node ast.Node, tok lexer.Token) ast.Node {
	if node != nil {
		key := ast.GetNodeKey(node)
		if key != nil {
			ast.NodePositions.Store(key, ast.Position{Filename: p.filename, Line: tok.Line, Col: tok.Col})
		}
	}
	return node
}

func (p *Parser) peek() lexer.Token {
	if p.pos >= len(p.tokens) {
		return lexer.Token{Type: lexer.TOK_EOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) peek2() lexer.Token {
	if p.pos+1 >= len(p.tokens) {
		return lexer.Token{Type: lexer.TOK_EOF}
	}
	return p.tokens[p.pos+1]
}

func (p *Parser) peek3() lexer.Token {
	if p.pos+2 >= len(p.tokens) {
		return lexer.Token{Type: lexer.TOK_EOF}
	}
	return p.tokens[p.pos+2]
}

func (p *Parser) consume() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

// patProjBindings turns a destructuring pattern LHS into the bindings that
// project each variable out of `src`. The caller binds `src` (a fresh variable)
// to the right-hand side once, so the RHS is evaluated a single time and shared.
// binopNode builds the AST node for a binary operator token applied to left
// and right, or ok=false if the token is not a sectionable binary operator.
// `&` and `\/` desugar to short-circuit conditionals, matching the grammar.
func binopNode(op lexer.TokenType, l, r ast.Node) (ast.Node, bool) {
	switch op {
	case lexer.TOK_ADD:
		return ast.AddNode{Left: l, Right: r}, true
	case lexer.TOK_MUL:
		return ast.MulNode{Left: l, Right: r}, true
	case lexer.TOK_DIV:
		return ast.DivNode{Left: l, Right: r}, true
	case lexer.TOK_MOD:
		return ast.ModNode{Left: l, Right: r}, true
	case lexer.TOK_EQ:
		return ast.EqNode{Left: l, Right: r}, true
	case lexer.TOK_NE:
		return ast.NeNode{Left: l, Right: r}, true
	case lexer.TOK_LT:
		return ast.LtNode{Left: l, Right: r}, true
	case lexer.TOK_GT:
		return ast.GtNode{Left: l, Right: r}, true
	case lexer.TOK_LE:
		return ast.LeNode{Left: l, Right: r}, true
	case lexer.TOK_GE:
		return ast.GeNode{Left: l, Right: r}, true
	case lexer.TOK_PP:
		return ast.AppendNode{Left: l, Right: r}, true
	case lexer.TOK_DIFF:
		return ast.DiffNode{Left: l, Right: r}, true
	case lexer.TOK_COLON:
		return ast.ConsNode{Head: l, Tail: r}, true
	case lexer.TOK_AND:
		return ast.IfNode{Cond: l, Then: r, Else: ast.BoolNode{Val: false}}, true
	case lexer.TOK_OR:
		return ast.IfNode{Cond: l, Then: ast.BoolNode{Val: true}, Else: r}, true
	}
	return nil, false
}

// buildLetNode groups let/where raw bindings by name (multi-equation
// functions), desugars each, and wraps the body in a LetNode (letrec).
func buildLetNode(raws []RawBinding, body ast.Node) ast.Node {
	grouped := make(map[string][]RawBinding)
	var order []string
	for _, b := range raws {
		if _, ok := grouped[b.FName]; !ok {
			order = append(order, b.FName)
		}
		grouped[b.FName] = append(grouped[b.FName], b)
	}
	var desugared []ast.Binding
	for _, name := range order {
		desugared = append(desugared, ast.Binding{Name: name, Expr: DesugarEquations(grouped[name])})
	}
	return ast.LetNode{Bindings: desugared, Body: body}
}

func patProjBindings(pat ast.Pat, src ast.Node) []RawBinding {
	switch pt := pat.(type) {
	case ast.PatVar:
		if pt.Name == "_" {
			return nil
		}
		return []RawBinding{{FName: pt.Name, Body: src}}
	case ast.PatTuple:
		var out []RawBinding
		for i, e := range pt.Elems {
			out = append(out, patProjBindings(e, ast.ProjNode{Index: i, Tuple: src})...)
		}
		return out
	case ast.PatCons:
		out := patProjBindings(pt.Head, ast.AppNode{Left: ast.VarNode{Name: "hd"}, Right: src})
		out = append(out, patProjBindings(pt.Tail, ast.AppNode{Left: ast.VarNode{Name: "tl"}, Right: src})...)
		return out
	}
	// literal and wildcard patterns bind nothing
	return nil
}

func isAssignment(tokens []lexer.Token) bool {
	depth := 0
	for _, t := range tokens {
		if t.Type == lexer.TOK_SEMICOLON && depth == 0 {
			return false
		}
		if t.Type == lexer.TOK_RBRACE && depth == 0 {
			return false
		}
		if t.Type == lexer.TOK_ASSIGN && depth == 0 {
			return true
		}
		switch t.Type {
		case lexer.TOK_LBRACE, lexer.TOK_LPAREN, lexer.TOK_LBRACK:
			depth++
		case lexer.TOK_RBRACE, lexer.TOK_RPAREN, lexer.TOK_RBRACK:
			depth--
		}
	}
	return false
}

type guardedClause struct {
	Expr ast.Node
	Cond ast.Node
}

func (p *Parser) parseRHS() ast.Node {
	if p.peek().Type != lexer.TOK_ASSIGN {
		p.errorf("expected '='")
	}
	p.consume() // '='

	expr := p.parseExpr()

	var body ast.Node
	// Check if the equation is guarded.
	// A guarded equation RHS has a comma followed by 'if' or 'otherwise'.
	if p.peek().Type == lexer.TOK_COMMA && (p.peek2().Type == lexer.TOK_IF || (p.peek2().Type == lexer.TOK_VAR && p.peek2().Str == "otherwise")) {
		// Yes, it is guarded!
		var clauses []guardedClause

		// Parse the first guard
		p.consume() // ','
		var cond ast.Node
		if p.peek().Type == lexer.TOK_IF {
			p.consume() // 'if'
			cond = p.parseExpr()
		} else {
			p.consume() // 'otherwise'
			cond = ast.BoolNode{Val: true}
		}
		clauses = append(clauses, guardedClause{Expr: expr, Cond: cond})

		// Parse subsequent clauses starting with '='
		for p.peek().Type == lexer.TOK_ASSIGN {
			p.consume() // '='
			nextExpr := p.parseExpr()
			if p.peek().Type != lexer.TOK_COMMA {
				p.errorf("expected ',' after expression in guarded clause")
			}
			p.consume() // ','
			var nextCond ast.Node
			if p.peek().Type == lexer.TOK_IF {
				p.consume() // 'if'
				nextCond = p.parseExpr()
			} else if p.peek().Type == lexer.TOK_VAR && p.peek().Str == "otherwise" {
				p.consume() // 'otherwise'
				nextCond = ast.BoolNode{Val: true}
			} else {
				p.errorf("expected 'if' or 'otherwise' after ',' in guarded clause")
			}
			clauses = append(clauses, guardedClause{Expr: nextExpr, Cond: nextCond})
		}

		// Desugar the clauses into nested IfNode / MatchErrorNode
		var desugared ast.Node = ast.MatchErrorNode{}
		for i := len(clauses) - 1; i >= 0; i-- {
			desugared = ast.IfNode{
				Cond: clauses[i].Cond,
				Then: clauses[i].Expr,
				Else: desugared,
			}
		}
		body = desugared
	} else {
		// Normal equation
		body = expr
	}

	// Parse optional trailing 'where' clause for the entire RHS
	if p.peek().Type == lexer.TOK_WHERE {
		p.consume()
		if p.peek().Type != lexer.TOK_LBRACE {
			p.errorf("expected '{' after 'where'")
		}
		p.consume()

		var parseBindings func() []RawBinding
		parseBindings = func() []RawBinding {
			if p.peek().Type == lexer.TOK_RBRACE {
				p.consume()
				return nil
			}
			if isAssignment(p.tokens[p.pos:]) {
				var bs []RawBinding
				if p.peek().Type == lexer.TOK_LPAREN {
					// destructuring binding: (a, b) = e  /  (x:xs) = e
					pat := p.parsePattern()
					localBody := p.parseRHS()
					dt := newVarName("dt")
					bs = []RawBinding{{FName: dt, Body: localBody}}
					bs = append(bs, patProjBindings(pat, ast.VarNode{Name: dt})...)
				} else {
					nameTok := p.peek()
					if nameTok.Type != lexer.TOK_VAR {
						p.errorf("left hand side of local binding must start with an identifier")
					}
					p.consume()
					var pats []ast.Pat
					for p.peek().Type != lexer.TOK_ASSIGN {
						pats = append(pats, p.parsePattern())
					}

					// Parse the RHS of the local binding recursively
					localBody := p.parseRHS()
					bs = []RawBinding{{FName: nameTok.Str, Pats: pats, Body: localBody}}
				}

				var rest []RawBinding
				if p.peek().Type == lexer.TOK_SEMICOLON {
					p.consume()
					rest = parseBindings()
				} else if p.peek().Type == lexer.TOK_RBRACE {
					p.consume()
				} else {
					p.errorf("expected ';' or '}' in where bindings")
				}
				return append(bs, rest...)
			} else {
				p.errorf("expected local binding in where clause")
				return nil
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

		var desugared []ast.Binding
		for _, name := range order {
			eqList := grouped[name]
			desugared = append(desugared, ast.Binding{
				Name: name,
				Expr: DesugarEquations(eqList),
			})
		}
		return ast.LetNode{Bindings: desugared, Body: body}
	}

	return body
}

// checkLexErrors rejects any character the lexer could not tokenise, with
// its exact source position, before parsing begins.
func (p *Parser) checkLexErrors() {
	for i, t := range p.tokens {
		if t.Type == lexer.TOK_ERROR {
			p.pos = i
			p.errorf("unrecognised character %q", t.Str)
		}
	}
}

func (p *Parser) Parse() Stmt {
	p.checkLexErrors()
	if isAssignment(p.tokens[p.pos:]) {
		tok := p.peek()
		if tok.Type != lexer.TOK_VAR {
			p.errorf("left hand side of binding must start with an identifier")
		}
		p.consume()
		var pats []ast.Pat
		for p.peek().Type != lexer.TOK_ASSIGN {
			pats = append(pats, p.parsePattern())
		}
		exprBody := p.parseRHS()
		return ScriptBindStmt{Binding: RawBinding{FName: tok.Str, Pats: pats, Body: exprBody}}
	} else {
		e := p.parseExpr()
		if p.peek().Type != lexer.TOK_EOF {
			p.errorf("trailing tokens left unparsed: %s", p.peek().String())
		}
		return REPLEvalStmt{Expr: e}
	}
}

func (p *Parser) parseExpr() ast.Node {
	var e ast.Node
	tok := p.peek()
	switch tok.Type {
	case lexer.TOK_LAMBDA:
		p.consume()
		if p.peek().Type == lexer.TOK_LPAREN {
			// pattern lambda: \(a, b). e  /  \(x:xs). e — desugared like a
			// one-equation function, so the pattern-match machinery (and its
			// tuple-arity handling) is reused
			pat := p.parsePattern()
			if p.peek().Type != lexer.TOK_DOT {
				p.errorf("expected '.' after lambda pattern")
			}
			p.consume()
			body := p.parseExpr()
			e = DesugarEquations([]RawBinding{{Pats: []ast.Pat{pat}, Body: body}})
		} else {
			v := p.peek()
			if v.Type != lexer.TOK_VAR {
				p.errorf("expected variable or pattern after lambda '\\'")
			}
			p.consume()
			if p.peek().Type != lexer.TOK_DOT {
				p.errorf("expected '.' after lambda variable")
			}
			p.consume()
			e = ast.LamNode{Var: v.Str, Body: p.parseExpr()}
		}
	case lexer.TOK_IFZERO:
		p.consume()
		cond := p.parseExpr()
		if p.peek().Type != lexer.TOK_THEN {
			p.errorf("expected 'then'")
		}
		p.consume()
		tBranch := p.parseExpr()
		if p.peek().Type != lexer.TOK_ELSE {
			p.errorf("expected 'else'")
		}
		p.consume()
		fBranch := p.parseExpr()
		e = ast.IfZeroNode{Cond: cond, Then: tBranch, Else: fBranch}
	case lexer.TOK_IF:
		p.consume()
		cond := p.parseExpr()
		if p.peek().Type != lexer.TOK_THEN {
			p.errorf("expected 'then'")
		}
		p.consume()
		tBranch := p.parseExpr()
		if p.peek().Type != lexer.TOK_ELSE {
			p.errorf("expected 'else'")
		}
		p.consume()
		fBranch := p.parseExpr()
		e = ast.IfNode{Cond: cond, Then: tBranch, Else: fBranch}
	case lexer.TOK_LET:
		// let b1 ; b2 ; ... in body   (bindings may destructure: (a,b) = e)
		p.consume()
		var raws []RawBinding
		for {
			if p.peek().Type == lexer.TOK_LPAREN {
				pat := p.parsePattern()
				body := p.parseRHS()
				dt := newVarName("dt")
				raws = append(raws, RawBinding{FName: dt, Body: body})
				raws = append(raws, patProjBindings(pat, ast.VarNode{Name: dt})...)
			} else {
				nameTok := p.peek()
				if nameTok.Type != lexer.TOK_VAR {
					p.errorf("expected identifier or pattern in 'let' binding")
				}
				p.consume()
				var pats []ast.Pat
				for p.peek().Type != lexer.TOK_ASSIGN {
					pats = append(pats, p.parsePattern())
				}
				body := p.parseRHS()
				raws = append(raws, RawBinding{FName: nameTok.Str, Pats: pats, Body: body})
			}
			if p.peek().Type == lexer.TOK_SEMICOLON {
				p.consume()
				continue
			}
			break
		}
		if p.peek().Type != lexer.TOK_IN {
			p.errorf("expected 'in' after 'let' bindings")
		}
		p.consume()
		e = buildLetNode(raws, p.parseExpr())
	default:
		e = p.parsePipe()
	}
	return p.mark(e, tok)
}

// parsePipe handles the pipe operator `x |> f`, sugar for the application
// `f x`. It is the loosest binary operator and left-associative, so
// `x |> f |> g` reads `g (f x)`.
func (p *Parser) parsePipe() ast.Node {
	tok := p.peek()
	left := p.parseOr()
	for p.peek().Type == lexer.TOK_PIPEGT {
		p.consume()
		right := p.parseOr()
		left = p.mark(ast.AppNode{Left: right, Right: left}, tok)
	}
	return p.mark(left, tok)
}

func (p *Parser) parseOr() ast.Node {
	tok := p.peek()
	left := p.parseAnd()
	if p.peek().Type == lexer.TOK_OR {
		p.consume()
		return p.mark(ast.IfNode{Cond: left, Then: ast.BoolNode{Val: true}, Else: p.parseOr()}, tok)
	}
	return p.mark(left, tok)
}

func (p *Parser) parseAnd() ast.Node {
	tok := p.peek()
	left := p.parseNot()
	if p.peek().Type == lexer.TOK_AND {
		p.consume()
		return p.mark(ast.IfNode{Cond: left, Then: p.parseAnd(), Else: ast.BoolNode{Val: false}}, tok)
	}
	return p.mark(left, tok)
}

// parseNot handles prefix logical negation `~e`, binding tighter than `&`
// and `\/` but looser than comparisons, so `~ a == b` reads ~(a == b).
func (p *Parser) parseNot() ast.Node {
	tok := p.peek()
	if tok.Type == lexer.TOK_NOT {
		p.consume()
		return p.mark(ast.IfNode{Cond: p.parseNot(), Then: ast.BoolNode{Val: false}, Else: ast.BoolNode{Val: true}}, tok)
	}
	return p.parseCons()
}

func (p *Parser) parseCons() ast.Node {
	tok := p.peek()
	left := p.parsePP()
	if p.peek().Type == lexer.TOK_COLON {
		p.consume()
		return p.mark(ast.ConsNode{Head: left, Tail: p.parseCons()}, tok)
	}
	return p.mark(left, tok)
}

func (p *Parser) parsePP() ast.Node {
	tok := p.peek()
	left := p.parseComp()
	nextTok := p.peek()
	if nextTok.Type == lexer.TOK_PP {
		p.consume()
		return p.mark(ast.AppendNode{Left: left, Right: p.parsePP()}, tok)
	} else if nextTok.Type == lexer.TOK_DIFF {
		p.consume()
		return p.mark(ast.DiffNode{Left: left, Right: p.parsePP()}, tok)
	}
	return p.mark(left, tok)
}

func (p *Parser) parseComp() ast.Node {
	tok := p.peek()
	left := p.parseAddSub()
	nextTok := p.peek()
	switch nextTok.Type {
	case lexer.TOK_EQ:
		p.consume()
		return p.mark(ast.EqNode{Left: left, Right: p.parseAddSub()}, tok)
	case lexer.TOK_NE:
		p.consume()
		return p.mark(ast.NeNode{Left: left, Right: p.parseAddSub()}, tok)
	case lexer.TOK_LT:
		p.consume()
		return p.mark(ast.LtNode{Left: left, Right: p.parseAddSub()}, tok)
	case lexer.TOK_GT:
		p.consume()
		return p.mark(ast.GtNode{Left: left, Right: p.parseAddSub()}, tok)
	case lexer.TOK_LE:
		p.consume()
		return p.mark(ast.LeNode{Left: left, Right: p.parseAddSub()}, tok)
	case lexer.TOK_GE:
		p.consume()
		return p.mark(ast.GeNode{Left: left, Right: p.parseAddSub()}, tok)
	}
	return p.mark(left, tok)
}

func (p *Parser) parseAddSub() ast.Node {
	tok := p.peek()
	left := p.parseMod()
	for {
		nextTok := p.peek()
		if nextTok.Type == lexer.TOK_ADD {
			p.consume()
			left = p.mark(ast.AddNode{Left: left, Right: p.parseMod()}, tok)
		} else if nextTok.Type == lexer.TOK_SUB {
			p.consume()
			left = p.mark(ast.SubNode{Left: left, Right: p.parseMod()}, tok)
		} else {
			break
		}
	}
	return p.mark(left, tok)
}

func (p *Parser) parseMod() ast.Node {
	tok := p.peek()
	left := p.parsePow()
	for {
		nextTok := p.peek()
		if nextTok.Type == lexer.TOK_MOD {
			p.consume()
			left = p.mark(ast.ModNode{Left: left, Right: p.parsePow()}, tok)
		} else if nextTok.Type == lexer.TOK_MUL {
			p.consume()
			left = p.mark(ast.MulNode{Left: left, Right: p.parsePow()}, tok)
		} else if nextTok.Type == lexer.TOK_DIV {
			p.consume()
			left = p.mark(ast.DivNode{Left: left, Right: p.parsePow()}, tok)
		} else {
			break
		}
	}
	return p.mark(left, tok)
}

// parsePow handles exponentiation `x ^ y`, tighter than `* / mod` and
// right-associative (`2 ^ 3 ^ 2` = `2 ^ (3 ^ 2)`).
func (p *Parser) parsePow() ast.Node {
	tok := p.peek()
	left := p.parseCompose()
	if p.peek().Type == lexer.TOK_CARET {
		p.consume()
		return p.mark(ast.PowNode{Left: left, Right: p.parsePow()}, tok)
	}
	return p.mark(left, tok)
}

func (p *Parser) parseCompose() ast.Node {
	tok := p.peek()
	left := p.parseIndex()
	if p.peek().Type == lexer.TOK_DOT {
		p.consume()
		right := p.parseCompose()
		varName := newVarName("cx")
		return p.mark(ast.LamNode{
			Var:  varName,
			Body: p.mark(ast.AppNode{Left: left, Right: p.mark(ast.AppNode{Left: right, Right: p.mark(ast.VarNode{Name: varName}, tok)}, tok)}, tok),
		}, tok)
	}
	return p.mark(left, tok)
}

// parseIndex handles the list subscript `xs ! n`, tighter than the arithmetic
// operators but looser than application (`f xs ! n` is `(f xs) ! n`), and
// left-associative (`grid ! i ! j`).
func (p *Parser) parseIndex() ast.Node {
	tok := p.peek()
	left := p.parseApp()
	for p.peek().Type == lexer.TOK_BANG {
		p.consume()
		left = p.mark(ast.IndexNode{List: left, Index: p.parseApp()}, tok)
	}
	return p.mark(left, tok)
}

func (p *Parser) parseApp() ast.Node {
	tok := p.peek()
	left := p.parseAtom()
	for {
		tokApp := p.peek()
		if tokApp.Type == lexer.TOK_INT || tokApp.Type == lexer.TOK_CHAR || tokApp.Type == lexer.TOK_STRING ||
			tokApp.Type == lexer.TOK_VAR || tokApp.Type == lexer.TOK_LPAREN || tokApp.Type == lexer.TOK_LBRACK {
			left = p.mark(ast.AppNode{Left: left, Right: p.parseAtom()}, tokApp)
		} else {
			break
		}
	}
	return p.mark(left, tok)
}

func makeStringNode(s string) ast.Node {
	runes := []rune(s)
	var makeList func([]rune) ast.Node
	makeList = func(rs []rune) ast.Node {
		if len(rs) == 0 {
			return ast.NilNode{}
		}
		return ast.ConsNode{
			Head: ast.CharNode{Val: rs[0]},
			Tail: makeList(rs[1:]),
		}
	}
	return makeList(runes)
}

func (p *Parser) parseAtom() ast.Node {
	tok := p.peek()
	var res ast.Node
	switch tok.Type {
	case lexer.TOK_HASH:
		p.consume()
		res = p.mark(ast.AppNode{Left: ast.VarNode{Name: "length"}, Right: p.parseAtom()}, tok)
	case lexer.TOK_INT:
		p.consume()
		res = ast.IntNode{Val: tok.Int}
	case lexer.TOK_CHAR:
		p.consume()
		res = ast.CharNode{Val: tok.Char}
	case lexer.TOK_STRING:
		p.consume()
		res = makeStringNode(tok.Str)
	case lexer.TOK_VAR:
		p.consume()
		res = ast.VarNode{Name: tok.Str}
	case lexer.TOK_SUB:
		p.consume()
		res = ast.SubNode{Left: ast.IntNode{Val: 0}, Right: p.parseAtom()}
	case lexer.TOK_LBRACK:
		p.consume()
		res = p.parseListElements()
	case lexer.TOK_LPAREN:
		if p.peek2().Type == lexer.TOK_SUB {
			// (-) is two-argument subtraction; (- e) is unary minus (negation),
			// not a section
			if p.peek3().Type == lexer.TOK_RPAREN {
				p.consume() // '('
				p.consume() // '-'
				p.consume() // ')'
				res = ast.LamNode{Var: "x", Body: ast.LamNode{Var: "y",
					Body: ast.SubNode{Left: ast.VarNode{Name: "x"}, Right: ast.VarNode{Name: "y"}}}}
			} else {
				p.consume() // '('
				p.consume() // '-'
				e := p.parseExpr()
				if p.peek().Type != lexer.TOK_RPAREN {
					p.errorf("expected ')'")
				}
				p.consume()
				res = ast.SubNode{Left: ast.IntNode{Val: 0}, Right: e}
			}
		} else if _, isBin := binopNode(p.peek2().Type, nil, nil); isBin {
			// operator sections: (op) is the two-argument operator, (op e) is a
			// right section \x. x op e
			op := p.peek2().Type
			if p.peek3().Type == lexer.TOK_RPAREN {
				p.consume() // '('
				p.consume() // op
				p.consume() // ')'
				vx := newVarName("s")
				vy := newVarName("s")
				body, _ := binopNode(op, ast.VarNode{Name: vx}, ast.VarNode{Name: vy})
				res = ast.LamNode{Var: vx, Body: ast.LamNode{Var: vy, Body: body}}
			} else {
				p.consume() // '('
				p.consume() // op
				e := p.parseExpr()
				if p.peek().Type != lexer.TOK_RPAREN {
					p.errorf("expected ')'")
				}
				p.consume()
				vx := newVarName("s")
				body, _ := binopNode(op, ast.VarNode{Name: vx}, e)
				res = ast.LamNode{Var: vx, Body: body}
			}
		} else {
			p.consume() // '('
			first := p.parseExpr()
			if p.peek().Type == lexer.TOK_COMMA {
				p.consume()
				var elms []ast.Node
				elms = append(elms, first)
				for {
					elms = append(elms, p.parseExpr())
					if p.peek().Type == lexer.TOK_COMMA {
						p.consume()
					} else if p.peek().Type == lexer.TOK_RPAREN {
						p.consume()
						break
					} else {
						p.errorf("expected ',' or ')' inside tuple")
					}
				}
				res = ast.TupleNode{Elems: elms}
			} else {
				if p.peek().Type != lexer.TOK_RPAREN {
					p.errorf("expected ')'")
				}
				p.consume()
				res = first
			}
		}
	default:
		p.errorf("unexpected token %s inside atom expression", tok.String())
		return nil
	}
	return p.mark(res, tok)
}

func (p *Parser) parseListElements() ast.Node {
	if p.peek().Type == lexer.TOK_RBRACK {
		p.consume()
		return ast.NilNode{}
	}
	head := p.parseExpr()
	tok := p.peek()
	if tok.Type == lexer.TOK_PIPE {
		p.consume()

		hasLArrow := func() bool {
			depth := 0
			for i := p.pos; i < len(p.tokens); i++ {
				t := p.tokens[i]
				if t.Type == lexer.TOK_SEMICOLON && depth == 0 {
					return false
				}
				if t.Type == lexer.TOK_RBRACK && depth == 0 {
					return false
				}
				if t.Type == lexer.TOK_LARROW && depth == 0 {
					return true
				}
				switch t.Type {
				case lexer.TOK_LBRACE, lexer.TOK_LPAREN, lexer.TOK_LBRACK:
					depth++
				case lexer.TOK_RBRACE, lexer.TOK_RPAREN, lexer.TOK_RBRACK:
					depth--
				}
			}
			return false
		}

		var parseQualifiers func() []ast.Qualifier
		parseQualifiers = func() []ast.Qualifier {
			var q ast.Qualifier
			if hasLArrow() {
				pat := p.parsePattern()
				if p.peek().Type != lexer.TOK_LARROW {
					p.errorf("expected '<-'")
				}
				p.consume()
				src := p.parseExpr()
				q = ast.GeneratorQual{Pat: pat, Src: src}
			} else {
				q = ast.FilterQual{Cond: p.parseExpr()}
			}

			next := p.peek()
			if next.Type == lexer.TOK_SEMICOLON {
				p.consume()
				return append([]ast.Qualifier{q}, parseQualifiers()...)
			} else if next.Type == lexer.TOK_RBRACK {
				p.consume()
				return []ast.Qualifier{q}
			} else {
				p.errorf("expected ';' or ']' in qualifiers")
				return nil
			}
		}

		quals := parseQualifiers()
		return ast.ZFNode{Body: head, Quals: quals}
	} else if tok.Type == lexer.TOK_DOTDOT {
		p.consume()
		if p.peek().Type == lexer.TOK_RBRACK {
			// unbounded range [start..]: an infinite lazy list
			p.consume()
			return ast.RangeFromNode{Start: head}
		}
		tailExpr := p.parseExpr()
		if p.peek().Type != lexer.TOK_RBRACK {
			p.errorf("expected ']' after range expression")
		}
		p.consume()
		return ast.RangeNode{Start: head, End: tailExpr}
	} else if tok.Type == lexer.TOK_COMMA {
		p.consume()
		second := p.parseExpr()
		if p.peek().Type == lexer.TOK_DOTDOT {
			// stepped range [head, second .. end] / [head, second ..]
			p.consume()
			step := ast.SubNode{Left: second, Right: head}
			if p.peek().Type == lexer.TOK_RBRACK {
				p.consume()
				return ast.RangeStepFromNode{Start: head, Step: step}
			}
			end := p.parseExpr()
			if p.peek().Type != lexer.TOK_RBRACK {
				p.errorf("expected ']' after stepped range")
			}
			p.consume()
			return ast.RangeStepNode{Start: head, Step: step, End: end}
		}
		// ordinary list literal: head : second : rest
		if p.peek().Type == lexer.TOK_COMMA {
			p.consume()
			return ast.ConsNode{Head: head, Tail: ast.ConsNode{Head: second, Tail: p.parseListElements()}}
		} else if p.peek().Type == lexer.TOK_RBRACK {
			p.consume()
			return ast.ConsNode{Head: head, Tail: ast.ConsNode{Head: second, Tail: ast.NilNode{}}}
		}
		p.errorf("expected ',', '..', or ']' in list expression")
		return nil
	} else if tok.Type == lexer.TOK_RBRACK {
		p.consume()
		return ast.ConsNode{Head: head, Tail: ast.NilNode{}}
	} else {
		p.errorf("expected '|', '..', ',', or ']' in list expression")
		return nil
	}
}

func (p *Parser) parsePattern() ast.Pat {
	tok := p.peek()
	switch tok.Type {
	case lexer.TOK_INT:
		p.consume()
		return ast.PatInt{Val: tok.Int}
	case lexer.TOK_CHAR:
		p.consume()
		return ast.PatChar{Val: tok.Char}
	case lexer.TOK_VAR:
		p.consume()
		if tok.Str == "True" {
			return ast.PatBool{Val: true}
		}
		if tok.Str == "False" {
			return ast.PatBool{Val: false}
		}
		return ast.PatVar{Name: tok.Str}
	case lexer.TOK_LBRACK:
		p.consume()
		if p.peek().Type == lexer.TOK_RBRACK {
			p.consume()
			return ast.PatNil{}
		}
		// fixed-length list pattern [p1, p2, ...] = (p1 : p2 : ... : [])
		var elems []ast.Pat
		for {
			elems = append(elems, p.parsePatternCons())
			if p.peek().Type == lexer.TOK_COMMA {
				p.consume()
			} else if p.peek().Type == lexer.TOK_RBRACK {
				p.consume()
				break
			} else {
				p.errorf("expected ',' or ']' in list pattern")
				return nil
			}
		}
		var listPat ast.Pat = ast.PatNil{}
		for i := len(elems) - 1; i >= 0; i-- {
			listPat = ast.PatCons{Head: elems[i], Tail: listPat}
		}
		return listPat
	case lexer.TOK_LPAREN:
		p.consume()
		var parseTuplePats func([]ast.Pat) []ast.Pat
		parseTuplePats = func(acc []ast.Pat) []ast.Pat {
			pCons := p.parsePatternCons()
			next := p.peek()
			if next.Type == lexer.TOK_COMMA {
				p.consume()
				return parseTuplePats(append(acc, pCons))
			} else if next.Type == lexer.TOK_RPAREN {
				p.consume()
				return append(acc, pCons)
			} else {
				p.errorf("expected ',' or ')' inside tuple pattern")
				return nil
			}
		}
		first := p.parsePatternCons()
		if p.peek().Type == lexer.TOK_COMMA {
			p.consume()
			return ast.PatTuple{Elems: parseTuplePats([]ast.Pat{first})}
		} else {
			if p.peek().Type != lexer.TOK_RPAREN {
				p.errorf("expected ')' in pattern")
			}
			p.consume()
			return first
		}
	default:
		p.errorf("malformed pattern in equation left hand side: %s", tok.String())
		return nil
	}
	return nil
}

func (p *Parser) parsePatternCons() ast.Pat {
	left := p.parsePattern()
	if p.peek().Type == lexer.TOK_COLON {
		p.consume()
		return ast.PatCons{Head: left, Tail: p.parsePatternCons()}
	}
	return left
}

// ==========================================================================
// Pattern Matching Desugarer
// ==========================================================================

func DesugarEquations(eqs []RawBinding) ast.Node {
	if len(eqs) == 0 {
		panic("empty equation sequence")
	}
	if len(eqs) == 1 && len(eqs[0].Pats) == 0 {
		return eqs[0].Body
	}
	if len(eqs) == 1 && len(eqs[0].Pats) == 1 {
		if pv, ok := eqs[0].Pats[0].(ast.PatVar); ok {
			return ast.LamNode{Var: pv.Name, Body: eqs[0].Body}
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
		paramNames = append(paramNames, newVarName("p"))
	}

	var buildDecisionTree func([]RawBinding) ast.Node
	buildDecisionTree = func(restEqs []RawBinding) ast.Node {
		if len(restEqs) == 0 {
			return ast.MatchErrorNode{}
		}
		eq := restEqs[0]
		// firstParam maps a pattern variable to the parameter/position of its
		// first occurrence within one equation, so a repeated variable can be
		// desugared into an equality check (a non-linear pattern).
		var checkPats func([]string, []ast.Pat, ast.Node, map[string]string) ast.Node
		checkPats = func(params []string, pats []ast.Pat, treeBody ast.Node, firstParam map[string]string) ast.Node {
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
			case ast.PatInt:
				cond := ast.SubNode{Left: ast.VarNode{Name: p}, Right: ast.IntNode{Val: pt.Val}}
				return ast.IfZeroNode{
					Cond: cond,
					Then: checkPats(pRest, patRest, treeBody, firstParam),
					Else: buildDecisionTree(restEqs[1:]),
				}
			case ast.PatChar:
				cond := ast.EqNode{Left: ast.VarNode{Name: p}, Right: ast.CharNode{Val: pt.Val}}
				return ast.IfNode{
					Cond: cond,
					Then: checkPats(pRest, patRest, treeBody, firstParam),
					Else: buildDecisionTree(restEqs[1:]),
				}
			case ast.PatBool:
				cond := ast.EqNode{Left: ast.VarNode{Name: p}, Right: ast.BoolNode{Val: pt.Val}}
				return ast.IfNode{
					Cond: cond,
					Then: checkPats(pRest, patRest, treeBody, firstParam),
					Else: buildDecisionTree(restEqs[1:]),
				}
			case ast.PatVar:
				// A variable repeated within one equation's patterns is a
				// non-linear pattern: it matches only when the repeated
				// positions are equal; otherwise control falls through to the
				// next equation. (`_` is exempt — each `_` is independent.)
				if pt.Name != "_" {
					if firstP, seen := firstParam[pt.Name]; seen {
						return ast.IfNode{
							Cond: ast.EqNode{Left: ast.VarNode{Name: firstP}, Right: ast.VarNode{Name: p}},
							Then: checkPats(pRest, patRest, treeBody, firstParam),
							Else: buildDecisionTree(restEqs[1:]),
						}
					}
					firstParam[pt.Name] = p
				}
				substitutedBody := treeBody
				if pt.Name != p {
					substitutedBody = ast.AppNode{
						Left:  ast.LamNode{Var: pt.Name, Body: treeBody},
						Right: ast.VarNode{Name: p},
					}
				}
				return checkPats(pRest, patRest, substitutedBody, firstParam)
			case ast.PatTuple:
				var elmsVars []string
				for i := 0; i < len(pt.Elems); i++ {
					elmsVars = append(elmsVars, newVarName(fmt.Sprintf("t%d", i)))
				}
				innerBody := checkPats(append(elmsVars, pRest...), append(pt.Elems, patRest...), treeBody, firstParam)
				var wrapProjs func([]string, int, ast.Node) ast.Node
				wrapProjs = func(vars []string, idx int, body ast.Node) ast.Node {
					if len(vars) == 0 {
						return body
					}
					return ast.AppNode{
						Left:  ast.LamNode{Var: vars[0], Body: wrapProjs(vars[1:], idx+1, body)},
						Right: ast.ProjNode{Index: idx, Tuple: ast.VarNode{Name: p}},
					}
				}
				return wrapProjs(elmsVars, 0, innerBody)
			case ast.PatNil:
				return ast.IfNilNode{
					Cond: ast.VarNode{Name: p},
					Then: checkPats(pRest, patRest, treeBody, firstParam),
					Else: buildDecisionTree(restEqs[1:]),
				}
			case ast.PatCons:
				hVar := newVarName("h")
				tVar := newVarName("t")
				failureBranch := buildDecisionTree(restEqs[1:])
				innerBody := checkPats(
					append([]string{hVar, tVar}, pRest...),
					append([]ast.Pat{pt.Head, pt.Tail}, patRest...),
					treeBody,
					firstParam,
				)
				return ast.IfNilNode{
					Cond: ast.VarNode{Name: p},
					Then: failureBranch,
					Else: ast.AppNode{
						Left: ast.LamNode{
							Var: hVar,
							Body: ast.AppNode{
								Left:  ast.LamNode{Var: tVar, Body: innerBody},
								Right: ast.AppNode{Left: ast.VarNode{Name: "tl"}, Right: ast.VarNode{Name: p}},
							},
						},
						Right: ast.AppNode{Left: ast.VarNode{Name: "hd"}, Right: ast.VarNode{Name: p}},
					},
				}
			}
			panic("unknown pattern type")
		}
		return checkPats(paramNames, eq.Pats, eq.Body, map[string]string{})
	}

	decisionTree := buildDecisionTree(eqs)
	res := decisionTree
	for i := len(paramNames) - 1; i >= 0; i-- {
		res = ast.LamNode{Var: paramNames[i], Body: res}
	}
	return res
}
