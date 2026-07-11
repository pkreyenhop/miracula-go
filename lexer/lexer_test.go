package lexer

import (
	"testing"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		input    string
		expected []TokenType
	}{
		{"123", []TokenType{TOK_INT, TOK_EOF}},
		{"x", []TokenType{TOK_VAR, TOK_EOF}},
		{"\\x -> x", []TokenType{TOK_LAMBDA, TOK_VAR, TOK_ARROW, TOK_VAR, TOK_EOF}},
		{"if x then y else z", []TokenType{TOK_IF, TOK_VAR, TOK_THEN, TOK_VAR, TOK_ELSE, TOK_VAR, TOK_EOF}},
		{"'a'", []TokenType{TOK_CHAR, TOK_EOF}},
		{"\"hello\"", []TokenType{TOK_STRING, TOK_EOF}},
		{"x ++ y -- z", []TokenType{TOK_VAR, TOK_PP, TOK_VAR, TOK_DIFF, TOK_VAR, TOK_EOF}},
	}

	for _, tt := range tests {
		toks := Tokenize(tt.input)
		if len(toks) != len(tt.expected) {
			t.Fatalf("For input %q: expected %d tokens, got %d", tt.input, len(tt.expected), len(toks))
		}
		for i, tok := range toks {
			if tok.Type != tt.expected[i] {
				t.Errorf("For input %q at index %d: expected token type %v, got %v", tt.input, i, tt.expected[i], tok.Type)
			}
		}
	}
}

func TestApplyLayout(t *testing.T) {
	// Let's test a simple off-side layout rule conversion.
	// f x = x
	//   where
	//     y = 1
	//     z = 2
	lines := []LayoutLine{
		{Indent: 0, Toks: []Token{{Type: TOK_VAR, Str: "f"}, {Type: TOK_VAR, Str: "x"}, {Type: TOK_ASSIGN}, {Type: TOK_VAR, Str: "x"}}},
		{Indent: 2, Toks: []Token{{Type: TOK_WHERE}}},
		{Indent: 4, Toks: []Token{{Type: TOK_VAR, Str: "y"}, {Type: TOK_ASSIGN}, {Type: TOK_INT, Int: 1}}},
		{Indent: 4, Toks: []Token{{Type: TOK_VAR, Str: "z"}, {Type: TOK_ASSIGN}, {Type: TOK_INT, Int: 2}}},
	}

	toks := ApplyLayout(lines)
	// We expect braces inserted for where, and semicolons inserted between local definitions.
	// e.g., f x = x where { y = 1 ; z = 2 }
	var hasLBrace, hasRBrace, hasSemi bool
	for _, tok := range toks {
		if tok.Type == TOK_LBRACE {
			hasLBrace = true
		}
		if tok.Type == TOK_RBRACE {
			hasRBrace = true
		}
		if tok.Type == TOK_SEMICOLON {
			hasSemi = true
		}
	}

	if !hasLBrace || !hasRBrace {
		t.Errorf("Layout resolution failed to insert braces")
	}
	if !hasSemi {
		t.Errorf("Layout resolution failed to insert semicolon")
	}
}
