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
	build := exec.Command("go", "build", "-buildvcs=false", "-cover", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %s: %v", output, err)
	}

	fake := `#!/bin/sh
printf '%s\n' "$*" >> "$CALLS"
case "$*" in
  'config current-context')
    if [ "$FAIL" = current ]; then echo 'current failed' >&2; exit 1; fi
    printf '%s' "$CURRENT" ;;
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
		name, current, fail, want, calls string
		args                             []string
		failure                          bool
	}{
		{name: "both long", args: []string{"--context", "dev-gke", "--namespace", "loon"}, want: "Context: dev-gke\nNamespace: loon", calls: "config use-context dev-gke\nconfig set-context --current --namespace=loon\n"},
		{name: "both short", args: []string{"-c", "my-cluster", "-n", "books"}, want: "Context: my-cluster\nNamespace: books", calls: "config use-context my-cluster\nconfig set-context --current --namespace=books\n"},
		{name: "mixed flags", args: []string{"-c", "dev", "--namespace", "loon"}, want: "Context: dev", calls: "config use-context dev\nconfig set-context --current --namespace=loon\n"},
		{name: "namespace only", args: []string{"-n", "loon"}, current: "other", want: "Context: other\nNamespace: loon", calls: "config current-context\nconfig set-context --current --namespace=loon\n"},
		{name: "context only", args: []string{"-c", "prod-gke"}, want: "Context: prod-gke", calls: "config use-context prod-gke\n"},
		{name: "no flags", want: "provide --context", failure: true},
		{name: "positional argument", args: []string{"oops"}, want: "unexpected positional arguments", failure: true},
		{name: "empty namespace", args: []string{"--namespace="}, want: "namespace cannot be empty", failure: true},
		{name: "empty context", args: []string{"-c="}, want: "context cannot be empty", failure: true},
		{name: "duplicate context", args: []string{"-c", "a", "--context", "b"}, want: "use either --context or -c", failure: true},
		{name: "duplicate namespace", args: []string{"-n", "a", "--namespace", "b"}, want: "use either --namespace or -n", failure: true},
		{name: "current failure", args: []string{"--namespace", "loon"}, fail: "current", want: "current failed", calls: "config current-context\n", failure: true},
		{name: "empty current", args: []string{"--namespace", "loon"}, want: "current context is empty", calls: "config current-context\n", failure: true},
		{name: "switch failure", args: []string{"--context", "dev-gke", "--namespace", "loon"}, fail: "use", want: "switch failed", calls: "config use-context dev-gke\n", failure: true},
		{name: "namespace failure", args: []string{"--context", "dev-gke", "--namespace", "loon"}, fail: "set", want: "context may have changed to dev-gke", calls: "config use-context dev-gke\nconfig set-context --current --namespace=loon\n", failure: true},
		{name: "namespace-only failure", args: []string{"-n", "loon"}, current: "other", fail: "set", want: "set failed", calls: "config current-context\nconfig set-context --current --namespace=loon\n", failure: true},
		{name: "version", args: []string{"--version"}, want: "dev\n"},
		{name: "help", args: []string{"--help"}, want: "Usage of"},
		{name: "unknown flag", args: []string{"--unknown"}, want: "flag provided but not defined", failure: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := filepath.Join(t.TempDir(), "calls")
			cmd := exec.Command(binary, tc.args...)
			cmd.Env = append(os.Environ(), "PATH="+dir, "GOCOVERDIR="+coverage, "CALLS="+calls, "CURRENT="+tc.current, "FAIL="+tc.fail)
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

	t.Run("release version", func(t *testing.T) {
		release := filepath.Join(t.TempDir(), "kctx")
		build := exec.Command("go", "build", "-cover", "-ldflags", "-X main.version=v1.2.3", "-o", release, ".")
		if output, err := build.CombinedOutput(); err != nil {
			t.Fatalf("release build failed: %v: %s", err, output)
		}
		cmd := exec.Command(release, "--version")
		cmd.Env = append(os.Environ(), "PATH="+t.TempDir(), "GOCOVERDIR="+coverage)
		output, err := cmd.CombinedOutput()
		if err != nil || string(output) != "v1.2.3\n" {
			t.Fatalf("version = %q, error = %v; want v1.2.3", output, err)
		}
	})

	t.Run("module version", func(t *testing.T) {
		root, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		moduleDir := t.TempDir()
		module := "module example.com/versiontest\n\ngo 1.21\n\nrequire github.com/chrisjones-brack3t/kctx v1.2.3\n\nreplace github.com/chrisjones-brack3t/kctx => " + filepath.ToSlash(root) + "\n"
		if err := os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte(module), 0o644); err != nil {
			t.Fatal(err)
		}
		binary := filepath.Join(moduleDir, "kctx")
		build := exec.Command("go", "build", "-cover", "-o", binary, "github.com/chrisjones-brack3t/kctx")
		build.Dir = moduleDir
		if output, err := build.CombinedOutput(); err != nil {
			t.Fatalf("module build failed: %v: %s", err, output)
		}
		cmd := exec.Command(binary, "--version")
		cmd.Env = append(os.Environ(), "PATH="+t.TempDir(), "GOCOVERDIR="+coverage)
		output, err := cmd.CombinedOutput()
		if err != nil || string(output) != "v1.2.3\n" {
			t.Fatalf("version = %q, error = %v; want v1.2.3", output, err)
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
