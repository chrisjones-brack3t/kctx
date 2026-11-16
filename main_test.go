package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCLI runs the built program with a fake kubectl and checks its commands.
func TestCLI(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "kctx")
	coverage := filepath.Join(dir, "coverage")
	if err := os.Mkdir(coverage, 0o755); err != nil {
		t.Fatal(err)
	}
	build := exec.Command("go", "build", "-cover", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %s: %v", output, err)
	}

	fake := `#!/bin/sh
printf '%s\n' "$*" >> "$CALLS"
case "$*" in
  'config current-context')
    if [ "$FAIL" = current ]; then echo 'current failed' >&2; exit 1; fi
    printf '%s' "$CURRENT" ;;
  'config view --minify -o jsonpath={.contexts[0].context.namespace}')
    if [ "$FAIL" = view ]; then echo 'view failed' >&2; exit 1; fi
    printf '%s' "$NAMESPACE" ;;
  'config use-context '*)
    if [ "$FAIL" = use ]; then echo 'switch failed' >&2; exit 1; fi ;;
  'config set-context --current --namespace='*)
    if [ "$FAIL" = set ]; then echo 'set failed' >&2; exit 1; fi ;;
  *) echo 'unexpected command' >&2; exit 1 ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "kubectl"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name, current, namespace, fail, want, calls string
		args                                        []string
		failure                                     bool
	}{
		{name: "both", args: []string{"--context", "dev", "--namespace", "loon"}, want: "Context: dev-gke\nNamespace: loon", calls: "config use-context dev-gke\nconfig set-context --current --namespace=loon\n"},
		{name: "prod", args: []string{"--context", "prod", "--namespace", "books"}, want: "Context: prod-gke", calls: "config use-context prod-gke\nconfig set-context --current --namespace=books\n"},
		{name: "namespace only", args: []string{"--namespace", "loon"}, current: "prod-gke", want: "Namespace: loon", calls: "config current-context\nconfig use-context prod-gke\nconfig set-context --current --namespace=loon\n"},
		{name: "context only", args: []string{"--context", "prod"}, current: "dev-gke", namespace: "loon", want: "Namespace: loon", calls: "config current-context\nconfig view --minify -o jsonpath={.contexts[0].context.namespace}\nconfig use-context prod-gke\nconfig set-context --current --namespace=loon\n"},
		{name: "default namespace", args: []string{"--context", "dev"}, current: "prod-gke", want: "Namespace: default", calls: "config current-context\nconfig view --minify -o jsonpath={.contexts[0].context.namespace}\nconfig use-context dev-gke\nconfig set-context --current --namespace=default\n"},
		{name: "no flags", want: "provide --context", failure: true},
		{name: "positional argument", args: []string{"oops"}, want: "unexpected positional arguments", failure: true},
		{name: "empty namespace", args: []string{"--namespace="}, want: "namespace cannot be empty", failure: true},
		{name: "invalid context", args: []string{"--context", "other"}, want: "context must be dev or prod", failure: true},
		{name: "current failure", args: []string{"--namespace", "loon"}, fail: "current", want: "current failed", calls: "config current-context\n", failure: true},
		{name: "empty current", args: []string{"--namespace", "loon"}, want: "current context is empty", calls: "config current-context\n", failure: true},
		{name: "other current", args: []string{"--namespace", "loon"}, current: "other", want: "is not dev-gke or prod-gke", calls: "config current-context\n", failure: true},
		{name: "view failure", args: []string{"--context", "dev"}, current: "prod-gke", fail: "view", want: "view failed", calls: "config current-context\nconfig view --minify -o jsonpath={.contexts[0].context.namespace}\n", failure: true},
		{name: "switch failure", args: []string{"--context", "dev", "--namespace", "loon"}, fail: "use", want: "switch failed", calls: "config use-context dev-gke\n", failure: true},
		{name: "namespace failure", args: []string{"--context", "dev", "--namespace", "loon"}, fail: "set", want: "context may have changed to dev-gke", calls: "config use-context dev-gke\nconfig set-context --current --namespace=loon\n", failure: true},
		{name: "help", args: []string{"--help"}, want: "Usage of"},
		{name: "unknown flag", args: []string{"--unknown"}, want: "flag provided but not defined", failure: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := filepath.Join(t.TempDir(), "calls")
			cmd := exec.Command(binary, tc.args...)
			cmd.Env = append(os.Environ(), "PATH="+dir, "GOCOVERDIR="+coverage, "CALLS="+calls, "CURRENT="+tc.current, "NAMESPACE="+tc.namespace, "FAIL="+tc.fail)
			output, err := cmd.CombinedOutput()
			if (err != nil) != tc.failure {
				t.Fatalf("unexpected exit: %v, output: %s", err, output)
			}
			if !strings.Contains(string(output), tc.want) {
				t.Fatalf("output %q does not contain %q", output, tc.want)
			}
			got, err := os.ReadFile(calls)
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			if string(got) != tc.calls {
				t.Fatalf("commands = %q, want %q", got, tc.calls)
			}
		})
	}

	t.Run("missing kubectl", func(t *testing.T) {
		cmd := exec.Command(binary, "--context", "dev", "--namespace", "loon")
		cmd.Env = append(os.Environ(), "PATH="+t.TempDir(), "GOCOVERDIR="+coverage)
		output, err := cmd.CombinedOutput()
		if err == nil || !strings.Contains(string(output), "kubectl") {
			t.Fatalf("expected missing kubectl error, got %v: %s", err, output)
		}
	})

	report := exec.Command("go", "tool", "covdata", "percent", "-i="+coverage)
	output, err := report.CombinedOutput()
	if err != nil {
		t.Fatalf("coverage report failed: %v: %s", err, output)
	}
	if !strings.Contains(string(output), "100.0% of statements") {
		t.Errorf("expected 100%% coverage, got: %s", output)
	}
	t.Logf("integration coverage: %s", strings.TrimSpace(string(output)))
}
