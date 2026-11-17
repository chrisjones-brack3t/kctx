package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
)

// version is set to the release tag when building release binaries.
var version string

// buildVersion returns the release tag or the version recorded by Go.
func buildVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

// kubectl runs a kubectl command and returns its output or an error.
func kubectl(args ...string) (string, error) {
	output, err := exec.Command("kubectl", args...).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message != "" {
			return "", fmt.Errorf("kubectl %s: %w: %s", strings.Join(args, " "), err, message)
		}
		return "", fmt.Errorf("kubectl %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(output)), nil
}

// run reads the flags and updates the selected context and namespace.
func run() error {
	context := flag.String("context", "", "context to use")
	flag.StringVar(context, "c", "", "context to use (short form)")
	namespace := flag.String("namespace", "", "namespace to use")
	flag.StringVar(namespace, "n", "", "namespace to use (short form)")
	showVersion := flag.Bool("version", false, "show the version")
	flag.Parse()
	if *showVersion {
		fmt.Println(buildVersion())
		return nil
	}

	provided := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { provided[f.Name] = true })
	if flag.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(flag.Args(), " "))
	}
	if provided["context"] && provided["c"] {
		return fmt.Errorf("use either --context or -c, not both")
	}
	if provided["namespace"] && provided["n"] {
		return fmt.Errorf("use either --namespace or -n, not both")
	}
	hasContext := provided["context"] || provided["c"]
	hasNamespace := provided["namespace"] || provided["n"]
	if !hasContext && !hasNamespace {
		return fmt.Errorf("provide --context, --namespace, or both")
	}
	if hasContext && *context == "" {
		return fmt.Errorf("context cannot be empty")
	}
	if hasNamespace && *namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}

	if hasContext {
		if _, err := kubectl("config", "use-context", *context); err != nil {
			return err
		}
	} else {
		current, err := kubectl("config", "current-context")
		if err != nil {
			return err
		}
		if current == "" {
			return fmt.Errorf("current context is empty")
		}
		*context = current
	}

	// A context-only switch keeps the target's existing namespace.
	if hasNamespace {
		if _, err := kubectl("config", "set-context", "--current", "--namespace="+*namespace); err != nil {
			if hasContext {
				return fmt.Errorf("context may have changed to %s; setting namespace failed: %w", *context, err)
			}
			return err
		}
		fmt.Printf("Context: %s\nNamespace: %s\n", *context, *namespace)
	} else {
		fmt.Printf("Context: %s\n", *context)
	}
	return nil
}

// main reports errors and exits with a nonzero status when the update fails.
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "kctx:", err)
		os.Exit(1)
	}
}
