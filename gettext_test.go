package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"
)

func init() {
	optTest = true
}

func TestGetArgSimple(t *testing.T) {
	const input = `package main

func A(arg string) {}

func main() {
	A("one")
}
`
	filename := "test.go"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, input, 0)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		locations []Location
		want      []Call
	}{
		{
			locations: []Location{
				Location{Line: 6, Column: 3},
				Location{Line: 999, Column: 9999},
			},
			want: []Call{{
				Location: Location{Line: 6, Column: 3},
				Arg:      `"one"`,
				Status:   ARG_FOUND,
			}, {
				Location: Location{Line: 999, Column: 9999},
				Arg:      ``,
				Status:   ARG_NOTFOUND,
			}},
		},
	}

	for i, tt := range tests {
		got := getArg(filename, fset, file, tt.locations)
		if len(tt.want) != len(got) {
			t.Errorf("%d: %s failed (different lengths)\nwant: %#v\n got: %#v",
				i, filename, tt.want, got)
		} else {
			for k, want := range tt.want {
				g := got[k]
				want.Filename = "test.go"
				if !reflect.DeepEqual(want, g) {
					t.Errorf("tests[%d].want[%d]: %s failed\nwant: %#v\n got: %#v",
						i, k, filename, want, g)
				}
			}
		}
	}
}

func TestGetArg(t *testing.T) {
	const input = `package main

func A(arg string) {}

func main() {
	A("one")
	A("two" + " " + "three")
	A("four" +
	" " +
	"five" + " " + "six")
	` + "A(`seven` + \"=\" + \"eight\")" + `
	A("with\\nnewline")
	A(` + "`backtick`" + `)
	` + "A(`01 multi" + `
02 line
03 backtick` + "`)" + `
	A(hello)
}
`
	filename := "test.go"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, input, 0)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		locations []Location
		want      []Call
	}{
		{
			locations: []Location{
				Location{Line: 6, Column: 3},
				Location{Line: 7, Column: 3},
				Location{Line: 8, Column: 3},
				Location{Line: 8, Column: 3},
				Location{Line: 11, Column: 3},
				Location{Line: 12, Column: 3},
				Location{Line: 13, Column: 3},
				Location{Line: 14, Column: 3},
				Location{Line: 15, Column: 3},
				Location{Line: 999, Column: 9999},
			},
			want: []Call{{
				Location: Location{Line: 6, Column: 3},
				Arg:      `"one"`,
				Status:   ARG_FOUND,
			}, {
				Location: Location{Line: 7, Column: 3},
				Arg:      `"two three"`,
				Status:   ARG_FOUND,
			}, {
				Location: Location{Line: 8, Column: 3},
				Arg:      `"four five six"`,
				Status:   ARG_FOUND,
			}, {
				Location: Location{Line: 11, Column: 3},
				Arg:      `"seven=eight"`,
				Status:   ARG_FOUND,
			}, {
				Location: Location{Line: 12, Column: 3},
				Arg:      `"with\\nnewline"`,
				Status:   ARG_FOUND,
			}, {
				Location: Location{Line: 13, Column: 3},
				Arg:      `"backtick"`,
				Status:   ARG_FOUND,
			}, {
				Location: Location{Line: 14, Column: 3},
				Arg:      `"01 multi\n02 line\n03 backtick"`,
				Status:   ARG_FOUND,
			}, {
				Location: Location{Line: 15, Column: 3},
				Arg:      ``,
				Status:   ARG_NOTFOUND,
			}, {
				Location: Location{Line: 999, Column: 9999},
				Arg:      ``,
				Status:   ARG_NOTFOUND,
			}},
		},
	}

	for i, tt := range tests {
		got := getArg(filename, fset, file, tt.locations)
		if len(tt.want) != len(got) {
			t.Errorf("%d: %s failed (different lengths)\nwant: %#v\n got: %#v",
				i, filename, tt.want, got)
		} else {
			for k, want := range tt.want {
				g := got[k]
				want.Filename = "test.go"
				if !reflect.DeepEqual(want, g) {
					t.Errorf("tests[%d].want[%d]: %s failed\nwant: %#v\n got: %#v",
						i, k, filename, want, g)
				}
			}
		}
	}
}

func newFset(t *testing.T, filename, input string) (*token.FileSet, *ast.File) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, input, 0)
	if err != nil {
		t.Fatal(err)
	}

	return fset, file
}

func TestWarning(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		0: {input: "package main\nfunc main() {\n\tA()\n}",
			want: "WARNING: no argument in function call\n" +
				"test.go:3:2:\n" +
				"	A()\n" +
				"	^\n",
		},
		1: {input: "package main\nfunc main() {\n\tA(foo())\n}",
			want: "WARNING: argument not a string literal (*ast.CallExpr):\n" +
				"test.go:3:4:\n" +
				"	A(foo())\n" +
				"	  ^\n",
		},
	}
	const (
		Line   = 3
		Column = 3
	)

	out := &bytes.Buffer{}
	errOut = out

	for i, tt := range tests {
		filename := "test.go"
		fileContent[filename] = strings.Split(tt.input, "\n")
		locations := []Location{Location{Line: Line, Column: Column}}
		fset, file := newFset(t, filename, tt.input)

		out.Reset()
		getArg("test.go", fset, file, locations)

		got := out.String()
		if tt.want != got {
			t.Errorf("tests[%d] failed\nwant: %s\n got: %s", i, tt.want, got)
		}
	}
}

type String string

// OneLine helps diffing got vs want
func (s String) OneLine() string {
	return strings.Replace(string(s), "\n", "•", -1)
}

func (s String) Replace(pattern, replacement string, n int) String {
	return String(strings.Replace(string(s), pattern, replacement, n))
}

func (s String) Diff(o String) string {
	buf := &bytes.Buffer{}
	so := s.OneLine()
	oo := o.OneLine()
	fmt.Fprintf(buf, "want: [%s]\n", so)
	fmt.Fprintf(buf, " got: [%s]\n", oo)
	fmt.Fprintf(buf, "       ")
	a := []rune(so)
	b := []rune(oo)
	if len(b) < len(a) {
		a, b = b, a
	}
	for i, ch := range a {
		m := ' '
		if b[i] != ch {
			m = '^'
		}
		fmt.Fprintf(buf, "%c", m)
	}
	fmt.Fprintln(buf)
	return buf.String()
}

func TestGettext(t *testing.T) {
	tests := []struct {
		input     string
		want      String
		errors    String
		locations []Location
	}{
		0: {
			input: "package main\n" +
				"func main() {\n" +
				"	A(`hello`)\n" +
				"	A(foo)\n" +
				"}",
			locations: []Location{
				Location{Line: 3, Column: 3},
				Location{Line: 4, Column: 3},
			},
			want: "#: {FILENAME}:3:3\n" +
				"msgid \"hello\"\n" +
				"msgstr \"\"\n" +
				"\n",
			errors: "WARNING: argument not a string literal (*ast.Ident):\n" +
				"{FILENAME}:4:4:\n" +
				"	A(foo)\n" +
				"	  ^\n",
		},
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	errOut = stderr

	for i, tt := range tests {
		filename := fmt.Sprintf("test_%d.go", i)
		tt.want = tt.want.Replace("{FILENAME}", filename, -1)
		tt.errors = tt.errors.Replace("{FILENAME}", filename, -1)
		fileContent[filename] = strings.Split(tt.input, "\n")

		gt := NewGettext()
		for _, l := range tt.locations {
			gt.Add(filename, l.Line, l.Column)
		}

		stdout.Reset()
		stderr.Reset()
		gt.ExtractText()
		errors := String(stderr.String())
		if tt.errors != errors {
			t.Errorf("tests[%d].errors failed\nwant: %s\n got: %s", i, tt.errors, errors)
			t.Errorf("Diff:\n%s", tt.errors.Diff(errors))
		}

		gt.Each(func(call Call) {
			fmt.Fprintf(stdout, "%s\n", call.AsGettext())
		})

		got := String(stdout.String())
		if tt.want != got {
			t.Errorf("tests[%d] failed\nwant: %s\n got: %s", i, tt.want, got)
			t.Errorf("Diff:\n%s", tt.want.Diff(got))
		}
	}
}
