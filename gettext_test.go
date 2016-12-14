package main

import (
	"go/parser"
	"go/token"
	"reflect"
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
				Status:   WANT_FOUND,
			}, {
				Location: Location{Line: 999, Column: 9999},
				Arg:      ``,
				Status:   WANT_NOTFOUND,
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
				Status:   WANT_FOUND,
			}, {
				Location: Location{Line: 7, Column: 3},
				Arg:      `"two three"`,
				Status:   WANT_FOUND,
			}, {
				Location: Location{Line: 8, Column: 3},
				Arg:      `"four five six"`,
				Status:   WANT_FOUND,
			}, {
				Location: Location{Line: 11, Column: 3},
				Arg:      `"seven=eight"`,
				Status:   WANT_FOUND,
			}, {
				Location: Location{Line: 12, Column: 3},
				Arg:      `"with\\nnewline"`,
				Status:   WANT_FOUND,
			}, {
				Location: Location{Line: 13, Column: 3},
				Arg:      `"backtick"`,
				Status:   WANT_FOUND,
			}, {
				Location: Location{Line: 14, Column: 3},
				Arg:      `"01 multi\n02 line\n03 backtick"`,
				Status:   WANT_FOUND,
			}, {
				Location: Location{Line: 15, Column: 3},
				Arg:      ``,
				Status:   WANT_NOTFOUND,
			}, {
				Location: Location{Line: 999, Column: 9999},
				Arg:      ``,
				Status:   WANT_NOTFOUND,
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