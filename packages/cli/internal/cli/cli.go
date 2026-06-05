package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/actlane/actlane/packages/cli/internal/adoption"
	"github.com/actlane/actlane/packages/cli/internal/authoringmcp"
	"github.com/actlane/actlane/packages/cli/internal/evaluator"
	"github.com/actlane/actlane/packages/cli/internal/generator/profile"
	"github.com/actlane/actlane/packages/cli/internal/mcpserver"
	"github.com/actlane/actlane/packages/cli/internal/pack"
	"github.com/actlane/actlane/packages/cli/internal/scaffold"
	"github.com/actlane/actlane/packages/cli/internal/schema"
)

const version = "0.3.0-alpha.14"

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
	case "inspect":
		return runInspect(args[1:], stdout, stderr)
	case "import":
		return runImport(args[1:], stdout, stderr)
	case "pack":
		return runPack(args[1:], stdout, stderr)
	case "generate":
		return runGenerate(args[1:], stdout, stderr)
	case "plan":
		return runPlan(args[1:], stdout, stderr)
	case "apply":
		return runApply(args[1:], stdout, stderr)
	case "remove":
		return runRemove(args[1:], stdout, stderr)
	case "check":
		return runCheck(args[1:], stdin, stdout, stderr)
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

func runInspect(args []string, stdout, stderr io.Writer) int {
	opts := adoption.InspectOptions{From: ".", AIAgent: "auto"}
	for i := 0; i < len(args); i++ {
		switch {
		case strings.HasPrefix(args[i], "--from="):
			opts.From = strings.TrimPrefix(args[i], "--from=")
		case strings.HasPrefix(args[i], "--ai-agent="):
			opts.AIAgent = strings.TrimPrefix(args[i], "--ai-agent=")
		case args[i] == "--from":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--from requires a value")
				return 2
			}
			opts.From = args[i]
		case args[i] == "--ai-agent":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--ai-agent requires a value")
				return 2
			}
			opts.AIAgent = args[i]
		default:
			fmt.Fprintf(stderr, "unknown inspect flag %q\n", args[i])
			return 2
		}
	}
	discovery, err := adoption.Inspect(opts)
	if err != nil {
		fmt.Fprintf(stderr, "inspect failed: %v\n", err)
		return 1
	}
	if discovery.Runtime == "" {
		fmt.Fprintln(stdout, "No supported ai-agent detected.")
		fmt.Fprintln(stdout, "Try: actlane inspect --ai-agent opencode or --ai-agent codex")
		return 0
	}
	if discovery.Runtime == "codex" {
		printCodexInspection(stdout, discovery)
		return 0
	}
	fmt.Fprintln(stdout, "Detected:")
	fmt.Fprintf(stdout, "- ai-agent: %s\n", discovery.Runtime)
	fmt.Fprintf(stdout, "- confidence: %s\n", discovery.Confidence)
	for _, command := range discovery.Commands {
		fmt.Fprintf(stdout, "- command: %s\n", command.Name)
	}
	for _, agent := range discovery.Agents {
		fmt.Fprintf(stdout, "- agent: %s\n", agent.Name)
	}
	for _, skill := range discovery.Skills {
		fmt.Fprintf(stdout, "- skill: %s\n", skill.Name)
	}
	for _, server := range discovery.MCPServers {
		fmt.Fprintf(stdout, "- mcp server: %s\n", server.Name)
		for _, tool := range server.Tools {
			fmt.Fprintf(stdout, "- mcp tool: %s/%s\n", server.Name, tool)
		}
	}
	for key, value := range discovery.Permissions {
		fmt.Fprintf(stdout, "- permission %s=%s\n", key, value)
	}
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Next:")
	fmt.Fprintln(stdout, "  actlane import")
	return 0
}

func printCodexInspection(stdout io.Writer, discovery adoption.Discovery) {
	fmt.Fprintln(stdout, "Detected:")
	fmt.Fprintln(stdout, "- ai-agent: codex")
	fmt.Fprintf(stdout, "- confidence: %s\n\n", discovery.Confidence)
	fmt.Fprintln(stdout, "Project-local:")
	for _, guidance := range discovery.Agents {
		fmt.Fprintf(stdout, "- guidance: %s\n", filepath.Base(guidance.Path))
	}
	for _, skill := range discovery.Skills {
		fmt.Fprintf(stdout, "- skill: %s\n", skill.Name)
	}
	for _, server := range discovery.MCPServers {
		fmt.Fprintf(stdout, "- mcp: %s\n", server.Name)
	}
	fmt.Fprintln(stdout, "\nAvailable global objects:")
	for _, skill := range discovery.GlobalSkills {
		fmt.Fprintf(stdout, "- skill: %s [%s]\n", skill.Name, skill.Portability)
	}
	for _, server := range discovery.GlobalMCPServers {
		fmt.Fprintf(stdout, "- mcp: %s [%s]\n", server.Name, server.Portability)
		if server.Reason != "" && strings.Contains(strings.ToLower(server.Reason), "absolute") {
			fmt.Fprintf(stdout, "  reason: %s\n", server.Reason)
		}
	}
	for _, hook := range discovery.GlobalHooks {
		fmt.Fprintf(stdout, "- hook: %s [%s]\n", hook.Name, hook.Portability)
	}
	fmt.Fprintln(stdout, "\nGlobal configuration has lower migration accuracy.")
	fmt.Fprintln(stdout, "Safe candidates:")
	fmt.Fprintln(stdout, "- Global skills without external dependencies.")
	fmt.Fprintln(stdout, "Import with caution:")
	fmt.Fprintln(stdout, "- MCP servers may contain local paths and machine-specific commands.")
	fmt.Fprintln(stdout, "- MCP environment variable values are never transferred.")
	fmt.Fprintln(stdout, "Not imported:")
	fmt.Fprintln(stdout, "- Hooks, credentials, auth, sessions, history, trust state, logs, caches, and SQLite state.")
	fmt.Fprintln(stdout, "Recommendation:")
	fmt.Fprintln(stdout, "- Review and migrate global configuration manually when possible.")
	fmt.Fprintln(stdout, "\nNext:")
	fmt.Fprintln(stdout, "  actlane import --ai-agent codex")
	for _, skill := range discovery.GlobalSkills {
		fmt.Fprintf(stdout, "  actlane import --ai-agent codex --include-global-skill %s\n", skill.Name)
	}
	for _, server := range discovery.GlobalMCPServers {
		fmt.Fprintf(stdout, "  actlane import --ai-agent codex --include-global-mcp %s\n", server.Name)
	}
}

func runImport(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "report" {
		return runImportReport(args[1:], stdout, stderr)
	}
	opts := adoption.ImportOptions{From: ".", Out: ".actlane", AIAgent: "auto"}
	for i := 0; i < len(args); i++ {
		switch {
		case strings.HasPrefix(args[i], "--from="):
			opts.From = strings.TrimPrefix(args[i], "--from=")
		case strings.HasPrefix(args[i], "--out="):
			opts.Out = strings.TrimPrefix(args[i], "--out=")
		case strings.HasPrefix(args[i], "--ai-agent="):
			opts.AIAgent = strings.TrimPrefix(args[i], "--ai-agent=")
		case strings.HasPrefix(args[i], "--include-global-skill="):
			opts.IncludeGlobalSkills = append(opts.IncludeGlobalSkills, strings.TrimPrefix(args[i], "--include-global-skill="))
		case strings.HasPrefix(args[i], "--include-global-mcp="):
			opts.IncludeGlobalMCP = append(opts.IncludeGlobalMCP, strings.TrimPrefix(args[i], "--include-global-mcp="))
		case args[i] == "opencode" || args[i] == "codex":
			opts.AIAgent = args[i]
		case args[i] == "--from":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--from requires a value")
				return 2
			}
			opts.From = args[i]
		case args[i] == "--out":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--out requires a value")
				return 2
			}
			opts.Out = args[i]
		case args[i] == "--ai-agent":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--ai-agent requires a value")
				return 2
			}
			opts.AIAgent = args[i]
		case args[i] == "--include-global-skill":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--include-global-skill requires a value")
				return 2
			}
			opts.IncludeGlobalSkills = append(opts.IncludeGlobalSkills, args[i])
		case args[i] == "--include-global-mcp":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--include-global-mcp requires a value")
				return 2
			}
			opts.IncludeGlobalMCP = append(opts.IncludeGlobalMCP, args[i])
		case args[i] == "--force":
			opts.Force = true
		default:
			fmt.Fprintf(stderr, "unknown import flag %q\n", args[i])
			return 2
		}
	}
	result, err := adoption.Import(opts)
	if err != nil {
		fmt.Fprintf(stderr, "import failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Imported %s project into %s\n", result.Runtime, result.Out)
	fmt.Fprintf(stdout, "Generated source files: %d\n", len(result.Files))
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Next:")
	fmt.Fprintln(stdout, "  actlane pack create")
	return 0
}

func runImportReport(args []string, stdout, stderr io.Writer) int {
	from := ".actlane"
	for i := 0; i < len(args); i++ {
		switch {
		case strings.HasPrefix(args[i], "--from="):
			from = strings.TrimPrefix(args[i], "--from=")
		case args[i] == "--from":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--from requires a value")
				return 2
			}
			from = args[i]
		default:
			fmt.Fprintf(stderr, "unknown import report flag %q\n", args[i])
			return 2
		}
	}
	data, err := adoption.ReadImportReport(from)
	if err != nil {
		fmt.Fprintf(stderr, "import report failed: %v\n", err)
		return 1
	}
	fmt.Fprint(stdout, string(data))
	return 0
}

func runPack(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: actlane pack <init|create|inspect|install>")
		return 2
	}
	switch args[0] {
	case "init":
		return runPackInit(args[1:], stdout, stderr)
	case "create":
		return runPackCreate(args[1:], stdout, stderr)
	case "inspect":
		return runPackInspect(args[1:], stdout, stderr)
	case "install":
		return runPackInstall(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown pack command %q\n", args[0])
		return 2
	}
}

func runPackInit(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintln(stdout, packInitUsage())
		return 0
	}
	if len(args) == 0 || isFlag(args[0]) {
		fmt.Fprintln(stderr, packInitUsage())
		return 2
	}
	name := scaffold.CleanName(args[0])
	opts := struct {
		Out       string
		Targets   []string
		Contracts []string
		Force     bool
	}{
		Out:     filepath.Join("packs", name),
		Targets: []string{"codex"},
	}
	for i := 1; i < len(args); i++ {
		switch {
		case strings.HasPrefix(args[i], "--out="):
			opts.Out = strings.TrimPrefix(args[i], "--out=")
		case strings.HasPrefix(args[i], "--targets="):
			opts.Targets = splitTargets(strings.TrimPrefix(args[i], "--targets="))
		case strings.HasPrefix(args[i], "--target="):
			opts.Targets = []string{strings.TrimPrefix(args[i], "--target=")}
		case strings.HasPrefix(args[i], "--contracts="):
			opts.Contracts = splitTargets(strings.TrimPrefix(args[i], "--contracts="))
		case args[i] == "--out":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--out requires a value")
				return 2
			}
			opts.Out = args[i]
		case args[i] == "--targets":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--targets requires a value")
				return 2
			}
			opts.Targets = splitTargets(args[i])
		case args[i] == "--target":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--target requires a value")
				return 2
			}
			opts.Targets = []string{args[i]}
		case args[i] == "--contracts":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--contracts requires a value")
				return 2
			}
			opts.Contracts = splitTargets(args[i])
		case args[i] == "--force":
			opts.Force = true
		default:
			fmt.Fprintf(stderr, "unknown pack init flag %q\n", args[i])
			return 2
		}
	}
	files, err := scaffold.Plan(scaffold.Options{Name: name, Targets: opts.Targets, Contracts: opts.Contracts})
	if err != nil {
		fmt.Fprintf(stderr, "pack init failed: %v\n", err)
		return 2
	}
	written, skipped, err := scaffold.Write(opts.Out, files, opts.Force)
	if len(skipped) > 0 {
		fmt.Fprintf(stderr, "pack init failed: existing files: %s\n", strings.Join(skipped, ", "))
		return 1
	}
	if err != nil {
		fmt.Fprintf(stderr, "pack init failed: %v\n", err)
		return 1
	}
	loaded, err := pack.Load(opts.Out)
	if err != nil {
		fmt.Fprintf(stderr, "pack init failed: %v\n", err)
		return 1
	}
	if err := pack.Validate(loaded); err != nil {
		fmt.Fprintf(stderr, "pack init failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "initialized pack: %s\n", opts.Out)
	fmt.Fprintln(stdout, "Written:")
	for _, path := range written {
		fmt.Fprintf(stdout, "- %s\n", path)
	}
	fmt.Fprintln(stdout, "Next:")
	fmt.Fprintf(stdout, "  actlane generate %s --target %s\n", opts.Out, opts.Targets[0])
	fmt.Fprintf(stdout, "  actlane plan %s --target %s --project .\n", opts.Out, opts.Targets[0])
	return 0
}

func packInitUsage() string {
	return "usage: actlane pack init <name> [--out <dir>] [--targets codex] [--contracts capability,policy,mcp,skill,target-profile] [--force]\ncontracts: default or comma-list; all includes command,agent,responsibility,runtime-profile,evidence"
}

func runPackCreate(args []string, stdout, stderr io.Writer) int {
	opts := adoption.PackCreateOptions{From: ".actlane", Out: "actlane-pack.zip"}
	for i := 0; i < len(args); i++ {
		switch {
		case strings.HasPrefix(args[i], "--from="):
			opts.From = strings.TrimPrefix(args[i], "--from=")
		case strings.HasPrefix(args[i], "--out="):
			opts.Out = strings.TrimPrefix(args[i], "--out=")
		case args[i] == "--from":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--from requires a value")
				return 2
			}
			opts.From = args[i]
		case args[i] == "--out":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--out requires a value")
				return 2
			}
			opts.Out = args[i]
		case args[i] == "--force":
			opts.Force = true
		default:
			fmt.Fprintf(stderr, "unknown pack create flag %q\n", args[i])
			return 2
		}
	}
	loaded, err := pack.Load(opts.From)
	if err != nil {
		fmt.Fprintf(stderr, "pack create failed: %v\n", err)
		return 1
	}
	if err := pack.Validate(loaded); err != nil {
		fmt.Fprintf(stderr, "pack create failed: %v\n", err)
		return 1
	}
	if err := adoption.CreatePack(opts); err != nil {
		fmt.Fprintf(stderr, "pack create failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "created pack: %s\n", opts.Out)
	return 0
}

func runPackInspect(args []string, stdout, stderr io.Writer) int {
	archive := "actlane-pack.zip"
	if len(args) > 1 {
		fmt.Fprintln(stderr, "usage: actlane pack inspect [actlane-pack.zip]")
		return 2
	}
	if len(args) == 1 {
		archive = args[0]
	}
	info, err := adoption.InspectPack(archive)
	if err != nil {
		fmt.Fprintf(stderr, "pack inspect failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Pack: %s\n", info.Name)
	if info.Version != "" {
		fmt.Fprintf(stdout, "Version: %s\n", info.Version)
	}
	if info.SourceRuntime != "" {
		fmt.Fprintf(stdout, "Source runtime: %s\n", info.SourceRuntime)
	}
	fmt.Fprintln(stdout, "Objects:")
	for _, kind := range sortedObjectKinds(info.Objects) {
		count := info.Objects[kind]
		fmt.Fprintf(stdout, "- %s: %d\n", kind, count)
	}
	fmt.Fprintln(stdout, "Targets:")
	for _, target := range info.Targets {
		fmt.Fprintf(stdout, "- %s\n", target)
	}
	if len(info.Warnings) > 0 {
		fmt.Fprintln(stdout, "Warnings:")
		for _, warning := range info.Warnings {
			fmt.Fprintf(stdout, "- %s\n", warning)
		}
	}
	return 0
}

func runPackInstall(args []string, stdout, stderr io.Writer) int {
	opts := adoption.PackInstallOptions{Out: ".actlane", Mode: "overlay"}
	if len(args) > 0 && !isFlag(args[0]) {
		opts.Archive = args[0]
		args = args[1:]
	}
	for i := 0; i < len(args); i++ {
		switch {
		case strings.HasPrefix(args[i], "--target="):
			opts.Target = strings.TrimPrefix(args[i], "--target=")
		case strings.HasPrefix(args[i], "--mode="):
			opts.Mode = strings.TrimPrefix(args[i], "--mode=")
		case strings.HasPrefix(args[i], "--out="):
			opts.Out = strings.TrimPrefix(args[i], "--out=")
		case args[i] == "--target":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--target requires a value")
				return 2
			}
			opts.Target = args[i]
		case args[i] == "--mode":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--mode requires a value")
				return 2
			}
			opts.Mode = args[i]
		case args[i] == "--out":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--out requires a value")
				return 2
			}
			opts.Out = args[i]
		case args[i] == "--force":
			opts.Force = true
		default:
			fmt.Fprintf(stderr, "unknown pack install flag %q\n", args[i])
			return 2
		}
	}
	if opts.Archive == "" {
		opts.Archive = "actlane-pack.zip"
	}
	if err := adoption.InstallPack(opts); err != nil {
		fmt.Fprintf(stderr, "pack install failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "installed pack into %s\n", opts.Out)
	fmt.Fprintf(stdout, "default target: %s\n", opts.Target)
	return 0
}

func runMCP(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) >= 2 && (args[0] == "author" || args[0] == "pack-author") {
		return runMCPAuthor(args[1:], stdin, stdout, stderr)
	}
	if len(args) < 1 || args[0] != "serve" {
		fmt.Fprintln(stderr, "usage: actlane mcp serve (--broker-bundle <broker-bundle.json> | --policy-bundle <policy-bundle.json> | --pack <pack>)")
		fmt.Fprintln(stderr, "usage: actlane mcp author serve [--pack <pack>]")
		return 2
	}
	packDir := "."
	policyBundlePath := ""
	brokerBundlePath := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--pack":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--pack requires a value")
				return 2
			}
			packDir = args[i]
		case "--broker-bundle":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--broker-bundle requires a value")
				return 2
			}
			brokerBundlePath = args[i]
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
	if brokerBundlePath != "" {
		data, err := os.ReadFile(brokerBundlePath)
		if err != nil {
			fmt.Fprintf(stderr, "mcp serve failed: read broker bundle: %v\n", err)
			return 1
		}
		var bundle pack.BrokerBundle
		if err := json.Unmarshal(data, &bundle); err != nil {
			fmt.Fprintf(stderr, "mcp serve failed: parse broker bundle: %v\n", err)
			return 1
		}
		if err := mcpserver.NewFromBrokerBundle(bundle).Serve(stdin, stdout); err != nil {
			fmt.Fprintf(stderr, "mcp serve failed: %v\n", err)
			return 1
		}
		return 0
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

func runMCPAuthor(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) < 1 || args[0] != "serve" {
		fmt.Fprintln(stderr, "usage: actlane mcp author serve [--pack <pack>]")
		return 2
	}
	packDir := "."
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--pack":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--pack requires a value")
				return 2
			}
			packDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown mcp author serve flag %q\n", args[i])
			return 2
		}
	}
	if err := authoringmcp.New(packDir).Serve(stdin, stdout); err != nil {
		fmt.Fprintf(stderr, "mcp author serve failed: %v\n", err)
		return 1
	}
	return 0
}

func runCheck(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	packDir := "."
	policyBundlePath := ""
	inputPath := ""
	capabilityName := ""
	toolName := "actlane.check"
	mode := "audit"
	for i := 0; i < len(args); i++ {
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
		case "--input":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--input requires a value")
				return 2
			}
			inputPath = args[i]
		case "--capability":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--capability requires a value")
				return 2
			}
			capabilityName = args[i]
		case "--tool":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--tool requires a value")
				return 2
			}
			toolName = args[i]
		case "--mode":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--mode requires a value")
				return 2
			}
			mode = args[i]
		default:
			fmt.Fprintf(stderr, "unknown check flag %q\n", args[i])
			return 2
		}
	}

	loaded, err := loadedForCheck(packDir, policyBundlePath)
	if err != nil {
		fmt.Fprintf(stderr, "check failed: %v\n", err)
		return 1
	}
	if capabilityName == "" {
		if len(loaded.Capabilities) > 0 {
			capabilityName = loaded.Capabilities[0].Metadata.Name
		} else if len(loaded.Contracts) > 0 {
			capabilityName = loaded.Contracts[0].Metadata.Name
		}
	}
	if capabilityName == "" {
		fmt.Fprintln(stderr, "check failed: capability is required")
		return 1
	}
	input, err := readCheckInput(stdin, inputPath)
	if err != nil {
		fmt.Fprintf(stderr, "check failed: %v\n", err)
		return 1
	}
	eval := evaluator.Evaluate(loaded, evaluator.Request{
		Tool:       toolName,
		Mode:       mode,
		Capability: capabilityName,
		Input:      input,
	})
	data, err := json.MarshalIndent(eval, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "check failed: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, string(data))
	if mode == "enforce" && !eval.Allowed {
		return 1
	}
	return 0
}

func loadedForCheck(packDir, policyBundlePath string) (*pack.LoadedPack, error) {
	if policyBundlePath != "" {
		data, err := os.ReadFile(policyBundlePath)
		if err != nil {
			return nil, fmt.Errorf("read policy bundle: %w", err)
		}
		var bundle mcpserver.PolicyBundle
		if err := json.Unmarshal(data, &bundle); err != nil {
			return nil, fmt.Errorf("parse policy bundle: %w", err)
		}
		return mcpserver.LoadedFromPolicyBundle(bundle), nil
	}
	loaded, err := pack.Load(packDir)
	if err != nil {
		return nil, err
	}
	if err := pack.Validate(loaded); err != nil {
		return nil, err
	}
	return loaded, nil
}

func readCheckInput(stdin io.Reader, inputPath string) (map[string]any, error) {
	var data []byte
	var err error
	if inputPath != "" {
		data, err = os.ReadFile(inputPath)
	} else {
		data, err = io.ReadAll(stdin)
	}
	if err != nil {
		return nil, err
	}
	input := map[string]any{}
	if len(strings.TrimSpace(string(data))) == 0 {
		return input, nil
	}
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("parse input JSON: %w", err)
	}
	return input, nil
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
	packDir := ".actlane"
	packArgExplicit := false
	if len(args) > 0 && !isFlag(args[0]) {
		packDir = args[0]
		packArgExplicit = true
		args = args[1:]
	}
	outExplicit := false
	outDir := ""
	outBase := packDir
	opts := profile.Options{}
	for i := 0; i < len(args); i++ {
		switch {
		case strings.HasPrefix(args[i], "--target="):
			opts.Target = strings.TrimPrefix(args[i], "--target=")
		case strings.HasPrefix(args[i], "--out="):
			outDir = strings.TrimPrefix(args[i], "--out=")
			outExplicit = true
		case args[i] == "--target":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--target requires a value")
				return 2
			}
			opts.Target = args[i]
		case args[i] == "--out":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--out requires a value")
				return 2
			}
			outDir = args[i]
			outExplicit = true
		case args[i] == "--check":
			opts.Check = true
		case args[i] == "--frozen-lockfile":
			opts.FrozenLockfile = true
		default:
			fmt.Fprintf(stderr, "unknown generate flag %q\n", args[i])
			return 2
		}
	}
	cleanup := func() {}
	if shouldUsePackArchive(packDir, packArgExplicit) {
		archive := packDir
		if !packArgExplicit {
			archive = "actlane-pack.zip"
		}
		tempDir, err := os.MkdirTemp("", "actlane-pack-*")
		if err != nil {
			fmt.Fprintf(stderr, "generate failed: %v\n", err)
			return 1
		}
		cleanup = func() { _ = os.RemoveAll(tempDir) }
		defer cleanup()
		if err := adoption.ExtractPack(archive, tempDir); err != nil {
			fmt.Fprintf(stderr, "generate failed: read pack archive: %v\n", err)
			return 1
		}
		packDir = tempDir
		outBase = "."
		if !outExplicit {
			outDir = "."
		}
	}
	if outDir == "" {
		outDir = packDir
	}
	opts.OutDir = outDir
	if !filepath.IsAbs(opts.OutDir) && opts.OutDir != "." {
		opts.OutDir = filepath.Join(outBase, opts.OutDir)
	}
	if opts.Target == "" {
		target, err := adoption.ReadDefaultTarget(packDir)
		if err == nil {
			opts.Target = target
		}
	}
	if opts.Target == "" {
		fmt.Fprintln(stderr, "generate failed: --target is required when no default target exists")
		return 2
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

func runPlan(args []string, stdout, stderr io.Writer) int {
	packDir := ".actlane"
	packArgExplicit := false
	if len(args) > 0 && !isFlag(args[0]) {
		packDir = args[0]
		packArgExplicit = true
		args = args[1:]
	}
	opts := adoption.PlanOptions{Project: "."}
	for i := 0; i < len(args); i++ {
		switch {
		case strings.HasPrefix(args[i], "--target="):
			opts.Target = strings.TrimPrefix(args[i], "--target=")
		case strings.HasPrefix(args[i], "--from="):
			opts.From = strings.TrimPrefix(args[i], "--from=")
		case strings.HasPrefix(args[i], "--project="):
			opts.Project = strings.TrimPrefix(args[i], "--project=")
		case args[i] == "--target":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--target requires a value")
				return 2
			}
			opts.Target = args[i]
		case args[i] == "--from":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--from requires a value")
				return 2
			}
			opts.From = args[i]
		case args[i] == "--project":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--project requires a value")
				return 2
			}
			opts.Project = args[i]
		case args[i] == "--json":
			opts.JSON = true
		case args[i] == "--diff":
			opts.Diff = true
		case args[i] == "--show-content":
			opts.ShowContent = true
		default:
			fmt.Fprintf(stderr, "unknown plan flag %q\n", args[i])
			return 2
		}
	}
	cleanup := func() {}
	if shouldUsePackArchive(packDir, packArgExplicit) {
		archive := packDir
		if !packArgExplicit {
			archive = "actlane-pack.zip"
		}
		tempDir, err := os.MkdirTemp("", "actlane-pack-*")
		if err != nil {
			fmt.Fprintf(stderr, "plan failed: %v\n", err)
			return 1
		}
		cleanup = func() { _ = os.RemoveAll(tempDir) }
		defer cleanup()
		if err := adoption.ExtractPack(archive, tempDir); err != nil {
			fmt.Fprintf(stderr, "plan failed: read pack archive: %v\n", err)
			return 1
		}
		packDir = tempDir
	}
	loaded, err := pack.Load(packDir)
	if err != nil {
		fmt.Fprintf(stderr, "plan failed: %v\n", err)
		return 1
	}
	if err := pack.Validate(loaded); err != nil {
		fmt.Fprintf(stderr, "plan failed: %v\n", err)
		return 1
	}
	if opts.Target == "" {
		fmt.Fprintln(stderr, "plan failed: --target is required")
		return 2
	}
	plan, err := adoption.BuildPlan(loaded, opts)
	if err != nil {
		fmt.Fprintf(stderr, "plan failed: %v\n", err)
		return 1
	}
	if opts.JSON {
		data, err := adoption.FormatPlanJSON(plan, opts)
		if err != nil {
			fmt.Fprintf(stderr, "plan failed: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, string(data))
		return 0
	}
	fmt.Fprint(stdout, adoption.FormatPlanText(plan, opts))
	return 0
}

func runApply(args []string, stdout, stderr io.Writer) int {
	packDir := ".actlane"
	packArgExplicit := false
	if len(args) > 0 && !isFlag(args[0]) {
		packDir = args[0]
		packArgExplicit = true
		args = args[1:]
	}
	opts := adoption.ApplyOptions{PlanOptions: adoption.PlanOptions{Project: "."}}
	for i := 0; i < len(args); i++ {
		switch {
		case strings.HasPrefix(args[i], "--target="):
			opts.Target = strings.TrimPrefix(args[i], "--target=")
		case strings.HasPrefix(args[i], "--from="):
			opts.From = strings.TrimPrefix(args[i], "--from=")
		case strings.HasPrefix(args[i], "--project="):
			opts.Project = strings.TrimPrefix(args[i], "--project=")
		case args[i] == "--target":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--target requires a value")
				return 2
			}
			opts.Target = args[i]
		case args[i] == "--from":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--from requires a value")
				return 2
			}
			opts.From = args[i]
		case args[i] == "--project":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--project requires a value")
				return 2
			}
			opts.Project = args[i]
		case args[i] == "--dry-run":
			opts.DryRun = true
		case args[i] == "--json":
			opts.JSON = true
		default:
			fmt.Fprintf(stderr, "unknown apply flag %q\n", args[i])
			return 2
		}
	}
	if opts.Target == "" {
		fmt.Fprintln(stderr, "apply failed: --target is required")
		return 2
	}
	cleanup := func() {}
	if shouldUsePackArchive(packDir, packArgExplicit) {
		archive := packDir
		if !packArgExplicit {
			archive = "actlane-pack.zip"
		}
		tempDir, err := os.MkdirTemp("", "actlane-pack-*")
		if err != nil {
			fmt.Fprintf(stderr, "apply failed: %v\n", err)
			return 1
		}
		cleanup = func() { _ = os.RemoveAll(tempDir) }
		defer cleanup()
		if err := adoption.ExtractPack(archive, tempDir); err != nil {
			fmt.Fprintf(stderr, "apply failed: read pack archive: %v\n", err)
			return 1
		}
		packDir = tempDir
	}
	loaded, err := pack.Load(packDir)
	if err != nil {
		fmt.Fprintf(stderr, "apply failed: %v\n", err)
		return 1
	}
	if err := pack.Validate(loaded); err != nil {
		fmt.Fprintf(stderr, "apply failed: %v\n", err)
		return 1
	}
	plan, err := adoption.BuildPlan(loaded, opts.PlanOptions)
	if err != nil {
		fmt.Fprintf(stderr, "apply failed: %v\n", err)
		return 1
	}
	result, err := adoption.ApplyPlan(plan, opts)
	if opts.JSON {
		data, jsonErr := adoption.FormatApplyJSON(result)
		if jsonErr != nil {
			fmt.Fprintf(stderr, "apply failed: %v\n", jsonErr)
			return 1
		}
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprint(stdout, adoption.FormatApplyText(result))
	}
	if err != nil {
		fmt.Fprintf(stderr, "apply failed: %v\n", err)
		return 1
	}
	return 0
}

func runRemove(args []string, stdout, stderr io.Writer) int {
	packDir := ".actlane"
	packArgExplicit := false
	if len(args) > 0 && !isFlag(args[0]) {
		packDir = args[0]
		packArgExplicit = true
		args = args[1:]
	}
	opts := adoption.RemoveOptions{PlanOptions: adoption.PlanOptions{Project: "."}}
	for i := 0; i < len(args); i++ {
		switch {
		case strings.HasPrefix(args[i], "--target="):
			opts.Target = strings.TrimPrefix(args[i], "--target=")
		case strings.HasPrefix(args[i], "--from="):
			opts.From = strings.TrimPrefix(args[i], "--from=")
		case strings.HasPrefix(args[i], "--project="):
			opts.Project = strings.TrimPrefix(args[i], "--project=")
		case args[i] == "--target":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--target requires a value")
				return 2
			}
			opts.Target = args[i]
		case args[i] == "--from":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--from requires a value")
				return 2
			}
			opts.From = args[i]
		case args[i] == "--project":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "--project requires a value")
				return 2
			}
			opts.Project = args[i]
		case args[i] == "--dry-run":
			opts.DryRun = true
		case args[i] == "--json":
			opts.JSON = true
		default:
			fmt.Fprintf(stderr, "unknown remove flag %q\n", args[i])
			return 2
		}
	}
	if opts.Target == "" {
		fmt.Fprintln(stderr, "remove failed: --target is required")
		return 2
	}
	cleanup := func() {}
	if shouldUsePackArchive(packDir, packArgExplicit) {
		archive := packDir
		if !packArgExplicit {
			archive = "actlane-pack.zip"
		}
		tempDir, err := os.MkdirTemp("", "actlane-pack-*")
		if err != nil {
			fmt.Fprintf(stderr, "remove failed: %v\n", err)
			return 1
		}
		cleanup = func() { _ = os.RemoveAll(tempDir) }
		defer cleanup()
		if err := adoption.ExtractPack(archive, tempDir); err != nil {
			fmt.Fprintf(stderr, "remove failed: read pack archive: %v\n", err)
			return 1
		}
		packDir = tempDir
	}
	loaded, err := pack.Load(packDir)
	if err != nil {
		fmt.Fprintf(stderr, "remove failed: %v\n", err)
		return 1
	}
	if err := pack.Validate(loaded); err != nil {
		fmt.Fprintf(stderr, "remove failed: %v\n", err)
		return 1
	}
	result, err := adoption.RemovePlan(loaded, opts)
	if opts.JSON {
		data, jsonErr := adoption.FormatRemoveJSON(result)
		if jsonErr != nil {
			fmt.Fprintf(stderr, "remove failed: %v\n", jsonErr)
			return 1
		}
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprint(stdout, adoption.FormatRemoveText(result))
	}
	if err != nil {
		fmt.Fprintf(stderr, "remove failed: %v\n", err)
		return 1
	}
	return 0
}

func runSchema(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && args[0] == "list" {
		fmt.Fprintln(stdout, "agent-contract https://actlane.ru/schemas/v1alpha1/agent-contract.schema.json")
		fmt.Fprintln(stdout, "capability https://actlane.ru/schemas/v1alpha1/capability.schema.json")
		fmt.Fprintln(stdout, "capability-pack https://actlane.ru/schemas/v1alpha1/capability-pack.schema.json")
		fmt.Fprintln(stdout, "command-contract https://actlane.ru/schemas/v1alpha1/command-contract.schema.json")
		fmt.Fprintln(stdout, "evidence-contract https://actlane.ru/schemas/v1alpha1/evidence-contract.schema.json")
		fmt.Fprintln(stdout, "mcp-binding https://actlane.ru/schemas/v1alpha1/mcp-binding.schema.json")
		fmt.Fprintln(stdout, "runtime-profile https://actlane.ru/schemas/v1alpha1/runtime-profile.schema.json")
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
	fmt.Fprintln(w, "usage: actlane <version|inspect|import|pack|validate|generate|plan|apply|remove|check|mcp|schema>")
}

func isFlag(value string) bool {
	return len(value) > 0 && value[0] == '-'
}

func shouldUsePackArchive(packDir string, explicit bool) bool {
	if explicit {
		info, err := os.Stat(packDir)
		return err == nil && !info.IsDir()
	}
	if _, err := os.Stat(filepath.Join(packDir, "actlane.yaml")); err == nil {
		return false
	}
	info, err := os.Stat("actlane-pack.zip")
	return err == nil && !info.IsDir()
}

func splitTargets(value string) []string {
	var targets []string
	for _, part := range strings.Split(value, ",") {
		target := strings.TrimSpace(part)
		if target != "" {
			targets = append(targets, target)
		}
	}
	if len(targets) == 0 {
		return []string{"codex"}
	}
	return targets
}

func sortedObjectKinds(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
