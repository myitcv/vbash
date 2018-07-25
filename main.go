package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"mvdan.cc/sh/syntax"
)

var (
	debugOut = false
	stdOut   = false

	fLog    = flag.Bool("log", false, "log output instead of running it")
	fGoRoot = flag.String("goroot", "", "path to GOROOT to use")
)

const (
	scriptName      = "script.sh"
	outputSeparator = "============================================="
)

type block string

func (b *block) String() string {
	if b == nil {
		return "nil"
	}

	return string(*b)
}

func main() {
	flag.Parse()

	dir := os.TempDir()
	var fn = "<stdin>"
	var err error

	if len(flag.Args()) > 0 {
		fn = flag.Arg(0)
		dir = filepath.Dir(fn)
	}

	toRun := new(bytes.Buffer)

	fmt.Fprintf(toRun, `
trap 'echo Error on linue ${OUR_LINE_NO} in ${OUR_SOURCE_FILE}' ERR

OUR_SOURCE_FILE="%[1]v"
`, fn)

	var fi *os.File
	if len(flag.Args()) == 0 {
		fi = os.Stdin
	} else {
		fi, err = os.Open(fn)
		if err != nil {
			errorf("failed to open %v: %v", fn, err)
		}
	}

	f, err := syntax.NewParser(syntax.KeepComments).Parse(fi, fn)
	if err != nil {
		errorf("failed to parse %v: %v", fn, err)
	}

	p := syntax.NewPrinter()

	stmtString := func(s *syntax.Stmt) string {
		// temporarily "blank" the comments associated with the stmt
		cs := s.Comments
		s.Comments = nil
		var b bytes.Buffer
		p.Print(&b, s)
		s.Comments = cs
		return b.String()
	}

	type cmdOutput struct {
		Cmd string
		Out string
	}

	for _, s := range f.Stmts {
		fmt.Fprintf(toRun, "OUR_LINE_NO=%v\n", s.Pos().Line())
		fmt.Fprintf(toRun, "cat <<'THISWILLNEVERMATCH'\n$ %v\nTHISWILLNEVERMATCH\n", stmtString(s))
		fmt.Fprintf(toRun, "%v\n", stmtString(s))
	}

	if *fLog {
		fmt.Printf("%v", toRun.String())
		return
	}

	tf, err := ioutil.TempFile(dir, ".vbash")
	if err != nil {
		errorf("failed to create temp file: %v", err)
	}

	tfn := tf.Name()

	defer func() {
		os.Remove(tfn)
	}()

	if err := ioutil.WriteFile(tfn, toRun.Bytes(), 0644); err != nil {
		errorf("failed to write to temp file %v: %v", tfn, err)
	}

	args := []string{"/usr/bin/env", "bash", tfn}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			status := ee.Sys().(syscall.WaitStatus)
			os.Exit(status.ExitStatus())
		} else {
			errorf("failed to run %v: %v", strings.Join(args, " "), err)
		}
	}
}

func errorf(format string, args ...interface{}) {
	if debugOut {
		panic(fmt.Errorf(format, args...))
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func debugf(format string, args ...interface{}) {
	if debugOut {
		fmt.Printf(format, args...)
	}
}
