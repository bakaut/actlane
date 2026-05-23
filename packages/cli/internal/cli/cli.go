package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/actlane/actlane/packages/cli/internal/generator/profile"
	"github.com/actlane/actlane/packages/cli/internal/mcpserver"
	"github.com/actlane/actlane/packages/cli/internal/pack"
	"github.com/actlane/actlane/packages/cli/internal/schema"
)

const version = "0.1.0-alpha.1"

func Main(args []string, stdout, stderr io.Writer) int {
	return MainWithIO(args, os.Stdin, stdout, stderr)
}

func MainWithIO(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}

	switch args[0] {
	case "version":
		fmt.Fprintf(stdout, "actlane %s\n", version)
		return 0
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "generate":
		return runGenerate(args[1:], stdout, stderr)
	case "mcp":
		return runMCP(args[1:], stdin, stdout, stderr)
	case "schema":
		return runSchema(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		usage(stderr)
		return 2
	}
}

func runMCP(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) < 1 || args[0] != "serve" {
		fmt.Fprintln(stderr, "usage: actlane mcp serve (--policy-bundle <policy-bundle.json> | --pack <pack>)")
		return 2
	}
	packDir := "."
	policyBundlePath := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--pack":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--pack requires a value")
				return 2
			}
			packDir = args[i]
		case "--policy-bundle":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--policy-bundle requires a value")
				return 2
			}
			policyBundlePath = args[i]
		default:
			fmt.Fprintf(stderr, "unknown mcp serve flag %q\n", args[i])
			return 2
		}
	}
	if policyBundlePath != "" {
		data, err := os.ReadFile(policyBundlePath)
		if err != nil {
			fmt.Fprintf(stderr, "mcp serve failed: read policy bundle: %v\n", err)
			return 1
		}
		var bundle mcpserver.PolicyBundle
		if err := json.Unmarshal(data, &bundle); err != nil {
			fmt.Fprintf(stderr, "mcp serve failed: parse policy bundle: %v\n", err)
			return 1
		}
		if err := mcpserver.NewFromPolicyBundle(bundle).Serve(stdin, stdout); err != nil {
			fmt.Fprintf(stderr, "mcp serve failed: %v\n", err)
			return 1
		}
		return 0
	}
	loaded, err := pack.Load(packDir)
	if err != nil {
		fmt.Fprintf(stderr, "mcp serve failed: %v\n", err)
		return 1
	}
	if err := pack.Validate(loaded); err != nil {
		fmt.Fprintf(stderr, "mcp serve failed: %v\n", err)
		return 1
	}
	if err := mcpserver.New(loaded).Serve(stdin, stdout); err != nil {
		fmt.Fprintf(stderr, "mcp serve failed: %v\n", err)
		return 1
	}
	return 0
}

func runValidate(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: actlane validate <pack>")
		return 2
	}
	loaded, err := pack.Load(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "validate failed: %v\n", err)
		return 1
	}
	if err := pack.Validate(loaded); err != nil {
		fmt.Fprintf(stderr, "validate failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "valid: %s\n", loaded.Manifest.Metadata.Name)
	return 0
}

func runGenerate(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: actlane generate <pack> --target <target> [--out <dir>] [--check] [--frozen-lockfile]")
		return 2
	}

	packDir := args[0]
	opts := profile.Options{Target: "opencode", OutDir: packDir}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--target":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--target requires a value")
				return 2
			}
			opts.Target = args[i]
		case "--out":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--out requires a value")
				return 2
			}
			opts.OutDir = args[i]
			if !filepath.IsAbs(opts.OutDir) {
				opts.OutDir = filepath.Join(packDir, opts.OutDir)
			}
		case "--check":
			opts.Check = true
		case "--frozen-lockfile":
			opts.FrozenLockfile = true
		default:
			fmt.Fprintf(stderr, "unknown generate flag %q\n", args[i])
			return 2
		}
	}

	loaded, err := pack.Load(packDir)
	if err != nil {
		fmt.Fprintf(stderr, "generate failed: %v\n", err)
		return 1
	}
	if err := pack.Validate(loaded); err != nil {
		fmt.Fprintf(stderr, "generate failed: %v\n", err)
		return 1
	}
	result, err := profile.Generate(loaded, opts)
	if err != nil {
		fmt.Fprintf(stderr, "generate failed: %v\n", err)
		return 1
	}
	if opts.Check {
		fmt.Fprintf(stdout, "generated output is current: %d files\n", len(result.Files))
		return 0
	}
	if opts.FrozenLockfile {
		fmt.Fprintln(stdout, "lockfile is current")
		return 0
	}
	fmt.Fprintf(stdout, "generated %s target: %d files\n", opts.Target, len(result.Files))
	return 0
}

func runSchema(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && args[0] == "list" {
		fmt.Fprintln(stdout, "capability https://actlane.ru/schemas/v1alpha1/capability.schema.json")
		fmt.Fprintln(stdout, "capability-pack https://actlane.ru/schemas/v1alpha1/capability-pack.schema.json")
		fmt.Fprintln(stdout, "mcp-binding https://actlane.ru/schemas/v1alpha1/mcp-binding.schema.json")
		fmt.Fprintln(stdout, "skill-contract https://actlane.ru/schemas/v1alpha1/skill-contract.schema.json")
		fmt.Fprintln(stdout, "tool-call-policy https://actlane.ru/schemas/v1alpha1/tool-call-policy.schema.json")
		fmt.Fprintln(stdout, "target-profile https://actlane.ru/schemas/v1alpha1/target-profile.schema.json")
		fmt.Fprintln(stdout, "adoption-profile https://actlane.ru/schemas/v1alpha1/adoption-profile.schema.json")
		return 0
	}
	if len(args) == 2 && args[0] == "print" {
		content, err := schema.Read(args[1])
		if err != nil {
			fmt.Fprintf(stderr, "schema print failed: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, content)
		return 0
	}
	fmt.Fprintln(stderr, "usage: actlane schema list | actlane schema print capability")
	return 2
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "usage: actlane <version|validate|generate|mcp|schema>")
}
