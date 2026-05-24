package profile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/actlane/actlane/packages/cli/internal/pack"
)

type Options struct {
	Target         string
	OutDir         string
	Check          bool
	FrozenLockfile bool
}

type Result struct {
	Files map[string][]byte
}

func Generate(loaded *pack.LoadedPack, opts Options) (*Result, error) {
	if opts.OutDir == "" {
		opts.OutDir = loaded.Root
	}
	targetProfile, err := targetProfileFor(loaded, opts.Target)
	if err != nil {
		return nil, err
	}

	files, err := render(loaded, targetProfile, opts.Target)
	if err != nil {
		return nil, err
	}
	lockPath := lockfilePath(opts.Target)
	files[lockPath] = mustJSON(buildLockfile(loaded, files, opts.Target, lockPath))

	result := &Result{Files: files}
	if opts.Check || opts.FrozenLockfile {
		if err := compareExisting(opts.OutDir, files, opts.FrozenLockfile, lockPath); err != nil {
			return nil, err
		}
		return result, nil
	}
	if err := writeFiles(opts.OutDir, files); err != nil {
		return nil, err
	}
	return result, nil
}

func render(loaded *pack.LoadedPack, targetProfile pack.TargetProfile, target string) (map[string][]byte, error) {
	if len(loaded.Capabilities) != 1 {
		return nil, fmt.Errorf("MVP expects exactly one capability, got %d", len(loaded.Capabilities))
	}
	capability := loaded.Capabilities[0]
	rules := collectRules(loaded.Policies)
	renderer, err := rendererFor(targetProfile.Spec.Target)
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	if err := renderGuidance(files, loaded, targetProfile); err != nil {
		return nil, err
	}
	if err := renderer.Render(files, loaded, capability, targetProfile); err != nil {
		return nil, err
	}
	if err := renderCapabilityProfile(files, loaded, capability, targetProfile, target); err != nil {
		return nil, err
	}
	if targetProfile.Spec.Generate.MCP {
		renderMCPBindingArtifacts(files, loaded)
	}
	files[targetPolicyBundlePath(target)] = mustJSON(policyBundle{
		Pack:         loaded.Manifest.Metadata.Name,
		Version:      loaded.Manifest.Metadata.Version,
		Target:       target,
		Capabilities: []string{capability.Metadata.Name},
		Decisions:    []string{"allow", "deny", "mutate", "requires-approval"},
		Rules:        rules,
		MCPBindings:  policyBundleMCPBindings(loaded.MCPBindings),
	})
	return files, nil
}

func targetPolicyBundlePath(target string) string {
	return filepath.ToSlash(filepath.Join("generated", target, "policies", "policy-bundle.json"))
}

func renderGuidance(files map[string][]byte, loaded *pack.LoadedPack, targetProfile pack.TargetProfile) error {
	compose := loaded.Manifest.Spec.Guidance.Compose
	if !compose.Enabled {
		return nil
	}
	if compose.Output == "" {
		return fmt.Errorf("guidance compose output is required")
	}
	sources := map[string]pack.GuidanceSource{}
	for _, source := range loaded.Manifest.Spec.Guidance.Sources {
		sources[source.Name] = source
	}
	var b strings.Builder
	for i, name := range compose.Order {
		source, ok := sources[name]
		if !ok {
			return fmt.Errorf("guidance compose references missing source %s", name)
		}
		data, err := readPackSource(loaded.Root, source.Path)
		if err != nil {
			return fmt.Errorf("read guidance source %s: %w", source.Path, err)
		}
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.Write(data)
	}
	outputPath, err := targetOutputPath(targetProfile, compose.Output)
	if err != nil {
		return fmt.Errorf("invalid guidance compose output %q: %w", compose.Output, err)
	}
	files[filepath.ToSlash(outputPath)] = []byte(strings.TrimRight(b.String(), "\n") + "\n")
	return nil
}

func renderCapabilityProfile(files map[string][]byte, loaded *pack.LoadedPack, capability pack.Capability, targetProfile pack.TargetProfile, target string) error {
	profile, ok := capability.Spec.Profiles[target]
	if !ok {
		return nil
	}
	if profile.Config == nil {
		return fmt.Errorf("capability %s is missing spec.profiles.%s.config", capability.Metadata.Name, target)
	}
	if len(profile.Files) == 0 {
		return fmt.Errorf("capability %s is missing spec.profiles.%s.files", capability.Metadata.Name, target)
	}

	configPath, err := targetConfigPath(targetProfile)
	if err != nil {
		return err
	}
	if _, exists := files[configPath]; !exists {
		files[configPath] = mustJSON(cloneStringAnyMap(profile.Config))
	}
	for _, file := range profile.Files {
		if file.Source == "" && file.SkillContract == "" && file.CommandContract == "" && file.AgentContract == "" {
			return fmt.Errorf("capability %s profile file %q must declare source, skillContract, commandContract, or agentContract", capability.Metadata.Name, file.Path)
		}
		if countFileGenerators(file) != 1 {
			return fmt.Errorf("capability %s profile file %q must use exactly one generator", capability.Metadata.Name, file.Path)
		}
		if file.Content != "" {
			return fmt.Errorf("capability %s profile file %q must use source instead of inline content", capability.Metadata.Name, file.Path)
		}
		rel, err := targetOutputPath(targetProfile, file.Path)
		if err != nil {
			return fmt.Errorf("invalid generated profile file path %q: %w", file.Path, err)
		}
		if _, exists := files[rel]; exists {
			return fmt.Errorf("duplicate generated profile file path %q", file.Path)
		}
		if file.SkillContract != "" {
			content, err := renderSkillContract(loaded, file.SkillContract)
			if err != nil {
				return err
			}
			files[rel] = content
			if err := renderSkillContractResources(files, loaded, file.SkillContract, filepath.Dir(rel)); err != nil {
				return err
			}
			continue
		}
		if file.CommandContract != "" {
			content, err := renderCommandContract(loaded, target, file.CommandContract)
			if err != nil {
				return err
			}
			files[rel] = content
			continue
		}
		if file.AgentContract != "" {
			content, err := renderAgentContract(loaded, target, file.AgentContract)
			if err != nil {
				return err
			}
			files[rel] = content
			continue
		}
		content, err := readProfileSource(loaded.Root, capability.Path, file.Source)
		if err != nil {
			return err
		}
		files[rel] = content
	}
	return nil
}

func countFileGenerators(file pack.GeneratedFile) int {
	count := 0
	if file.Source != "" {
		count++
	}
	if file.SkillContract != "" {
		count++
	}
	if file.CommandContract != "" {
		count++
	}
	if file.AgentContract != "" {
		count++
	}
	return count
}

func renderSkillContract(loaded *pack.LoadedPack, name string) ([]byte, error) {
	skill, ok := skillContractFor(loaded, name)
	if !ok {
		return nil, fmt.Errorf("missing skill contract %s", name)
	}

	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("name: " + jsonString(skill.Metadata.Name) + "\n")
	b.WriteString("description: " + jsonString(skill.Metadata.Description) + "\n")
	b.WriteString("---\n\n")
	b.WriteString(strings.TrimRight(skill.Spec.Body, "\n"))
	b.WriteString("\n")

	return []byte(strings.TrimRight(b.String(), "\n") + "\n"), nil
}

func renderSkillContractResources(files map[string][]byte, loaded *pack.LoadedPack, name, skillDir string) error {
	skill, ok := skillContractFor(loaded, name)
	if !ok {
		return fmt.Errorf("missing skill contract %s", name)
	}
	for _, group := range []struct {
		dir       string
		resources []pack.SkillResource
	}{
		{dir: "scripts", resources: skill.Spec.Scripts},
		{dir: "references", resources: skill.Spec.References},
		{dir: "assets", resources: skill.Spec.Assets},
	} {
		for _, resource := range group.resources {
			if resource.Source == "" || resource.Path == "" {
				return fmt.Errorf("skill contract %s %s resource must declare source and path", name, group.dir)
			}
			relPath, err := cleanRelativePath(resource.Path)
			if err != nil {
				return fmt.Errorf("invalid skill contract %s %s path %q: %w", name, group.dir, resource.Path, err)
			}
			if !strings.HasPrefix(relPath, group.dir+"/") {
				return fmt.Errorf("skill contract %s %s resource path %q must be under %s/", name, group.dir, resource.Path, group.dir)
			}
			source, err := readSkillResourceSource(loaded.Root, skill.Path, resource.Source)
			if err != nil {
				return err
			}
			output := filepath.ToSlash(filepath.Join(skillDir, filepath.FromSlash(relPath)))
			if _, exists := files[output]; exists {
				return fmt.Errorf("duplicate generated skill resource path %q", output)
			}
			files[output] = source
		}
	}
	return nil
}

func skillContractFor(loaded *pack.LoadedPack, name string) (pack.SkillContract, bool) {
	for _, skill := range loaded.Skills {
		if skill.Metadata.Name == name {
			return skill, true
		}
	}
	return pack.SkillContract{}, false
}

func renderCommandContract(loaded *pack.LoadedPack, target, name string) ([]byte, error) {
	command, ok := commandContractFor(loaded, name)
	if !ok {
		return nil, fmt.Errorf("missing command contract %s", name)
	}
	projection, ok := command.Spec.Projections[target]
	if !ok || !projection.Enabled {
		return nil, fmt.Errorf("command contract %s missing enabled projection %s", name, target)
	}
	var b strings.Builder
	b.WriteString("---\n")
	if command.Spec.AgentRef.Name != "" {
		b.WriteString("agent: " + jsonString(command.Spec.AgentRef.Name) + "\n")
	}
	b.WriteString("description: " + jsonString(command.Metadata.Description) + "\n")
	b.WriteString("---\n\n")
	body := strings.ReplaceAll(command.Spec.Prompt.Template, "{{ arguments }}", command.Spec.Arguments.Placeholder)
	b.WriteString(strings.TrimRight(body, "\n"))
	b.WriteString("\n")
	return []byte(strings.TrimRight(b.String(), "\n") + "\n"), nil
}

func commandContractFor(loaded *pack.LoadedPack, name string) (pack.CommandContract, bool) {
	for _, command := range loaded.Commands {
		if command.Metadata.Name == name {
			return command, true
		}
	}
	return pack.CommandContract{}, false
}

func renderAgentContract(loaded *pack.LoadedPack, target, name string) ([]byte, error) {
	agent, ok := agentContractFor(loaded, name)
	if !ok {
		return nil, fmt.Errorf("missing agent contract %s", name)
	}
	projection, ok := agent.Spec.Projections[target]
	if !ok || !projection.Enabled {
		return nil, fmt.Errorf("agent contract %s missing enabled projection %s", name, target)
	}
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("description: " + jsonString(agent.Metadata.Description) + "\n")
	if agent.Spec.Mode != "" {
		b.WriteString("mode: " + agent.Spec.Mode + "\n")
	}
	if len(agent.Spec.Permissions) > 0 {
		b.WriteString("permission:\n")
		for _, key := range sortedStringMapKeys(agent.Spec.Permissions) {
			b.WriteString("  " + key + ": " + jsonString(agent.Spec.Permissions[key]) + "\n")
		}
	}
	b.WriteString("---\n\n")
	if agent.Spec.Role.Summary != "" {
		b.WriteString(agent.Spec.Role.Summary + "\n\n")
	}
	if len(agent.Spec.Activation.WhenToUse) > 0 {
		b.WriteString("When to use:\n\n")
		for _, item := range agent.Spec.Activation.WhenToUse {
			b.WriteString("- " + item + "\n")
		}
		b.WriteString("\n")
	}
	if len(agent.Spec.Capabilities.Allowed) > 0 || len(agent.Spec.Skills.Allowed) > 0 || agent.Spec.Tools.Strategy != "" || agent.Spec.Tools.RawMCPTools.Default != "" {
		b.WriteString("Operational rules:\n\n")
		for _, capability := range agent.Spec.Capabilities.Allowed {
			b.WriteString("- Use capability `" + capability + "`.\n")
		}
		for _, skill := range agent.Spec.Skills.Allowed {
			b.WriteString("- Use skill `" + skill + "`.\n")
		}
		if agent.Spec.Tools.Strategy != "" {
			b.WriteString("- Tool strategy: `" + agent.Spec.Tools.Strategy + "`.\n")
		}
		if agent.Spec.Tools.RawMCPTools.Default != "" {
			b.WriteString("- Raw MCP tools default: `" + agent.Spec.Tools.RawMCPTools.Default + "`.\n")
		}
		b.WriteString("\n")
	}
	if len(agent.Spec.Output.MustInclude) > 0 {
		b.WriteString("Output must include:\n\n")
		for _, item := range agent.Spec.Output.MustInclude {
			b.WriteString("- " + item + "\n")
		}
		b.WriteString("\n")
	}
	return []byte(strings.TrimRight(b.String(), "\n") + "\n"), nil
}

func agentContractFor(loaded *pack.LoadedPack, name string) (pack.AgentContract, bool) {
	for _, agent := range loaded.Agents {
		if agent.Metadata.Name == name {
			return agent, true
		}
	}
	return pack.AgentContract{}, false
}

func sortedStringMapKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func jsonString(value string) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func cloneStringAnyMap(source map[string]any) map[string]any {
	clone := make(map[string]any, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func readProfileSource(packRoot, capabilityPath, source string) ([]byte, error) {
	sourcePath, err := profileSourcePath(packRoot, capabilityPath, source)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("read profile source %s: %w", source, err)
	}
	return data, nil
}

func readSkillResourceSource(packRoot, skillPath, source string) ([]byte, error) {
	sourcePath, err := profileSourcePath(packRoot, skillPath, source)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("read skill resource source %s: %w", source, err)
	}
	return data, nil
}

func readPackSource(packRoot, source string) ([]byte, error) {
	cleaned, err := cleanRelativePath(source)
	if err != nil {
		return nil, fmt.Errorf("invalid pack source path %q: %w", source, err)
	}
	sourcePath := filepath.Join(packRoot, filepath.FromSlash(cleaned))
	if err := ensureInsideRoot(packRoot, sourcePath); err != nil {
		return nil, fmt.Errorf("invalid pack source path %q: %w", source, err)
	}
	if err := ensureExactPath(packRoot, sourcePath); err != nil {
		return nil, fmt.Errorf("invalid pack source path %q: %w", source, err)
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func profileSourcePath(packRoot, capabilityPath, source string) (string, error) {
	cleaned, err := cleanRelativePath(source)
	if err != nil {
		return "", fmt.Errorf("invalid profile source path %q: %w", source, err)
	}
	base := filepath.Dir(capabilityPath)
	sourcePath := filepath.Join(base, filepath.FromSlash(cleaned))
	if err := ensureInsideRoot(packRoot, sourcePath); err != nil {
		return "", fmt.Errorf("invalid profile source path %q: %w", source, err)
	}
	if err := ensureExactPath(packRoot, sourcePath); err != nil {
		return "", fmt.Errorf("invalid profile source path %q: %w", source, err)
	}
	return sourcePath, nil
}

func ensureInsideRoot(root, child string) error {
	rel, err := filepath.Rel(root, child)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path escapes pack root")
	}
	return nil
}

func ensureExactPath(root, child string) error {
	rel, err := filepath.Rel(root, child)
	if err != nil {
		return err
	}
	current := root
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == "." || part == "" {
			continue
		}
		entries, err := os.ReadDir(current)
		if err != nil {
			return err
		}
		found := false
		for _, entry := range entries {
			if entry.Name() == part {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("path component %q does not match filesystem casing", part)
		}
		current = filepath.Join(current, part)
	}
	return nil
}

func compareExisting(outDir string, files map[string][]byte, frozen bool, lockPath string) error {
	if frozen {
		existing, err := os.ReadFile(filepath.Join(outDir, lockPath))
		if err != nil {
			return fmt.Errorf("lockfile is missing or unreadable: %w", err)
		}
		if !bytes.Equal(existing, files[lockPath]) {
			return fmt.Errorf("lockfile would change")
		}
		return nil
	}
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, rel := range paths {
		existing, err := os.ReadFile(filepath.Join(outDir, rel))
		if err != nil {
			return fmt.Errorf("generated output is stale: %s is missing", rel)
		}
		if !bytes.Equal(existing, files[rel]) {
			return fmt.Errorf("generated output is stale: %s differs", rel)
		}
	}
	return nil
}

func writeFiles(outDir string, files map[string][]byte) error {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, rel := range paths {
		if err := writeAtomic(filepath.Join(outDir, rel), files[rel]); err != nil {
			return err
		}
	}
	return nil
}

func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func mustJSON(value any) []byte {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		panic(err)
	}
	return append(data, '\n')
}
