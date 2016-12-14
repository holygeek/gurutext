package main

import (
	"bytes"
	"fmt"
)

type Call struct {
	Filename string
	Location
	Arg    string
	Status ArgStatus
}

func (c Call) AsGettext() string {
	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "#: %s:%d:%d\n", c.Filename, c.Line, c.Column)
	fmt.Fprintf(buf, "msgid %s\n", c.Arg)
	fmt.Fprintf(buf, "msgstr \"\"\n")

	return buf.String()
}

type CallSlice []Call

func (cs CallSlice) Len() int           { return len(cs) }
func (cs CallSlice) Swap(i, j int)      { cs[i], cs[j] = cs[j], cs[i] }
func (cs CallSlice) Less(i, j int) bool { return cs[i].Arg < cs[j].Arg }
