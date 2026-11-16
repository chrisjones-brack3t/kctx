package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

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
	context := flag.String("context", "", "context to use: dev or prod")
	namespace := flag.String("namespace", "", "namespace to use")
	flag.Parse()

	provided := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { provided[f.Name] = true })
	if flag.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(flag.Args(), " "))
	}
	if !provided["context"] && !provided["namespace"] {
		return fmt.Errorf("provide --context, --namespace, or both")
	}
	if provided["namespace"] && *namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}

	target := ""
	if provided["context"] {
		switch *context {
		case "dev":
			target = "dev-gke"
		case "prod":
			target = "prod-gke"
		default:
			return fmt.Errorf("context must be dev or prod")
		}
	}

	// Read missing values before switching, while the old context is current.
	if !provided["context"] || !provided["namespace"] {
		current, err := kubectl("config", "current-context")
		if err != nil {
			return err
		}
		if current == "" {
			return fmt.Errorf("current context is empty")
		}
		if !provided["context"] {
			if current != "dev-gke" && current != "prod-gke" {
				return fmt.Errorf("current context %q is not dev-gke or prod-gke", current)
			}
			target = current
		}
		if !provided["namespace"] {
			value, err := kubectl("config", "view", "--minify", "-o", "jsonpath={.contexts[0].context.namespace}")
			if err != nil {
				return err
			}
			if value == "" {
				value = "default"
			}
			*namespace = value
		}
	}

	if _, err := kubectl("config", "use-context", target); err != nil {
		return err
	}
	if _, err := kubectl("config", "set-context", "--current", "--namespace="+*namespace); err != nil {
		return fmt.Errorf("context may have changed to %s; setting namespace failed: %w", target, err)
	}
	fmt.Printf("Context: %s\nNamespace: %s\n", target, *namespace)
	return nil
}

// main reports errors and exits with a nonzero status when the update fails.
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "kctx:", err)
		os.Exit(1)
	}
}
