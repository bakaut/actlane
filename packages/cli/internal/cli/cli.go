package cli

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/actlane/actlane/packages/cli/internal/generator/opencode"
	"github.com/actlane/actlane/packages/cli/internal/pack"
	"github.com/actlane/actlane/packages/cli/internal/schema"
)

const version = "0.1.0-alpha.1"

func Main(args []string, stdout, stderr io.Writer) int {
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
	case "schema":
		return runSchema(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		usage(stderr)
		return 2
	}
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
		fmt.Fprintln(stderr, "usage: actlane generate <pack> --target opencode [--out <dir>] [--check] [--frozen-lockfile]")
		return 2
	}

	packDir := args[0]
	opts := opencode.Options{Target: "opencode", OutDir: filepath.Join(packDir, "generated")}
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

	if opts.Target != "opencode" {
		fmt.Fprintf(stderr, "unsupported target %q; supported target: opencode\n", opts.Target)
		return 1
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
	result, err := opencode.Generate(loaded, opts)
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
	fmt.Fprintf(stdout, "generated opencode target: %d files\n", len(result.Files))
	return 0
}

func runSchema(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && args[0] == "list" {
		fmt.Fprintln(stdout, "capability https://actlane.ru/schemas/v1alpha1/capability.schema.json")
		fmt.Fprintln(stdout, "capability-pack https://actlane.ru/schemas/v1alpha1/capability-pack.schema.json")
		fmt.Fprintln(stdout, "tool-call-policy https://actlane.ru/schemas/v1alpha1/tool-call-policy.schema.json")
		fmt.Fprintln(stdout, "target-profile https://actlane.ru/schemas/v1alpha1/target-profile.schema.json")
		fmt.Fprintln(stdout, "adoption-profile https://actlane.ru/schemas/v1alpha1/adoption-profile.schema.json")
		return 0
	}
	if len(args) == 2 && args[0] == "print" && args[1] == "capability" {
		content, err := schema.Read("capability")
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
	fmt.Fprintln(w, "usage: actlane <version|validate|generate|schema>")
}
