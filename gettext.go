package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"strings"
)

type Gettext struct {
	Caller map[string][]Location
	Calls  map[string]CallSlice
}

func NewGettext() *Gettext {
	return &Gettext{
		Caller: map[string][]Location{},
		Calls:  map[string]CallSlice{},
	}
}

func (g *Gettext) Add(file string, line, column int) {
	g.Caller[file] = append(g.Caller[file], Location{Line: line, Column: column})
}

func (g *Gettext) ExtractText() {
	for file, locations := range g.Caller {
		fset := token.NewFileSet()
		var src interface{}
		lines := fileContent[file]
		if lines != nil {
			src = strings.Join(lines, "\n")
		}
		f, err := parser.ParseFile(fset, file, src, parser.ParseComments)
		if err != nil {
			bail("%s: %v", file, err)
		}
		g.Calls[file] = getArg(file, fset, f, locations)
	}
}

func (g *Gettext) Each(visit func(Call)) {
	for _, calls := range g.Calls {
		for _, call := range calls {
			if call.Status != ARG_FOUND {
				continue
			}
			visit(call)
		}
	}
}

type Location struct {
	Line   int
	Column int
}

type ArgStatus int8

const (
	ARG_NONE ArgStatus = iota
	ARG_PENDING
	ARG_FOUND
	ARG_NOTFOUND
)

func getArg(filename string, fset *token.FileSet, f *ast.File, locations []Location) []Call {
	if len(locations) == 0 {
		return nil
	}

	wantLocation := map[int]map[int]ArgStatus{}
	for _, l := range locations {
		if wantLocation[l.Line] == nil {
			wantLocation[l.Line] = map[int]ArgStatus{}
		}
		wantLocation[l.Line][l.Column] = ARG_PENDING
	}

	var comments map[int]*ast.CommentGroup
	comments = map[int]*ast.CommentGroup{}
	for _, cg := range f.Comments {
		p := fset.Position(cg.End())
		comments[p.Line] = cg
	}

	var commentPrefix string
	if optComment != "" {
		commentPrefix = "// " + optComment
	}

	var calls []Call
	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			return false
		}

		// fmt.Printf("Pos %s\n", fset.Position(n.Pos())) // DEBUG
		//
		// fmt.Printf("type: %#v\n", n) // DEBUG
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		p := fset.Position(call.Lparen)
		if wantLocation[p.Line] == nil {
			return true
		}
		if wantLocation[p.Line][p.Column] == ARG_NONE {
			return true
		}
		if len(call.Args) < 1 {
			showWarning(fset, call, "no argument in function call")
			return true
		}

		if comment := comments[p.Line-1]; comment != nil &&
			len(optIgnore) > 0 && strings.Contains(comment.Text(), optIgnore) {
			return true
		}
		var arg string
		switch x := call.Args[0].(type) {
		case *ast.BasicLit:
			arg = quote(getString(x.Value))
		case *ast.BinaryExpr:
			if x.Op != token.ADD {
				panic(fmt.Sprintf("%s: not an add operation", fset.Position(x.OpPos)))
			}
			arg = stringAdd(fset, x)
		case *ast.SelectorExpr, *ast.Ident, *ast.CallExpr:
			showWarning(fset, x, "argument not a string literal (%T):", x)
			return true
		default:
			showWarning(fset, x, "FIXME handle %T %#v", x, x)
			return true
		}

		var commentGroup *ast.CommentGroup
		if len(commentPrefix) > 0 {
			var comment ast.CommentGroup
			cg := comments[p.Line-1]
			if cg != nil {
				doAppend := false
				for _, c := range cg.List {
					if len(optIgnore) > 0 && strings.Contains(c.Text, optIgnore) {
						fmt.Fprintf(os.Stderr, "ignored (%s): %s\n", optIgnore, fset.Position(call.Args[0].Pos()))
						return true
					}
					if strings.HasPrefix(c.Text, commentPrefix) {
						doAppend = true
					}
					if doAppend {
						comment.List = append(comment.List, c)
					}
				}
				if len(comment.List) > 0 {
					commentGroup = &comment
				}
			}
		}

		calls = append(calls, Call{
			Filename: p.Filename,
			Location: Location{
				Line:   p.Line,
				Column: p.Column,
			},
			Arg:     arg,
			Status:  ARG_FOUND,
			Comment: commentGroup,
		})
		wantLocation[p.Line][p.Column] = ARG_FOUND
		return true
	})

	for _, loc := range locations {
		line := loc.Line
		for column, state := range wantLocation[line] {
			if state == ARG_PENDING {
				calls = append(calls, Call{
					Filename: filename,
					Location: Location{
						Line:   line,
						Column: column,
					},
					Arg:    "",
					Status: ARG_NOTFOUND,
				})
			}
		}
	}
	return calls
}

func showWarning(fset *token.FileSet, x ast.Expr, format string, args ...interface{}) {
	pos := fset.Position(x.Pos()).String()
	warn(format, args...)
	showLine(pos)
}

func warn(str string, arg ...interface{}) {
	fmt.Fprintf(errOut, "WARNING: %s\n", format(str, arg...))
}

func stringAdd(fset *token.FileSet, x *ast.BinaryExpr) string {
	s := getValue(fset, x.X) + getValue(fset, x.Y)
	return quote(s)
}

func getValue(fset *token.FileSet, x ast.Expr) string {
	switch ex := x.(type) {
	case *ast.BasicLit:
		return getString(ex.Value)
	case *ast.BinaryExpr:
		return getValue(fset, ex.X) + getValue(fset, ex.Y)
	default:
		panic(fmt.Sprintf("FIXME %s: unhandled expression type %T", fset.Position(ex.Pos()), ex))
	}
}

func getString(v string) string {
	s := v[1 : len(v)-1]
	if v[0] == '`' {
		s = strings.Replace(s, `"`, `\"`, -1)
		s = strings.Replace(s, "\n", "\\n", -1)
	}
	return s
}

func quote(s string) string {
	return `"` + s + `"`
}

var fileContent = map[string][]string{}

func showLine(pos string) {
	file, lnum, column := splitPos(pos)
	lines := fileContent[file]
	if lines == nil {
		f, err := os.Open(file)
		if err != nil {
			if optTest {
				return
			}
			bail("%s: %v", file, err)
		}
		buf, err := ioutil.ReadAll(f)
		if err != nil {
			bail("%s: %v", file, err)
		}
		for _, l := range bytes.Split(buf, []byte{'\n'}) {
			lines = append(lines, string(l))
		}
		fileContent[file] = lines
	}

	if lnum > len(lines) {
		panic(fmt.Sprintf("lnum %d > lines in %s (%d)", lnum, file, len(lines)))
	}

	line := lines[lnum-1]
	fmt.Fprintf(errOut, "%s:\n", pos)
	fmt.Fprintf(errOut, "%s\n", line)
	for _, c := range line[0 : column-1] {
		m := ' '
		if c == '\t' {
			m = c
		}
		fmt.Fprintf(errOut, "%c", m)
	}
	fmt.Fprintln(errOut, "^")
}
