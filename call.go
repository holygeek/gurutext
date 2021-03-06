package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"strings"
)

type Call struct {
	Filename string
	Location
	Arg     string
	Status  ArgStatus
	Comment *ast.CommentGroup
}

func (c Call) AsGettext() string {
	buf := &bytes.Buffer{}

	if c.Comment != nil {
		text := c.Comment.Text()
		if len(text) > 0 && text[len(text)-1] == '\n' {
			text = text[0 : len(text)-1]
		}
		prefix := "#."
		for _, line := range strings.Split(text, "\n") {
			if len(line) > 0 {
				line = prefix + " " + line
			} else {
				line = prefix
			}
			fmt.Fprintf(buf, "%s\n", line)
		}
	}
	fmt.Fprintf(buf, "#: %s:%d:%d\n", c.Filename, c.Line, c.Column)
	fmt.Fprintf(buf, "msgid %s\n", c.Arg)
	fmt.Fprintf(buf, "msgstr \"\"\n")

	return buf.String()
}

type CallSlice []Call

func (cs CallSlice) Len() int           { return len(cs) }
func (cs CallSlice) Swap(i, j int)      { cs[i], cs[j] = cs[j], cs[i] }
func (cs CallSlice) Less(i, j int) bool { return cs[i].Arg < cs[j].Arg }
