// Command helmfile-render-manifests renders helmfile environments to files and
// fails when they are out of date. It embeds helm, so neither helm nor helmfile
// needs to be installed: it is meant to be used as a pre-commit hook.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/helmfile/helmfile/cmd"
	"github.com/helmfile/helmfile/pkg/app"
	"github.com/helmfile/helmfile/pkg/config"
	"github.com/helmfile/helmfile/pkg/helmexec"
)

var secretKind = regexp.MustCompile(`(?m)^kind:\s*["']?Secret["']?\s*$`)

type list []string

func (l *list) String() string     { return strings.Join(*l, ",") }
func (l *list) Set(v string) error { *l = append(*l, v); return nil }

func main() {
	var (
		envs        list
		stateSet    list
		file        = flag.String("file", "", "helmfile path (default: helmfile's own lookup)")
		output      = flag.String("output", "{env}/manifests/manifests.yaml", "output path; {env} is replaced by the environment name")
		check       = flag.Bool("check", false, "do not write, only fail when an output file is out of date")
		failSecret  = flag.Bool("fail-on-secret", true, "fail when a rendered manifest has kind: Secret")
		kubeVersion = flag.String("kube-version", "", "pass --kube-version to helm template")
		includeCRDs = flag.Bool("include-crds", false, "include CRDs in the output")
		skipTests   = flag.Bool("skip-tests", false, "skip tests from the output")
	)
	flag.Var(&envs, "environment", "environment to render (repeatable, required)")
	flag.Var(&envs, "e", "shorthand for --environment")
	flag.Var(&stateSet, "state-values-set", "state value override key=value (repeatable)")
	flag.Parse()
	if len(envs) == 0 {
		fatal("at least one --environment is required")
	}

	app.RunnerOverride = helmexec.EmbeddedRunner{}
	// ponytail: a kube context would leak its namespace into the render; always render context-free like CI.
	os.Setenv("KUBECONFIG", os.DevNull)

	failed := false
	for _, env := range envs {
		args := []string{"--log-level", "error", "-e", env}
		if *file != "" {
			args = append(args, "-f", *file)
		}
		for _, s := range stateSet {
			args = append(args, "--state-values-set", s)
		}
		args = append(args, "template", "--concurrency=1")
		if *kubeVersion != "" {
			args = append(args, "--kube-version", *kubeVersion)
		}
		if *includeCRDs {
			args = append(args, "--include-crds")
		}
		if *skipTests {
			args = append(args, "--skip-tests")
		}
		args = append(args, flag.Args()...)

		rendered, err := render(args)
		if err != nil {
			fatal("%s: %v", env, err)
		}
		if *failSecret && secretKind.Match(rendered) {
			fmt.Fprintf(os.Stderr, "%s: rendered manifests contain a Secret\n", env)
			failed = true
			continue
		}

		path := strings.ReplaceAll(*output, "{env}", env)
		current, _ := os.ReadFile(path)
		if bytes.Equal(current, rendered) {
			continue
		}
		failed = true
		if *check {
			fmt.Fprintf(os.Stderr, "%s is out of date\n", path)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			fatal("%v", err)
		}
		if err := os.WriteFile(path, rendered, 0o644); err != nil {
			fatal("%v", err)
		}
		fmt.Fprintf(os.Stderr, "updated %s\n", path)
	}
	if failed {
		os.Exit(1)
	}
}

// render runs `helmfile <args>` in-process and returns what it printed on stdout.
func render(args []string) ([]byte, error) {
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
