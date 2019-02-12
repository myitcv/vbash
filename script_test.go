package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestScripts(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Setup: func(env *testscript.Env) error {
			bin := filepath.Join(env.WorkDir, ".bin")

			cmd := exec.Command("go", "install")
			cmd.Env = append(os.Environ(), "GOBIN="+bin)

			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to run GOBIN=%v go install: %v\n%s", bin, err, out)
			}

			var path string
			for _, v := range env.Vars {
				if strings.HasPrefix(v, "PATH=") {
					path = v
				}
			}
			path = strings.TrimPrefix(path, "PATH=")
			env.Vars = append(env.Vars, "PATH="+bin+string(os.PathListSeparator)+path)
			return nil
		},
		Dir: "testdata",
	})
}
