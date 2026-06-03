package adoption

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type InspectOptions struct {
	From    string
	AIAgent string
}

type ImportOptions struct {
	From    string
	Out     string
	AIAgent string
	Force   bool
}

type PackCreateOptions struct {
	From  string
	Out   string
	Force bool
}

type PackInstallOptions struct {
	Archive string
	Out     string
	Target  string
	Mode    string
	Force   bool
}

type Discovery struct {
	Runtime     string
	Confidence  string
	Commands    []Artifact
	Agents      []Artifact
	Skills      []Artifact
	MCPServers  []MCPServer
	Permissions map[string]string
	Warnings    []string
}

type Artifact struct {
	Name        string
	Path        string
	Description string
	Body        string
	Agent       string
}

type MCPServer struct {
	Name    string
	Type    string
	Command []string
	Env     map[string]any
	URL     string
	Headers map[string]any
	OAuth   any
	Timeout int
	Enabled *bool
	Tools   []string
}

type ImportResult struct {
	Out        string
	Runtime    string
	Confidence string
	Files      []string
}

type PackInfo struct {
	Name          string
	Version       string
	SourceRuntime string
	Objects       map[string]int
	Targets       []string
	Warnings      []string
}

type localState struct {
	DefaultTarget string `yaml:"defaultTarget"`
	SourcePack    string `yaml:"sourcePack,omitempty"`
}

func Inspect(opts InspectOptions) (Discovery, error) {
	from := defaultString(opts.From, ".")
	agent := defaultString(opts.AIAgent, "auto")
	if agent != "auto" && agent != "opencode" {
		return Discovery{}, fmt.Errorf("unsupported ai-agent %q", agent)
	}
	return inspectOpenCode(from, agent)
}

func Import(opts ImportOptions) (*ImportResult, error) {
	from := defaultString(opts.From, ".")
	out := defaultString(opts.Out, ".actlane")
	agent := defaultString(opts.AIAgent, "auto")
	discovery, err := Inspect(InspectOptions{From: from, AIAgent: agent})
	if err != nil {
		return nil, err
	}
	if discovery.Runtime == "" {
		return nil, fmt.Errorf("no supported ai-agent detected; try actlane import --ai-agent opencode")
	}
	if err := ensureWritableOutput(out, opts.Force); err != nil {
		return nil, err
	}
	files, err := writeImportedPack(from, out, discovery)
	if err != nil {
		return nil, err
	}
	return &ImportResult{Out: out, Runtime: discovery.Runtime, Confidence: discovery.Confidence, Files: files}, nil
}

func ReadImportReport(from string) ([]byte, error) {
	root := defaultString(from, ".actlane")
	return os.ReadFile(filepath.Join(root, "import.report.md"))
}

func CreatePack(opts PackCreateOptions) error {
	from := defaultString(opts.From, ".actlane")
	out := defaultString(opts.Out, "actlane-pack.zip")
	if !opts.Force {
		if _, err := os.Stat(out); err == nil {
			return fmt.Errorf("%s already exists; use --force to overwrite", out)
		}
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	tmp := out + ".tmp"
	if err := zipDir(from, tmp); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, out)
}

func InspectPack(archive string) (*PackInfo, error) {
	if archive == "" {
		archive = "actlane-pack.zip"
	}
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	info := &PackInfo{Objects: map[string]int{}}
	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, "/") {
			continue
		}
		data, err := readZipFile(file)
		if err != nil {
			return nil, err
		}
		if file.Name == "actlane.yaml" {
			var manifest struct {
				Metadata struct {
					Name    string `yaml:"name"`
					Version string `yaml:"version"`
				} `yaml:"metadata"`
				Spec struct {
					Targets []string `yaml:"targets"`
				} `yaml:"spec"`
			}
			if err := yaml.Unmarshal(data, &manifest); err != nil {
				return nil, fmt.Errorf("parse actlane.yaml: %w", err)
			}
			info.Name = manifest.Metadata.Name
			info.Version = manifest.Metadata.Version
			info.Targets = append(info.Targets, manifest.Spec.Targets...)
		}
		if file.Name == "import.report.md" {
			info.SourceRuntime = extractReportValue(string(data), "Detected runtime:")
			info.Warnings = extractReportList(string(data), "Warnings:")
		}
		var doc struct {
			Kind string `yaml:"kind"`
		}
		if yaml.Unmarshal(data, &doc) == nil && doc.Kind != "" {
			info.Objects[doc.Kind]++
		}
	}
	sort.Strings(info.Targets)
	return info, nil
}

func InstallPack(opts PackInstallOptions) error {
	if opts.Archive == "" {
		return fmt.Errorf("pack archive is required")
	}
	if opts.Target == "" {
		return fmt.Errorf("--target is required")
	}
	if opts.Mode == "" {
		opts.Mode = "overlay"
	}
	if opts.Mode != "overlay" {
		return fmt.Errorf("unsupported install mode %q", opts.Mode)
	}
	out := defaultString(opts.Out, ".actlane")
	if err := ensureWritableOutput(out, opts.Force); err != nil {
		return err
	}
	if err := unzipDir(opts.Archive, out); err != nil {
		return err
	}
	return WriteDefaultTarget(out, opts.Target, opts.Archive)
}

func ExtractPack(archive, out string) error {
	return unzipDir(archive, out)
}

func ReadDefaultTarget(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, ".local.yaml"))
	if err != nil {
		return "", err
	}
	var state localState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return "", err
	}
	if state.DefaultTarget == "" {
		return "", fmt.Errorf("defaultTarget is empty")
	}
	return state.DefaultTarget, nil
}

func WriteDefaultTarget(root, target, sourcePack string) error {
	data, err := yaml.Marshal(localState{DefaultTarget: target, SourcePack: sourcePack})
	if err != nil {
		return err
	}
	return writeFile(filepath.Join(root, ".local.yaml"), data)
}

func openCodeConfigPath(root string) string {
	for _, rel := range []string{"opencode.jsonc", ".opencode/opencode.jsonc"} {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if exists(path) {
			return path
		}
	}
	return ""
}

func inspectOpenCode(from, requested string) (Discovery, error) {
	root, err := filepath.Abs(from)
	if err != nil {
		return Discovery{}, err
	}
	discovery := Discovery{Runtime: "", Confidence: "none", Permissions: map[string]string{}}
	configPath := openCodeConfigPath(root)
	opencodeDir := filepath.Join(root, ".opencode")
	hasConfig := configPath != ""
	hasDir := isDir(opencodeDir)
	if !hasConfig && !hasDir {
		if requested == "opencode" {
			return discovery, fmt.Errorf("opencode project not found in %s", from)
		}
		return discovery, nil
	}
	discovery.Runtime = "opencode"
	discovery.Confidence = "medium"
	if hasConfig && hasDir {
		discovery.Confidence = "high"
	}
	discovery.Commands = readMarkdownArtifacts(root, ".opencode/commands")
	discovery.Agents = readMarkdownArtifacts(root, ".opencode/agents")
	discovery.Skills = readSkillArtifacts(root)
	if hasConfig {
		permissions, servers, warnings := readOpenCodeConfig(configPath)
		discovery.Permissions = permissions
		discovery.MCPServers = servers
		discovery.Warnings = append(discovery.Warnings, warnings...)
	}
	if len(discovery.Commands) == 0 && len(discovery.Agents) == 0 && len(discovery.Skills) == 0 {
		discovery.Warnings = append(discovery.Warnings, "No OpenCode commands, agents, or skills were found.")
	}
	return discovery, nil
}

func readMarkdownArtifacts(root, relDir string) []Artifact {
	dir := filepath.Join(root, filepath.FromSlash(relDir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var artifacts []Artifact
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		rel := path.Join(relDir, entry.Name())
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		meta, body := splitFrontmatter(string(data))
		name := strings.TrimSuffix(entry.Name(), ".md")
		artifacts = append(artifacts, Artifact{
			Name:        cleanName(defaultString(meta["name"], name)),
			Path:        rel,
			Description: defaultString(meta["description"], "Imported OpenCode artifact "+name+"."),
			Body:        strings.TrimSpace(body),
			Agent:       cleanName(meta["agent"]),
		})
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Name < artifacts[j].Name })
	return artifacts
}

func readSkillArtifacts(root string) []Artifact {
	dir := filepath.Join(root, ".opencode", "skills")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var artifacts []Artifact
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		rel := path.Join(".opencode/skills", entry.Name(), "SKILL.md")
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		meta, body := splitFrontmatter(string(data))
		name := cleanName(defaultString(meta["name"], entry.Name()))
		artifacts = append(artifacts, Artifact{
			Name:        name,
			Path:        rel,
			Description: defaultString(meta["description"], "Imported OpenCode skill "+name+"."),
			Body:        strings.TrimSpace(body),
		})
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Name < artifacts[j].Name })
	return artifacts
}

func readOpenCodeConfig(path string) (map[string]string, []MCPServer, []string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, []string{fmt.Sprintf("Cannot read OpenCode config: %v", err)}
	}
	var config map[string]any
	if err := json.Unmarshal(stripJSONC(data), &config); err != nil {
		return nil, nil, []string{"Cannot parse OpenCode config as JSONC; MCP and permissions were not imported."}
	}
	permissions := map[string]string{}
	if raw, ok := config["permission"].(map[string]any); ok {
		for key, value := range raw {
			permissions[key] = fmt.Sprint(value)
		}
	}
	var servers []MCPServer
	if raw, ok := config["mcp"].(map[string]any); ok {
		for key := range raw {
			server := MCPServer{Name: cleanName(key)}
			if serverConfig, ok := raw[key].(map[string]any); ok {
				server.Type, _ = serverConfig["type"].(string)
				server.Command = stringArray(serverConfig["command"])
				server.URL, _ = serverConfig["url"].(string)
				server.OAuth = serverConfig["oauth"]
				server.Timeout = intValue(serverConfig["timeout"])
				if enabled, ok := serverConfig["enabled"].(bool); ok {
					server.Enabled = &enabled
				}
				if env, ok := serverConfig["environment"].(map[string]any); ok {
					server.Env = env
				}
				if env, ok := serverConfig["env"].(map[string]any); ok && len(server.Env) == 0 {
					server.Env = env
				}
				if headers, ok := serverConfig["headers"].(map[string]any); ok {
					server.Headers = headers
				}
			}
			servers = append(servers, server)
		}
	}
	attachMCPTools(servers, permissions)
	sort.Slice(servers, func(i, j int) bool { return servers[i].Name < servers[j].Name })
	return permissions, servers, nil
}

func writeImportedPack(from, out string, d Discovery) ([]string, error) {
	capabilityName := inferredCapabilityName(d)
	skillName := inferredSkillName(d, capabilityName)
	commandName := inferredCommandName(d, capabilityName)
	agentName := inferredAgentName(d, capabilityName)
	policyName := capabilityName + "-policy"
	var written []string
	write := func(rel string, value any) error {
		data, err := marshalYAML(value)
		if err != nil {
			return err
		}
		if err := writeFile(filepath.Join(out, filepath.FromSlash(rel)), data); err != nil {
			return err
		}
		written = append(written, rel)
		return nil
	}

	if err := write("actlane.yaml", importedManifest(capabilityName, skillName, commandName, agentName, d)); err != nil {
		return nil, err
	}
	if err := write("capabilities/"+capabilityName+".yaml", importedCapability(capabilityName, skillName, commandName, agentName, policyName, d)); err != nil {
		return nil, err
	}
	if err := write("skills/"+skillName+".yaml", importedSkill(skillName, d)); err != nil {
		return nil, err
	}
	if commandName != "" {
		if err := write("commands/"+commandName+".yaml", importedCommand(commandName, capabilityName, skillName, agentName, d)); err != nil {
			return nil, err
		}
	}
	if agentName != "" {
		if err := write("agents/"+agentName+".yaml", importedAgent(agentName, capabilityName, skillName, d)); err != nil {
			return nil, err
		}
	}
	if err := write("policies/"+policyName+".yaml", importedPolicy(policyName, capabilityName)); err != nil {
		return nil, err
	}
	if len(d.MCPServers) > 0 {
		if err := write("mcp/bindings/"+capabilityName+".yaml", importedMCPBinding(capabilityName, d)); err != nil {
			return nil, err
		}
	}
	if err := write("target-profiles/opencode.yaml", importedOpenCodeTarget(d, skillName, commandName, agentName)); err != nil {
		return nil, err
	}
	if err := write("target-profiles/codex.yaml", importedCodexTarget(skillName)); err != nil {
		return nil, err
	}
	if err := writeFile(filepath.Join(out, "files", "AGENTS.md"), importedGuidance(from)); err != nil {
		return nil, err
	}
	written = append(written, "files/AGENTS.md")
	report := importReport(d, capabilityName, skillName, commandName, agentName)
	if err := writeFile(filepath.Join(out, "import.report.md"), []byte(report)); err != nil {
		return nil, err
	}
	written = append(written, "import.report.md")
	lock := importLock(d, written)
	if err := writeFile(filepath.Join(out, "actlane.lock"), lock); err != nil {
		return nil, err
	}
	written = append(written, "actlane.lock")
	sort.Strings(written)
	return written, nil
}

func importedManifest(capabilityName, skillName, commandName, agentName string, d Discovery) map[string]any {
	spec := map[string]any{
		"capabilities":   []string{"capabilities/" + capabilityName + ".yaml"},
		"skills":         []string{"skills/" + skillName + ".yaml"},
		"policies":       []string{"policies/" + capabilityName + "-policy.yaml"},
		"targets":        []string{"opencode", "codex"},
		"targetProfiles": []string{"target-profiles/opencode.yaml", "target-profiles/codex.yaml"},
		"guidance": map[string]any{
			"sources": []map[string]any{{"name": "imported-agents", "path": "files/AGENTS.md", "role": "project-guidance"}},
			"compose": map[string]any{"enabled": true, "output": "AGENTS.md", "strategy": "ordered-concat", "order": []string{"imported-agents"}},
		},
	}
	if commandName != "" {
		spec["commands"] = []string{"commands/" + commandName + ".yaml"}
	}
	if agentName != "" {
		spec["agents"] = []string{"agents/" + agentName + ".yaml"}
	}
	if len(d.MCPServers) > 0 {
		spec["mcpBindings"] = []string{"mcp/bindings/" + capabilityName + ".yaml"}
	}
	return doc("CapabilityPack", "imported-"+d.Runtime+"-pack", "0.3.0-alpha.8", "Imported "+d.Runtime+" project.", spec, importedAnnotations(d, "", false))
}

func importedCapability(name, skillName, commandName, agentName, policyName string, d Discovery) map[string]any {
	spec := map[string]any{
		"whenToUse": "Use this imported capability when the user asks for the matching OpenCode workflow.",
		"inputs":    map[string]any{"request": map[string]any{"type": "string", "required": true}},
		"outputs":   map[string]any{"summary": map[string]any{"type": "string"}},
		"policyRef": map[string]any{"name": policyName},
	}
	if len(d.MCPServers) > 0 {
		spec["executionRef"] = map[string]any{"name": name}
	}
	return doc("Capability", name, "", "Imported OpenCode capability.", spec, importedAnnotations(d, firstSource(d), true))
}

func importedSkill(name string, d Discovery) map[string]any {
	body := "Use the imported OpenCode workflow."
	desc := "Imported OpenCode skill."
	source := ""
	if len(d.Skills) > 0 {
		body = defaultString(d.Skills[0].Body, body)
		desc = d.Skills[0].Description
		source = d.Skills[0].Path
	} else if len(d.Commands) > 0 {
		body = defaultString(d.Commands[0].Body, body)
		desc = d.Commands[0].Description
		source = d.Commands[0].Path
	}
	return doc("SkillContract", name, "", desc, map[string]any{"body": body}, importedAnnotations(d, source, len(d.Skills) == 0))
}

func importedCommand(name, capabilityName, skillName, agentName string, d Discovery) map[string]any {
	body := "Use capability `" + capabilityName + "` with the user request: {{ arguments }}"
	desc := "Imported OpenCode command."
	source := ""
	if len(d.Commands) > 0 {
		body = defaultString(d.Commands[0].Body, body)
		desc = d.Commands[0].Description
		source = d.Commands[0].Path
	}
	spec := map[string]any{
		"scope":         "project",
		"invocation":    map[string]any{"slash": "/" + name},
		"capabilityRef": map[string]any{"name": capabilityName},
		"skillRef":      map[string]any{"path": ".opencode/skills/" + skillName + "/SKILL.md"},
		"arguments":     map[string]any{"mode": "passthrough", "placeholder": "{{ input }}", "description": "User request passed to the imported command."},
		"prompt":        map[string]any{"template": body},
	}
	if agentName != "" {
		spec["agentRef"] = map[string]any{"name": agentName, "optional": true}
	}
	return doc("CommandContract", name, "", desc, spec, importedAnnotations(d, source, false))
}

func importedAgent(name, capabilityName, skillName string, d Discovery) map[string]any {
	body := "Imported OpenCode agent."
	desc := body
	source := ""
	if len(d.Agents) > 0 {
		body = defaultString(d.Agents[0].Body, body)
		desc = d.Agents[0].Description
		source = d.Agents[0].Path
	}
	spec := map[string]any{
		"scope":        "project",
		"mode":         "subagent",
		"role":         map[string]any{"summary": body},
		"activation":   map[string]any{"whenToUse": []string{"Use when the imported OpenCode command delegates to this agent."}},
		"capabilities": map[string]any{"allowed": []string{capabilityName}},
		"skills":       map[string]any{"allowed": []string{skillName}},
	}
	return doc("AgentContract", name, "", desc, spec, importedAnnotations(d, source, false))
}

func importedPolicy(name, capabilityName string) map[string]any {
	spec := map[string]any{
		"match":    map[string]any{"capabilities": []string{capabilityName}},
		"approval": map[string]any{"required": false, "reason": "Imported policy is inferred and must be reviewed before high-risk use."},
		"audit":    map[string]any{"level": "summary", "include": []string{"tool", "decision", "reason"}},
	}
	return doc("ToolCallPolicy", name, "", "Inferred imported policy.", spec, map[string]string{"actlane.ru/inferred": "true"})
}

func importedMCPBinding(capabilityName string, d Discovery) map[string]any {
	var servers []map[string]any
	var requiredTools []map[string]any
	for _, name := range d.MCPServers {
		server := map[string]any{"name": name.Name, "provider": name.Name, "source": "opencode", "transport": defaultString(name.Type, "local")}
		if len(name.Command) > 0 {
			server["command"] = []string{name.Command[0]}
			if len(name.Command) > 1 {
				server["args"] = name.Command[1:]
			}
		}
		if name.URL != "" {
			server["url"] = name.URL
		}
		if len(name.Env) > 0 {
			server["env"] = name.Env
		}
		if len(name.Headers) > 0 {
			server["headers"] = name.Headers
		}
		if name.OAuth != nil {
			server["oauth"] = name.OAuth
		}
		if name.Timeout > 0 {
			server["timeout"] = name.Timeout
		}
		if name.Enabled != nil {
			server["enabled"] = *name.Enabled
		}
		servers = append(servers, server)
		for _, tool := range name.Tools {
			requiredTools = append(requiredTools, map[string]any{"name": tool, "server": name.Name, "toolset": "opencode-permission"})
		}
	}
	spec := map[string]any{
		"capabilityRef": map[string]any{"name": capabilityName},
		"mcpservers":    servers,
		"strategy":      map[string]any{"type": "imported", "handler": "review-required"},
	}
	if len(requiredTools) > 0 {
		spec["requiredTools"] = requiredTools
	}
	return doc("MCPBinding", capabilityName, "", "Inferred imported MCP binding.", spec, importedAnnotations(d, "opencode.jsonc", true))
}

func importedOpenCodeTarget(d Discovery, skillName, commandName, agentName string) map[string]any {
	permission := d.Permissions
	if len(permission) == 0 {
		permission = map[string]string{"bash": "ask", "edit": "ask", "skill": "allow"}
	}
	files := []map[string]any{
		{"targetPath": "AGENTS.md", "generatedPath": "generated/opencode/AGENTS.md", "ownedBlock": true},
		{"targetPath": "opencode.jsonc", "generatedPath": "generated/opencode/opencode.jsonc", "owned": true},
	}
	if commandName != "" {
		files = append(files, map[string]any{"targetPath": ".opencode/commands/" + commandName + ".md", "generatedPath": "generated/opencode/.opencode/commands/" + commandName + ".md", "commandContract": commandName, "owned": true})
	}
	if agentName != "" {
		files = append(files, map[string]any{"targetPath": ".opencode/agents/" + agentName + ".md", "generatedPath": "generated/opencode/.opencode/agents/" + agentName + ".md", "agentContract": agentName, "owned": true})
	}
	files = append(files, map[string]any{"targetPath": ".opencode/skills/" + skillName + "/SKILL.md", "generatedPath": "generated/opencode/.opencode/skills/" + skillName + "/SKILL.md", "skillContract": skillName, "owned": true})
	spec := map[string]any{
		"target": "opencode", "scope": "project",
		"output":  map[string]any{"root": "generated/opencode"},
		"install": map[string]any{"mode": "manual-copy", "scope": "project", "requireExplicitApply": true, "requireDiffPreview": true},
		"generate": map[string]any{
			"config": true, "instructions": true, "agents": true, "commands": true, "skills": true, "mcp": len(d.MCPServers) > 0,
		},
		"opencode": map[string]any{"config": map[string]any{"filename": "opencode.jsonc", "schema": "https://opencode.ai/config.json", "instructions": []string{"AGENTS.md"}, "permission": permission}, "files": files},
	}
	return doc("TargetProfile", "opencode", "", "OpenCode project-local target.", spec, importedAnnotations(d, "opencode.jsonc", false))
}

func importedCodexTarget(skillName string) map[string]any {
	spec := map[string]any{
		"target": "codex", "scope": "project",
		"output":  map[string]any{"root": "generated/codex", "config": "codex.config.toml"},
		"install": map[string]any{"mode": "manual-copy", "scope": "project", "requireExplicitApply": true, "requireDiffPreview": true},
		"generate": map[string]any{
			"config": true, "instructions": true, "agents": false, "commands": false, "skills": true, "mcp": false,
		},
		"codex": map[string]any{"config": map[string]any{"filename": "codex.config.toml"}, "files": []map[string]any{
			{"targetPath": "AGENTS.md", "generatedPath": "generated/codex/AGENTS.md", "ownedBlock": true},
			{"targetPath": ".codex/skills/" + skillName + "/SKILL.md", "generatedPath": "generated/codex/.codex/skills/" + skillName + "/SKILL.md", "skillContract": skillName, "owned": true},
		}},
	}
	return doc("TargetProfile", "codex", "", "Codex project-local target.", spec, nil)
}

func doc(kind, name, version, description string, spec map[string]any, annotations map[string]string) map[string]any {
	metadata := map[string]any{"name": name}
	if version != "" {
		metadata["version"] = version
	}
	if description != "" {
		metadata["description"] = description
	}
	if len(annotations) > 0 {
		metadata["annotations"] = annotations
	}
	return map[string]any{"apiVersion": "actlane.ru/v1alpha1", "kind": kind, "metadata": metadata, "spec": spec}
}

func importedAnnotations(d Discovery, source string, inferred bool) map[string]string {
	annotations := map[string]string{
		"actlane.ru/imported-from":     d.Runtime,
		"actlane.ru/import-confidence": d.Confidence,
	}
	if source != "" {
		annotations["actlane.ru/import-source"] = source
	}
	if inferred {
		annotations["actlane.ru/inferred"] = "true"
	}
	return annotations
}

func importedGuidance(from string) []byte {
	for _, rel := range []string{"AGENTS.md", ".opencode/AGENTS.md"} {
		data, err := os.ReadFile(filepath.Join(from, filepath.FromSlash(rel)))
		if err == nil && len(bytes.TrimSpace(data)) > 0 {
			return append(bytes.TrimRight(data, "\n"), '\n')
		}
	}
	return []byte("Imported OpenCode project guidance. Review this file before applying generated profiles.\n")
}

func importReport(d Discovery, capabilityName, skillName, commandName, agentName string) string {
	var b strings.Builder
	b.WriteString("# Actlane Import Report\n\n")
	b.WriteString("Detected runtime: " + d.Runtime + "\n")
	b.WriteString("Confidence: " + d.Confidence + "\n\n")
	b.WriteString("## Imported\n\n")
	writeReportItems(&b, "command", namesOr(commandName))
	writeReportItems(&b, "agent", namesOr(agentName))
	writeReportItems(&b, "skill", namesOr(skillName))
	writeReportItems(&b, "mcp server", mcpServerNames(d.MCPServers))
	b.WriteString("\n## Inferred\n\n")
	writeReportItems(&b, "capability", []string{capabilityName})
	writeReportItems(&b, "policy", []string{capabilityName + "-policy"})
	if len(d.MCPServers) > 0 {
		writeReportItems(&b, "mcp binding", []string{capabilityName})
	}
	b.WriteString("\nWarnings:\n")
	warnings := append([]string{}, d.Warnings...)
	if len(d.MCPServers) > 0 {
		warnings = append(warnings, "Capability, policy, and MCP binding were inferred and require review.")
	} else {
		warnings = append(warnings, "Capability and policy were inferred and require review.")
	}
	for _, warning := range warnings {
		b.WriteString("- " + warning + "\n")
	}
	return b.String()
}

func mcpServerNames(servers []MCPServer) []string {
	names := make([]string, 0, len(servers))
	for _, server := range servers {
		names = append(names, server.Name)
	}
	return names
}

func attachMCPTools(servers []MCPServer, permissions map[string]string) {
	for i := range servers {
		var tools []string
		prefix := servers[i].Name + "_"
		for key, decision := range permissions {
			if decision != "allow" {
				continue
			}
			if key == servers[i].Name || strings.HasPrefix(key, prefix) {
				tools = append(tools, key)
			}
		}
		sort.Strings(tools)
		servers[i].Tools = tools
	}
}

func importLock(d Discovery, files []string) []byte {
	value := map[string]any{"lockfileVersion": 1, "sourceRuntime": d.Runtime, "confidence": d.Confidence, "files": files}
	data, _ := json.MarshalIndent(value, "", "  ")
	return append(data, '\n')
}

func writeReportItems(b *strings.Builder, label string, values []string) {
	for _, value := range values {
		if value != "" {
			b.WriteString("- " + label + ": " + value + "\n")
		}
	}
}

func namesOr(value string) []string {
	if value == "" {
		return nil
	}
	return []string{value}
}

func inferredCapabilityName(d Discovery) string {
	if len(d.Commands) > 0 {
		return d.Commands[0].Name
	}
	if len(d.Skills) > 0 {
		return d.Skills[0].Name
	}
	if len(d.Agents) > 0 {
		return d.Agents[0].Name
	}
	return "imported-opencode-setup"
}

func inferredSkillName(d Discovery, fallback string) string {
	if len(d.Skills) > 0 {
		return d.Skills[0].Name
	}
	return fallback
}

func inferredCommandName(d Discovery, fallback string) string {
	if len(d.Commands) > 0 {
		return d.Commands[0].Name
	}
	return fallback
}

func inferredAgentName(d Discovery, fallback string) string {
	if len(d.Commands) > 0 && d.Commands[0].Agent != "" {
		return d.Commands[0].Agent
	}
	if len(d.Agents) > 0 {
		return d.Agents[0].Name
	}
	return fallback + "-agent"
}

func firstSource(d Discovery) string {
	if len(d.Commands) > 0 {
		return d.Commands[0].Path
	}
	if len(d.Skills) > 0 {
		return d.Skills[0].Path
	}
	if len(d.Agents) > 0 {
		return d.Agents[0].Path
	}
	return ""
}

func splitFrontmatter(content string) (map[string]string, string) {
	meta := map[string]string{}
	lines := strings.Split(content, "\n")
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "---" {
		return meta, content
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return meta, content
	}
	for _, line := range lines[1:end] {
		key, value, ok := strings.Cut(line, ":")
		if ok {
			meta[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), "\"'")
		}
	}
	return meta, strings.Join(lines[end+1:], "\n")
}

func stripJSONC(data []byte) []byte {
	var out bytes.Buffer
	inString := false
	escaped := false
	for i := 0; i < len(data); i++ {
		ch := data[i]
		if inString {
			out.WriteByte(ch)
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inString = false
			}
			continue
		}
		if ch == '"' {
			inString = true
			out.WriteByte(ch)
			continue
		}
		if ch == '/' && i+1 < len(data) && data[i+1] == '/' {
			for i < len(data) && data[i] != '\n' {
				i++
			}
			out.WriteByte('\n')
			continue
		}
		out.WriteByte(ch)
	}
	return out.Bytes()
}

func stringArray(value any) []string {
	switch typed := value.(type) {
	case string:
		if typed == "" {
			return nil
		}
		return []string{typed}
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if str, ok := item.(string); ok && str != "" {
				values = append(values, str)
			}
		}
		return values
	default:
		return nil
	}
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func cleanName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "imported-artifact"
	}
	return value
}

func marshalYAML(value any) ([]byte, error) {
	data, err := yaml.Marshal(value)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func ensureWritableOutput(out string, force bool) error {
	if !exists(out) {
		return nil
	}
	if !force && !dirEmpty(out) {
		return fmt.Errorf("%s already exists and is not empty; use --force to overwrite", out)
	}
	if force {
		return os.RemoveAll(out)
	}
	return nil
}

func zipDir(root, archive string) error {
	out, err := os.Create(archive)
	if err != nil {
		return err
	}
	defer out.Close()
	writer := zip.NewWriter(out)
	defer writer.Close()
	return filepath.WalkDir(root, func(filePath string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "generated" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, filePath)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == ".local.yaml" {
			return nil
		}
		header, err := zip.FileInfoHeader(mustInfo(entry))
		if err != nil {
			return err
		}
		header.Name = rel
		header.Method = zip.Deflate
		fileWriter, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}
		in, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(fileWriter, in)
		return err
	})
}

func unzipDir(archive, out string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer reader.Close()
	root, err := filepath.Abs(out)
	if err != nil {
		return err
	}
	for _, file := range reader.File {
		cleaned := path.Clean(file.Name)
		if cleaned == "." || strings.HasPrefix(cleaned, "../") || path.IsAbs(cleaned) {
			return fmt.Errorf("unsafe archive path %q", file.Name)
		}
		target := filepath.Join(root, filepath.FromSlash(cleaned))
		if !strings.HasPrefix(target, root+string(filepath.Separator)) && target != root {
			return fmt.Errorf("unsafe archive path %q", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := file.Open()
		if err != nil {
			return err
		}
		data, err := io.ReadAll(in)
		_ = in.Close()
		if err != nil {
			return err
		}
		if err := os.WriteFile(target, data, file.FileInfo().Mode()); err != nil {
			return err
		}
	}
	return nil
}

func readZipFile(file *zip.File) ([]byte, error) {
	in, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer in.Close()
	return io.ReadAll(in)
}

func extractReportValue(report, prefix string) string {
	for _, line := range strings.Split(report, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

func extractReportList(report, header string) []string {
	lines := strings.Split(report, "\n")
	var values []string
	inList := false
	for _, line := range lines {
		if strings.TrimSpace(line) == header {
			inList = true
			continue
		}
		if inList {
			if strings.HasPrefix(line, "- ") {
				values = append(values, strings.TrimPrefix(line, "- "))
				continue
			}
			if strings.TrimSpace(line) != "" {
				break
			}
		}
	}
	return values
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func dirEmpty(path string) bool {
	entries, err := os.ReadDir(path)
	return err == nil && len(entries) == 0
}

func mustInfo(entry os.DirEntry) os.FileInfo {
	info, err := entry.Info()
	if err != nil {
		panic(err)
	}
	return info
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
