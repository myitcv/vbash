package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"mvdan.cc/sh/v3/syntax"
)

var (
	debugOut = false
	stdOut   = false

	fLog    = flag.Bool("log", false, "log output instead of running it")
	fGoRoot = flag.String("goroot", "", "path to GOROOT to use")

	tempFileMut  sync.Mutex
	tempFileName string
)

type block string

func (b *block) String() string {
	if b == nil {
		return "nil"
	}

	return string(*b)
}

func main() {
	os.Exit(main1())
}

func main1() int {
	if err := mainerr(); err != nil {
		if err, ok := err.(exitError); ok {
			return int(err)
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func mainerr() error {
	flag.Parse()

	var fn = "<stdin>"
	var err error

	if len(flag.Args()) > 0 {
		fn = flag.Arg(0)
	}

	var fi *os.File
	if len(flag.Args()) == 0 {
		fi = os.Stdin
	} else {
		fi, err = os.Open(fn)
		if err != nil {
			return fmt.Errorf("failed to open %v: %v", fn, err)
		}
	}

	f, err := syntax.NewParser(syntax.KeepComments).Parse(fi, fn)
	if err != nil {
		return fmt.Errorf("failed to parse %v: %v", fn, err)
	}

	wrapCall := func(s *syntax.Stmt) []*syntax.Stmt {
		var foundExpand bool
		for _, c := range s.Comments {
			if c.Text == "#" && c.Hash.Line() == s.Cmd.Pos().Line() {
				foundExpand = true
			}
		}
		newStmt := &syntax.Stmt{
			Position: s.Cmd.Pos(),
			Cmd:      s.Cmd,
			Redirs:   s.Redirs,
		}
		s.Redirs = nil
		var sb bytes.Buffer
		fmt.Fprintf(&sb, "OUR_LINE_NO=%v;\n", s.Cmd.Pos().Line())
		if foundExpand {
			fmt.Fprintf(&sb, "cat <<THISWILLNEVERMATCH\n> ")
		} else {
			fmt.Fprintf(&sb, "cat <<'THISWILLNEVERMATCH'\n> ")
		}
		s.Comments = nil
		syntax.NewPrinter().Print(&sb, newStmt)
		fmt.Fprintf(&sb, "\nTHISWILLNEVERMATCH\n")
		f, err := syntax.NewParser().Parse(strings.NewReader(sb.String()), "")
		if err != nil {
			panic(err)
		}
		f.Stmts = append(f.Stmts, newStmt)
		return f.Stmts
	}
	var toAmend []*syntax.Stmt
	var walk func(syntax.Node) bool
	walk = func(n syntax.Node) bool {
		if n == nil {
			return false
		}
		switch n := n.(type) {
		case *syntax.IfClause:
			for n != nil {
				for _, s := range n.Then.Stmts {
					syntax.Walk(s, walk)
				}
				n = n.Else
			}
			return false
		case *syntax.ProcSubst:
			return false
		case *syntax.CmdSubst:
			return false
		case *syntax.Subshell:
			return false
		case *syntax.FuncDecl:
			return false
		case *syntax.Stmt:
			var skipEcho bool
			for _, c := range n.Comments {
				if c.Text == "!" && c.Hash.Line() == n.Cmd.Pos().Line() {
					skipEcho = true
				}
			}
			switch n.Cmd.(type) {
			case *syntax.CallExpr:
				if !skipEcho {
					toAmend = append(toAmend, n)
				}
			case *syntax.BinaryCmd:
				if !skipEcho {
					toAmend = append(toAmend, n)
				}
				return false
			}
		}
		return true
	}
	syntax.Walk(f, walk)
	for _, n := range toAmend {
		n.Position = syntax.Pos{}
		n.Cmd = &syntax.Block{
			StmtList: syntax.StmtList{
				Stmts: wrapCall(n),
			},
		}
	}

	toRun := new(bytes.Buffer)
	fmt.Fprintf(toRun, `
set -o errtrace

trap 'set +u; echo Error on linue ${OUR_LINE_NO} in ${OUR_SOURCE_FILE}; exit 1' ERR

OUR_SOURCE_FILE="%[1]v"
`, fn)
	syntax.NewPrinter(syntax.Indent(2)).Print(toRun, f)

	if *fLog {
		fmt.Println(toRun.String())
		return nil
	}

	indent := os.Getenv("VBASHINDENT")

	cmd := exec.Command("env", "bash")
	// We want the wrapped process to write stdout and stderr to the same
	// io.Writer, because we ultimately define that vbash outputs everything
	// to stdout. This also means that indenting works.
	if indent != "" {
		cmd.Stdout = newIndentder(os.Stdout, indent)
		cmd.Stderr = cmd.Stdout
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stdout
	}
	if fi == os.Stdin {
		cmd.Stdin = toRun
	} else {
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt)
		done := make(chan bool)
		go func() {
			done := done
			for {
				select {
				case done <- true:
					done = nil
				case <-c:
					tempFileMut.Lock()
					if tempFileName != "" {
						os.Remove(tempFileName)
					}
					os.Exit(1)
				}
			}
		}()
		<-done
		cin, err := createTempFile(fi.Name(), toRun)
		defer func() {
			tempFileMut.Lock()
			defer tempFileMut.Unlock()
			if tempFileName != "" {
				os.Remove(tempFileName)
			}
		}()
		if err != nil {
			return err
		}
		cmd.Args = append(cmd.Args, cin.Name())
	}
	cmd.Env = append(os.Environ(), "VBASHINDENT="+indent+"\t")

	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			status := ee.Sys().(syscall.WaitStatus)
			return exitError(status.ExitStatus())
		} else {
			return fmt.Errorf("failed to run %v: %v", strings.Join(cmd.Args, " "), err)
		}
	}

	return nil
}

func createTempFile(orig string, b *bytes.Buffer) (*os.File, error) {
	tempFileMut.Lock()
	defer tempFileMut.Unlock()
	tf, err := ioutil.TempFile(filepath.Dir(orig), ".vbash_*.sh")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %v", err)
	}
	tempFileName = tf.Name()
	if _, err := io.Copy(tf, b); err != nil {
		return nil, fmt.Errorf("failed to write script to %v: %v", tempFileName, err)
	}
	if err := tf.Sync(); err != nil {
		return nil, fmt.Errorf("failed to sync %v: %v", tempFileName, err)
	}
	if _, err := tf.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek %v: %v", tempFileName, err)
	}
	return tf, nil
}

type exitError int

func (e exitError) Error() string {
	panic(fmt.Errorf("not expected to be called"))
}

func interfaceEqual(i1, i2 interface{}) bool {
	defer func() {
		recover()
	}()
	return i1 == i2
}

type indenter struct {
	und           io.Writer
	pendingIndent bool
	indent        []byte
}

func newIndentder(und io.Writer, indent string) *indenter {
	return &indenter{
		und:           und,
		indent:        append([]byte{'\n'}, []byte(indent)...),
		pendingIndent: true,
	}
}

func (i *indenter) Write(b []byte) (int, error) {
	r := bytes.Replace(b, []byte{'\n'}, i.indent, -1)
	if i.pendingIndent {
		r = append([]byte{'\t'}, r...)
		i.pendingIndent = false
	}
	if b[len(b)-1] == '\n' {
		r = r[:len(r)-1]
		i.pendingIndent = true
	}

	_, err := i.und.Write(r)

	if err == nil {
		return len(b), nil
	}

	return 0, err
}
