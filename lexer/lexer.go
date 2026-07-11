package lexer

import (
	"strings"
	"strconv"
	"unicode"
)

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

func Tokenize(str string) []Token {
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

func WrapWhereOnLine(toks []Token) []Token {
	var res []Token
	for i := 0; i < len(toks); i++ {
		if toks[i].Type == TOK_WHERE {
			res = append(res, toks[i])
			if i+1 < len(toks) {
				res = append(res, Token{Type: TOK_LBRACE})
				res = append(res, WrapWhereOnLine(toks[i+1:])...)
				res = append(res, Token{Type: TOK_RBRACE})
				break
			}
		} else {
			res = append(res, toks[i])
		}
	}
	return res
}

type LayoutLine struct {
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

func ApplyLayout(lines []LayoutLine) []Token {
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

func SplitTokens(tokens []Token) [][]Token {
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

func TokenToString(t Token) string {
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
