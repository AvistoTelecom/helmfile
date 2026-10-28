package helmexec

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	helmcmd "helm.sh/helm/v4/pkg/cmd"
)

// EmbeddedRunner runs helm commands in-process through the Helm v4 Go SDK, so no
// helm binary is needed on the host. Helm keeps its settings in package globals,
// so invocations are serialized: use it with --concurrency=1.
type EmbeddedRunner struct{}

var embeddedMu sync.Mutex

func (EmbeddedRunner) Execute(cmd string, args []string, env map[string]string, _ bool) ([]byte, error) {
	if filepath.Base(cmd) != "helm" {
		return nil, fmt.Errorf("embedded runner only runs helm, got %q", cmd)
	}

	embeddedMu.Lock()
	defer embeddedMu.Unlock()

	for k, v := range env {
		prev, had := os.LookupEnv(k)
		os.Setenv(k, v)
		defer func() {
			if had {
				os.Setenv(k, prev)
			} else {
				os.Unsetenv(k)
			}
		}()
	}

	var out bytes.Buffer
	root, err := helmcmd.NewRootCmd(&out, args, func(bool) {})
	if err != nil {
		return nil, err
	}
	root.SetArgs(args)
	root.SetOut(&out)
	root.SetErr(&out)
	err = root.Execute()
	return out.Bytes(), err
}

func (r EmbeddedRunner) ExecuteStdIn(cmd string, args []string, env map[string]string, stdin io.Reader) ([]byte, error) {
	return nil, fmt.Errorf("embedded runner does not support stdin")
}
