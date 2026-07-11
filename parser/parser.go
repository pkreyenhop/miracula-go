package parser

import (
	"fmt"
	"pkreyenhop.com/miracula-go/ast"
	"pkreyenhop.com/miracula-go/lexer"
)

var varCounter int

func newVarName(prefix string) string {
	c := varCounter
	varCounter++
	return fmt.Sprintf("%s_%d", prefix, c)
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
	tokens []lexer.Token
	pos    int
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

func (p *Parser) Parse() Stmt {
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
		p.consume() // '='
		exprBody := p.parseExpr()
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
		v := p.peek()
		if v.Type != lexer.TOK_VAR {
			p.errorf("expected variable after lambda '\\'")
		}
		p.consume()
		if p.peek().Type != lexer.TOK_DOT {
			p.errorf("expected '.' after lambda variable")
		}
		p.consume()
		e = ast.LamNode{Var: v.Str, Body: p.parseExpr()}
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
	default:
		e = p.parseOr()
	}

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
				nameTok := p.peek()
				if nameTok.Type != lexer.TOK_VAR {
					p.errorf("left hand side of local binding must start with an identifier")
				}
				p.consume()
				var pats []ast.Pat
				for p.peek().Type != lexer.TOK_ASSIGN {
					pats = append(pats, p.parsePattern())
				}
				p.consume() // '='
				exprBody := p.parseExpr()
				b := RawBinding{FName: nameTok.Str, Pats: pats, Body: exprBody}

				var rest []RawBinding
				if p.peek().Type == lexer.TOK_SEMICOLON {
					p.consume()
					rest = parseBindings()
				} else if p.peek().Type == lexer.TOK_RBRACE {
					p.consume()
				} else {
					p.errorf("expected ';' or '}' in where bindings")
				}
				return append([]RawBinding{b}, rest...)
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
		return ast.LetNode{Bindings: desugared, Body: e}
	}
	return e
}

func (p *Parser) parseOr() ast.Node {
	left := p.parseAnd()
	if p.peek().Type == lexer.TOK_OR {
		p.consume()
		return ast.IfNode{Cond: left, Then: ast.IntNode{Val: 1}, Else: p.parseOr()}
	}
	return left
}

func (p *Parser) parseAnd() ast.Node {
	left := p.parseCons()
	if p.peek().Type == lexer.TOK_AND {
		p.consume()
		return ast.IfNode{Cond: left, Then: p.parseAnd(), Else: ast.IntNode{Val: 0}}
	}
	return left
}

func (p *Parser) parseCons() ast.Node {
	left := p.parsePP()
	if p.peek().Type == lexer.TOK_COLON {
		p.consume()
		return ast.ConsNode{Head: left, Tail: p.parseCons()}
	}
	return left
}

func (p *Parser) parsePP() ast.Node {
	left := p.parseComp()
	tok := p.peek()
	if tok.Type == lexer.TOK_PP {
		p.consume()
		return ast.AppendNode{Left: left, Right: p.parsePP()}
	} else if tok.Type == lexer.TOK_DIFF {
		p.consume()
		return ast.DiffNode{Left: left, Right: p.parsePP()}
	}
	return left
}

func (p *Parser) parseComp() ast.Node {
	left := p.parseAddSub()
	tok := p.peek()
	switch tok.Type {
	case lexer.TOK_EQ:
		p.consume()
		return ast.EqNode{Left: left, Right: p.parseAddSub()}
	case lexer.TOK_NE:
		p.consume()
		return ast.NeNode{Left: left, Right: p.parseAddSub()}
	case lexer.TOK_LT:
		p.consume()
		return ast.LtNode{Left: left, Right: p.parseAddSub()}
	case lexer.TOK_GT:
		p.consume()
		return ast.GtNode{Left: left, Right: p.parseAddSub()}
	case lexer.TOK_LE:
		p.consume()
		return ast.LeNode{Left: left, Right: p.parseAddSub()}
	case lexer.TOK_GE:
		p.consume()
		return ast.GeNode{Left: left, Right: p.parseAddSub()}
	}
	return left
}

func (p *Parser) parseAddSub() ast.Node {
	left := p.parseMod()
	for {
		tok := p.peek()
		if tok.Type == lexer.TOK_ADD {
			p.consume()
			left = ast.AddNode{Left: left, Right: p.parseMod()}
		} else if tok.Type == lexer.TOK_SUB {
			p.consume()
			left = ast.SubNode{Left: left, Right: p.parseMod()}
		} else {
			break
		}
	}
	return left
}

func (p *Parser) parseMod() ast.Node {
	left := p.parseCompose()
	for {
		tok := p.peek()
		if tok.Type == lexer.TOK_MOD {
			p.consume()
			left = ast.ModNode{Left: left, Right: p.parseCompose()}
		} else if tok.Type == lexer.TOK_MUL {
			p.consume()
			left = ast.MulNode{Left: left, Right: p.parseCompose()}
		} else if tok.Type == lexer.TOK_DIV {
			p.consume()
			left = ast.DivNode{Left: left, Right: p.parseCompose()}
		} else {
			break
		}
	}
	return left
}

func (p *Parser) parseCompose() ast.Node {
	left := p.parseApp()
	if p.peek().Type == lexer.TOK_DOT {
		p.consume()
		right := p.parseCompose()
		varName := newVarName("cx")
		return ast.LamNode{
			Var:  varName,
			Body: ast.AppNode{Left: left, Right: ast.AppNode{Left: right, Right: ast.VarNode{Name: varName}}},
		}
	}
	return left
}

func (p *Parser) parseApp() ast.Node {
	left := p.parseAtom()
	for {
		tok := p.peek()
		if tok.Type == lexer.TOK_INT || tok.Type == lexer.TOK_CHAR || tok.Type == lexer.TOK_STRING ||
			tok.Type == lexer.TOK_VAR || tok.Type == lexer.TOK_LPAREN || tok.Type == lexer.TOK_LBRACK {
			left = ast.AppNode{Left: left, Right: p.parseAtom()}
		} else {
			break
		}
	}
	return left
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
	switch tok.Type {
	case lexer.TOK_HASH:
		p.consume()
		return ast.AppNode{Left: ast.VarNode{Name: "length"}, Right: p.parseAtom()}
	case lexer.TOK_INT:
		p.consume()
		return ast.IntNode{Val: tok.Int}
	case lexer.TOK_CHAR:
		p.consume()
		return ast.CharNode{Val: tok.Char}
	case lexer.TOK_STRING:
		p.consume()
		return makeStringNode(tok.Str)
	case lexer.TOK_VAR:
		p.consume()
		return ast.VarNode{Name: tok.Str}
	case lexer.TOK_SUB:
		p.consume()
		return ast.SubNode{Left: ast.IntNode{Val: 0}, Right: p.parseAtom()}
	case lexer.TOK_LBRACK:
		p.consume()
		return p.parseListElements()
	case lexer.TOK_LPAREN:
		if p.peek2().Type == lexer.TOK_COLON {
			if p.peek3().Type == lexer.TOK_RPAREN {
				p.consume() // '('
				p.consume() // ':'
				p.consume() // ')'
				return ast.LamNode{
					Var: "x",
					Body: ast.LamNode{
						Var:  "y",
						Body: ast.ConsNode{Head: ast.VarNode{Name: "x"}, Tail: ast.VarNode{Name: "y"}},
					},
				}
			} else {
				p.consume() // '('
				p.consume() // ':'
				e := p.parseExpr()
				if p.peek().Type != lexer.TOK_RPAREN {
					p.errorf("expected ')'")
				}
				p.consume()
				return ast.LamNode{
					Var:  "x",
					Body: ast.ConsNode{Head: ast.VarNode{Name: "x"}, Tail: e},
				}
			}
		} else if p.peek2().Type == lexer.TOK_ADD {
			if p.peek3().Type == lexer.TOK_RPAREN {
				p.consume()
				p.consume()
				p.consume()
				return ast.LamNode{
					Var: "x",
					Body: ast.LamNode{
						Var:  "y",
						Body: ast.AddNode{Left: ast.VarNode{Name: "x"}, Right: ast.VarNode{Name: "y"}},
					},
				}
			} else {
				p.consume()
				p.consume()
				e := p.parseExpr()
				if p.peek().Type != lexer.TOK_RPAREN {
					p.errorf("expected ')'")
				}
				p.consume()
				return ast.LamNode{
					Var:  "x",
					Body: ast.AddNode{Left: ast.VarNode{Name: "x"}, Right: e},
				}
			}
		} else if p.peek2().Type == lexer.TOK_SUB {
			if p.peek3().Type == lexer.TOK_RPAREN {
				p.consume()
				p.consume()
				p.consume()
				return ast.LamNode{
					Var: "x",
					Body: ast.LamNode{
						Var:  "y",
						Body: ast.SubNode{Left: ast.VarNode{Name: "x"}, Right: ast.VarNode{Name: "y"}},
					},
				}
			} else {
				p.consume()
				p.consume()
				e := p.parseExpr()
				if p.peek().Type != lexer.TOK_RPAREN {
					p.errorf("expected ')'")
				}
				p.consume()
				return ast.LamNode{
					Var:  "x",
					Body: ast.SubNode{Left: ast.VarNode{Name: "x"}, Right: e},
				}
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
				return ast.TupleNode{Elems: elms}
			} else {
				if p.peek().Type != lexer.TOK_RPAREN {
					p.errorf("expected ')'")
				}
				p.consume()
				return first
			}
		}
	default:
		p.errorf("unexpected token %s inside atom expression", tok.String())
		return nil
	}
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
		tailExpr := p.parseExpr()
		if p.peek().Type != lexer.TOK_RBRACK {
			p.errorf("expected ']' after range expression")
		}
		p.consume()
		return ast.RangeNode{Start: head, End: tailExpr}
	} else if tok.Type == lexer.TOK_COMMA {
		p.consume()
		return ast.ConsNode{Head: head, Tail: p.parseListElements()}
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
		p.errorf("only empty list pattern '[]' is supported directly")
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
		paramNames = append(paramNames, fmt.Sprintf("p%d", i))
	}

	var buildDecisionTree func([]RawBinding) ast.Node
	buildDecisionTree = func(restEqs []RawBinding) ast.Node {
		if len(restEqs) == 0 {
			return ast.MatchErrorNode{}
		}
		eq := restEqs[0]
		var checkPats func([]string, []ast.Pat, ast.Node) ast.Node
		checkPats = func(params []string, pats []ast.Pat, treeBody ast.Node) ast.Node {
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
					Then: checkPats(pRest, patRest, treeBody),
					Else: buildDecisionTree(restEqs[1:]),
				}
			case ast.PatChar:
				cond := ast.EqNode{Left: ast.VarNode{Name: p}, Right: ast.CharNode{Val: pt.Val}}
				return ast.IfNode{
					Cond: cond,
					Then: checkPats(pRest, patRest, treeBody),
					Else: buildDecisionTree(restEqs[1:]),
				}
			case ast.PatBool:
				cond := ast.EqNode{Left: ast.VarNode{Name: p}, Right: ast.BoolNode{Val: pt.Val}}
				return ast.IfNode{
					Cond: cond,
					Then: checkPats(pRest, patRest, treeBody),
					Else: buildDecisionTree(restEqs[1:]),
				}
			case ast.PatVar:
				substitutedBody := treeBody
				if pt.Name != p {
					substitutedBody = ast.AppNode{
						Left:  ast.LamNode{Var: pt.Name, Body: treeBody},
						Right: ast.VarNode{Name: p},
					}
				}
				return checkPats(pRest, patRest, substitutedBody)
			case ast.PatTuple:
				var elmsVars []string
				for i := 0; i < len(pt.Elems); i++ {
					elmsVars = append(elmsVars, newVarName(fmt.Sprintf("t%d", i)))
				}
				innerBody := checkPats(append(elmsVars, pRest...), append(pt.Elems, patRest...), treeBody)
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
					Then: checkPats(pRest, patRest, treeBody),
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
		return checkPats(paramNames, eq.Pats, eq.Body)
	}

	decisionTree := buildDecisionTree(eqs)
	res := decisionTree
	for i := len(paramNames) - 1; i >= 0; i-- {
		res = ast.LamNode{Var: paramNames[i], Body: res}
	}
	return res
}
