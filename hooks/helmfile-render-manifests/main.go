// Command helmfile-render-manifests runs the regular helmfile CLI with an embedded
// helm (no helm binary needed) and writes its stdout to a file, failing when the
// file was out of date or the output contains a Secret. Everything after "--" is
// passed to helmfile unchanged, e.g.:
//
//	helmfile-render-manifests --output int/manifests.yaml -- -f helmfile.yaml.gotmpl -e int template
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/helmfile/helmfile/cmd"
	"github.com/helmfile/helmfile/pkg/app"
	"github.com/helmfile/helmfile/pkg/config"
	"github.com/helmfile/helmfile/pkg/helmexec"
)

var secretKind = regexp.MustCompile(`(?m)^kind:\s*["']?Secret["']?\s*$`)

func main() {
	output := flag.String("output", "", "file the helmfile stdout is written to (required)")
	check := flag.Bool("check", false, "do not write, only fail when the output file is out of date")
	failSecret := flag.Bool("fail-on-secret", true, "fail when the output contains kind: Secret")
	flag.Parse()
	if *output == "" || flag.NArg() == 0 {
		fatal("usage: helmfile-render-manifests --output FILE [--check] [--fail-on-secret=false] -- <helmfile args>")
	}

	app.RunnerOverride = helmexec.EmbeddedRunner{}
	// A kube context would leak its namespace into the render; render context-free like CI.
	os.Setenv("KUBECONFIG", os.DevNull)

	rendered, err := run(flag.Args())
	if err != nil {
		fatal("%v", err)
	}
	if *failSecret && secretKind.Match(rendered) {
		fatal("%s: rendered manifests contain a Secret", *output)
	}
	current, _ := os.ReadFile(*output)
	if bytes.Equal(current, rendered) {
		return
	}
	if *check {
		fatal("%s is out of date", *output)
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fatal("%v", err)
	}
	if err := os.WriteFile(*output, rendered, 0o644); err != nil {
		fatal("%v", err)
	}
	fmt.Fprintf(os.Stderr, "updated %s\n", *output)
	os.Exit(1)
}

// run executes the helmfile CLI in-process and returns what it printed on stdout.
func run(args []string) ([]byte, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	stdout := os.Stdout
	os.Stdout = w
	done := make(chan []byte)
	go func() {
		b, _ := io.ReadAll(r)
		done <- b
	}()

	root, err := cmd.NewRootCmd(new(config.GlobalOptions))
	if err == nil {
		root.SetArgs(args)
		err = root.Execute()
	}
	os.Stdout = stdout
	w.Close()
	return <-done, err
}

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(2)
}
