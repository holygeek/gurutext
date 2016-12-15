package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const usage = `gurutext [-scope <scope>] file.go:#offset"
`

var (
	optScope   string
	optExclude string
	optSort    bool
	optComment string
	optIgnore  string = "GURUTEXT_IGNORE"

	optTest bool

	errOut io.Writer
)

func main() {
	errOut = os.Stderr
	flag.CommandLine.SetOutput(os.Stdout)
	flag.Usage = func() {
		fmt.Printf("%s\nOptions:\n", usage)
		flag.PrintDefaults()
	}
	flag.StringVar(&optScope, "scope", "", "Package `patterns` for guru's -scope argument")
	flag.StringVar(&optExclude, "exclude", "", "Exclude files matching the given `regex`")
	flag.BoolVar(&optSort, "sort", false, "Sort messages alphabetically")
	flag.StringVar(&optComment, "comment", "", "Extract comments that starts with `keyword`")
	flag.StringVar(&optIgnore, "ignore", optIgnore, "Ignore calls with preceeding comment that contains this text")
	flag.Parse()

	offsets := flag.Args()
	if len(offsets) <= 1 {
		bail("usage: %s", usage)
	}

	gettext(offsets)
}

type CallLocation struct {
	Pos    string
	Desc   string
	Caller string
}

func gettext(offsets []string) {
	var callers []CallLocation
	for _, position := range offsets {
		callers = append(callers, runGuru(position)...)
	}

	var excludeRe *regexp.Regexp
	if len(optExclude) > 0 {
		excludeRe = regexp.MustCompile(optExclude)
	}

	gt := NewGettext()
	for _, c := range callers {
		file, line, column := splitPos(c.Pos)
		if excludeRe != nil && excludeRe.MatchString(file) {
			continue
		}
		gt.Add(file, line, column)
	}

	gt.ExtractText()

	var visit func(Call)

	Print := func(call Call) {
		fmt.Printf("%s\n", call.AsGettext())
	}

	var sortedMessages CallSlice
	Collect := func(call Call) {
		sortedMessages = append(sortedMessages, call)
	}

	if optSort {
		visit = Collect
	} else {
		visit = Print
	}
	gt.Each(visit)

	if len(sortedMessages) > 0 {
		sort.Sort(sortedMessages)
		for _, calls := range sortedMessages {
			Print(calls)
		}
	}
}

func runGuru(offset string) []CallLocation {
	args := []string{"-json"}
	if optScope != "" {
		args = append(args, "-scope", optScope)
	}
	args = append(args, "callers", offset)

	var callers []CallLocation
	err := json.Unmarshal(run("guru", args...), &callers)
	if err != nil {
		bail("%v", err)
	}

	return callers
}

func asInt(str string) int {
	n, err := strconv.Atoi(str)
	if err != nil {
		bail("%s: %v", str, err)
	}
	return n
}

func run(name string, arg ...string) []byte {
	buf, err := exec.Command(name, arg...).Output()
	if err != nil {
		e := err.(*exec.ExitError)
		bail("%s", e.Stderr)
	}
	return buf
}

func bail(str string, arg ...interface{}) {
	fmt.Fprintf(errOut, "%s\n", format(str, arg...))
	os.Exit(1)
}

func format(s string, arg ...interface{}) string {
	str := fmt.Sprintf(s, arg...)
	if len(str) > 0 && str[len(str)-1] == '\n' {
		str = str[0 : len(str)-1]
	}
	return str
}

func splitPos(pos string) (file string, line, column int) {
	chunks := strings.Split(pos, ":")
	if len(chunks) != 3 {
		panic(fmt.Sprintf("pos not in /path/to/file.go:line:column format: %s", pos))
	}
	file, line, column = chunks[0], asInt(chunks[1]), asInt(chunks[2])
	return
}
