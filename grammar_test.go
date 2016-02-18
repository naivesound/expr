package expr

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestTokenize(t *testing.T) {
	// Let's preter there is no "&" operator, but there is "&&"
	defer func() {
		ops["&"] = bitwiseAnd
	}()
	delete(ops, "&")

	for s, parts := range map[string][]string{
		"2":         {"2"},
		"2+3/234.0": {"2", "+", "3", "/", "234.0"},
		"2+-3":      {"2", "+", "-u", "3"},
		"2--3":      {"2", "-", "-u", "3"},
		"-(-2)":     {"-u", "(", "-u", "2", ")"},
		"---2":      {"-u", "-u", "-u", "2"},
		"foo":       {"foo"},
		"1>2":       {"1", ">", "2"},
		"1>-2":      {"1", ">", "-u", "2"},
		"1>>2":      {"1", ">>", "2"},
		"1>>-2":     {"1", ">>", "-u", "2"},
		"1>>!2":     {"1", ">>", "!u", "2"},
		"1>>^!2":    {"1", ">>", "^u", "!u", "2"},
		"1&&2":      {"1", "&&", "2"},
		"1&&":       {"1", "&&"},
		"1&&&":      nil, // This should return an error: 'no such operator &'
	} {
		if tokens, err := tokenize([]rune(s)); err != nil {
			if parts != nil {
				t.Error(err, s)
			}
		} else if len(tokens) != len(parts) {
			t.Error(tokens, parts)
		} else {
			for i, tok := range tokens {
				if tok != parts[i] {
					t.Error(tokens, parts)
					break
				}
			}
		}
	}
}

func TestParse(t *testing.T) {
	env := map[string]Var{
		"x": NewVar(5),
	}
	funcs := map[string]Func{
		"add3": NewFunc(func(args FuncArgs, env FuncEnv) Num {
			if len(args) == 3 {
				return args[0].Eval() + args[1].Eval() + args[2].Eval()
			} else {
				return 0
			}
		}),
	}
	for input, result := range map[string]Num{
		"":    0,
		"2":   2,
		"(2)": 2,
		"x":   5,

		"-2": -2,
		"^2": -3,
		"!2": 0,
		"!0": 1,

		"3+2":       5,
		"3/2":       1.5,
		"(3/2)|0":   1, //Any bitwise operation will turn it into an int
		"2+3/2":     2 + 3/2.0,
		"4/2+8*4/2": 18,

		"2*x": 10,
		"2/x": 2 / 5.0,

		"4*(2+8)+4/2": 42,

		"2, 3, 5":  5,
		"2+3, 5*3": 15,

		"z=10":     10,
		"y=10,x+y": 15,

		"2+add3(3, 7, 9)":             21,
		"2+add3(3, add3(1, 2, 3), 9)": 20,
	} {
		if e, err := Parse(input, env, funcs); err != nil {
			t.Error(input, e, input, err)
		} else if n := e.Eval(); n != result {
			t.Error(input, e, n, result)
		}
	}
}

func TestParseFuzz(t *testing.T) {
	if testing.Short() {
		t.Skip("fuzzing test skipped")
	}
	env := map[string]Var{}
	funcs := map[string]Func{
		"f": NewFunc(func(args FuncArgs, env FuncEnv) Num {
			return 1
		}),
	}
	sym := "()+,1x>=f*"
	set := map[string]bool{}
	for i := 0; i < 40000; i++ {
		s := ""
		l := rand.Intn(10)
		for x := 0; x < l; x++ {
			s = s + string(sym[rand.Intn(len(sym))])
		}
		if e, err := Parse(s, env, funcs); err == nil {
			if !set[s] {
				set[s] = true
				t.Logf("%20s -> %v\n", s, e)
			}
		}
	}
}

func TestParseError(t *testing.T) {
	env := map[string]Var{}
	funcs := map[string]Func{
		"plusone": NewFunc(func(args FuncArgs, env FuncEnv) Num {
			return args[0].Eval() + 1
		}),
	}

	for input, e := range map[string]error{
		"(":   ErrParen,
		")":   ErrParen,
		"),":  ErrParen,
		")+(": ErrParen,
		"+(":  ErrOperandMissing,
		"2=3": ErrBadAssignment,
		"2@3": ErrBadOp,

		"1()":           ErrParen,
		"xfx((f1))":     ErrBadOp,
		",plusone(x)":   ErrOperandMissing,
		"plusone(,x)":   ErrOperandMissing,
		"plusone(x=)>1": ErrParen,

		"plusone": ErrBadCall,

		"1x":  ErrUnexpectedIdentifier,
		"1 x": ErrUnexpectedIdentifier,
		"1 1": ErrUnexpectedNumber,

		"2+":  ErrOperandMissing,
		"+2":  ErrOperandMissing,
		"+":   ErrOperandMissing,
		"-":   ErrOperandMissing,
		"1++": ErrOperandMissing,

		"+,": ErrOperandMissing,
	} {
		if expr, err := Parse(input, env, funcs); err != e {
			t.Error(e, err, expr, input)
		}
	}
}

func TestExprString(t *testing.T) {
	env := map[string]Var{
		"x": NewVar(5),
	}
	funcs := map[string]Func{
		"plusone": NewFunc(func(args FuncArgs, env FuncEnv) Num {
			return args[0].Eval() + 1
		}),
	}
	if e, err := Parse("-2+plusone(x)", env, funcs); err != nil {
		t.Error(err)
	} else if s := fmt.Sprintf("%v", e); s != "<8>(<1>(#2), fn[{5}])" {
		t.Error(e, s)
	}
}
