package profile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/actlane/actlane/packages/cli/internal/pack"
)

const generatorVersion = "actlane-go-profile-0.1.0-alpha.1"

type Options struct {
	Target         string
	OutDir         string
	Check          bool
	FrozenLockfile bool
}

type Result struct {
	Files map[string][]byte
}

type policyBundle struct {
	Pack         string         `json:"pack"`
	Version      string         `json:"version"`
	Target       string         `json:"target"`
	Capabilities []string       `json:"capabilities"`
	Decisions    []string       `json:"decisions"`
	Rules        map[string]any `json:"rules"`
}

type lockfile struct {
	LockfileVersion int                   `json:"lockfileVersion"`
	Pack            string                `json:"pack"`
	Version         string                `json:"version"`
	Target          string                `json:"target"`
	Generator       string                `json:"generator"`
	SourceDigests   map[string]string     `json:"sourceDigests"`
	GeneratedFiles  []generatedFileRecord `json:"generatedFiles"`
	Metadata        map[string]string     `json:"metadata"`
}

type generatedFileRecord struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

func Generate(loaded *pack.LoadedPack, opts Options) (*Result, error) {
	if opts.OutDir == "" {
		opts.OutDir = filepath.Join(loaded.Root, "generated")
	}
	targetProfile, err := targetProfileFor(loaded, opts.Target)
	if err != nil {
		return nil, err
	}

	files, err := render(loaded, targetProfile, opts.Target)
	if err != nil {
		return nil, err
	}
	files["actlane.lock"] = mustJSON(buildLockfile(loaded, files, opts.Target))

	result := &Result{Files: files}
	if opts.Check || opts.FrozenLockfile {
		if err := compareExisting(opts.OutDir, files, opts.FrozenLockfile); err != nil {
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
	files := map[string][]byte{}
	if err := renderCapabilityProfile(files, loaded.Root, capability, targetProfile, target); err != nil {
		return nil, err
	}
	if len(capability.Spec.MCP.Tools) > 0 {
		files["mcp/tools.json"] = mustJSON(mcpToolsDocument(capability))
	}
	if len(capability.Spec.MCP.Servers) > 0 {
		files["mcp/server.json"] = mustJSON(mcpServerDocument(capability))
	}
	files["policies/policy-bundle.json"] = mustJSON(policyBundle{
		Pack:         loaded.Manifest.Metadata.Name,
		Version:      loaded.Manifest.Metadata.Version,
		Target:       target,
		Capabilities: []string{capability.Metadata.Name},
		Decisions:    []string{"allow", "deny", "mutate", "requires-approval"},
		Rules:        rules,
	})
	return files, nil
}

func targetProfileFor(loaded *pack.LoadedPack, target string) (pack.TargetProfile, error) {
	for _, targetProfile := range loaded.TargetProfiles {
		if targetProfile.Spec.Target == target {
			return targetProfile, nil
		}
	}
	return pack.TargetProfile{}, fmt.Errorf("unsupported target %q; supported targets: %s", target, strings.Join(supportedTargets(loaded), ", "))
}

func supportedTargets(loaded *pack.LoadedPack) []string {
	targets := make([]string, 0, len(loaded.TargetProfiles))
	for _, targetProfile := range loaded.TargetProfiles {
		targets = append(targets, targetProfile.Spec.Target)
	}
	sort.Strings(targets)
	return targets
}

func renderCapabilityProfile(files map[string][]byte, packRoot string, capability pack.Capability, targetProfile pack.TargetProfile, target string) error {
	profile, ok := capability.Spec.Profiles[target]
	if !ok {
		return fmt.Errorf("capability %s is missing spec.profiles.%s", capability.Metadata.Name, target)
	}
	if profile.Config == nil {
		return fmt.Errorf("capability %s is missing spec.profiles.%s.config", capability.Metadata.Name, target)
	}
	if len(profile.Files) == 0 {
		return fmt.Errorf("capability %s is missing spec.profiles.%s.files", capability.Metadata.Name, target)
	}

	config := cloneStringAnyMap(profile.Config)
	if key := targetProfile.Spec.Transforms.MCP.ConfigKey; targetProfile.Spec.Transforms.MCP.Enabled && key != "" {
		if _, exists := config[key]; !exists && len(capability.Spec.MCP.Servers) > 0 {
			config[key] = mcpConfig(capability)
		}
	}
	configPath, err := targetOutputPath(targetProfile, targetProfile.Spec.Output.Config)
	if err != nil {
		return fmt.Errorf("invalid target profile config path %q: %w", targetProfile.Spec.Output.Config, err)
	}
	files[configPath] = mustJSON(config)
	for _, file := range profile.Files {
		if file.Source == "" {
			return fmt.Errorf("capability %s profile file %q must declare source", capability.Metadata.Name, file.Path)
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
		content, err := readProfileSource(packRoot, capability.Path, file.Source)
		if err != nil {
			return err
		}
		files[rel] = content
	}
	return nil
}

func cloneStringAnyMap(source map[string]any) map[string]any {
	clone := make(map[string]any, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func targetOutputPath(targetProfile pack.TargetProfile, filePath string) (string, error) {
	root, err := cleanRelativePath(targetProfile.Spec.Output.Root)
	if err != nil {
		return "", err
	}
	cleaned, err := cleanRelativePath(filePath)
	if err != nil {
		return "", err
	}
	return path.Join(root, cleaned), nil
}

func cleanRelativePath(filePath string) (string, error) {
	cleaned := path.Clean(filepath.ToSlash(strings.TrimSpace(filePath)))
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("path is empty")
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, "/") {
		return "", fmt.Errorf("path must be relative and stay inside the target profile")
	}
	return cleaned, nil
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

func mcpToolsDocument(capability pack.Capability) map[string]any {
	tools := make([]map[string]any, 0, len(capability.Spec.MCP.Tools))
	serverSources := map[string]string{}
	for _, server := range capability.Spec.MCP.Servers {
		serverSources[server.Name] = server.Source
	}
	for _, tool := range capability.Spec.MCP.Tools {
		tools = append(tools, map[string]any{
			"name":           tool.Name,
			"server":         tool.Server,
			"serverSource":   serverSources[tool.Server],
			"toolset":        tool.Toolset,
			"description":    tool.Description,
			"requiredScopes": tool.RequiredScopes,
		})
	}
	return map[string]any{
		"capability": capability.Metadata.Name,
		"tools":      tools,
	}
}

func mcpServerDocument(capability pack.Capability) map[string]any {
	return map[string]any{
		"mcp": mcpConfig(capability),
	}
}

func mcpConfig(capability pack.Capability) map[string]any {
	servers := map[string]any{}
	for _, server := range capability.Spec.MCP.Servers {
		config := map[string]any{
			"type": server.Type,
		}
		if len(server.Command) > 0 {
			config["command"] = server.Command
		}
		if len(server.Env) > 0 {
			config["environment"] = server.Env
		}
		if server.URL != "" {
			config["url"] = server.URL
		}
		if len(server.Headers) > 0 {
			config["headers"] = server.Headers
		}
		if server.OAuth != nil {
			config["oauth"] = server.OAuth
		}
		if server.Timeout != 0 {
			config["timeout"] = server.Timeout
		}
		if server.Enabled != nil {
			config["enabled"] = *server.Enabled
		}
		servers[server.Name] = config
	}
	return servers
}

func collectRules(policies []pack.Policy) map[string]any {
	forbidden := []string{}
	rules := map[string]any{}
	for _, policy := range policies {
		for _, deny := range policy.Spec.Deny {
			if deny.MaxFiles != 0 {
				rules["maxFiles"] = deny.MaxFiles
			}
			if deny.MaxDiffBytes != 0 {
				rules["maxDiffBytes"] = deny.MaxDiffBytes
			}
			forbidden = append(forbidden, deny.Paths...)
		}
		for _, mutate := range policy.Spec.Mutate {
			if mutate.Field == "branch" && mutate.EnsurePrefix != "" {
				rules["branchPrefix"] = mutate.EnsurePrefix
			}
			if mutate.Field == "draft" {
				if value, ok := mutate.Value.(bool); ok {
					rules["forceDraft"] = value
				}
			}
		}
	}
	sort.Strings(forbidden)
	if len(forbidden) > 0 {
		rules["forbiddenPaths"] = forbidden
	}
	return rules
}

func profileSources(loaded *pack.LoadedPack) []string {
	seen := map[string]bool{}
	var sources []string
	for _, capability := range loaded.Capabilities {
		for _, profile := range capability.Spec.Profiles {
			for _, file := range profile.Files {
				if file.Source == "" {
					continue
				}
				source, err := profileSourcePath(loaded.Root, capability.Path, file.Source)
				if err != nil || seen[source] {
					continue
				}
				seen[source] = true
				sources = append(sources, source)
			}
		}
	}
	sort.Strings(sources)
	return sources
}

func buildLockfile(loaded *pack.LoadedPack, files map[string][]byte, target string) lockfile {
	sourceDigests := map[string]string{
		"actlane.yaml": digest(loaded.ManifestRaw),
	}
	for _, capability := range loaded.Capabilities {
		sourceDigests[relToRoot(loaded.Root, capability.Path)] = digest(capability.Raw)
	}
	for _, policy := range loaded.Policies {
		sourceDigests[relToRoot(loaded.Root, policy.Path)] = digest(policy.Raw)
	}
	for _, targetProfile := range loaded.TargetProfiles {
		sourceDigests[relToRoot(loaded.Root, targetProfile.Path)] = digest(targetProfile.Raw)
	}
	for _, source := range profileSources(loaded) {
		data, err := os.ReadFile(source)
		if err != nil {
			continue
		}
		sourceDigests[relToRoot(loaded.Root, source)] = digest(data)
	}

	paths := make([]string, 0, len(files))
	for path := range files {
		if path != "actlane.lock" {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	records := make([]generatedFileRecord, 0, len(paths))
	for _, path := range paths {
		records = append(records, generatedFileRecord{
			Path:   path,
			SHA256: digest(files[path]),
		})
	}
	return lockfile{
		LockfileVersion: 1,
		Pack:            loaded.Manifest.Metadata.Name,
		Version:         loaded.Manifest.Metadata.Version,
		Target:          target,
		Generator:       generatorVersion,
		SourceDigests:   sourceDigests,
		GeneratedFiles:  records,
		Metadata: map[string]string{
			"schema": "actlane.ru/v1alpha1",
		},
	}
}

func compareExisting(outDir string, files map[string][]byte, frozen bool) error {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, rel := range paths {
		existing, err := os.ReadFile(filepath.Join(outDir, rel))
		if err != nil {
			if frozen && rel == "actlane.lock" {
				return fmt.Errorf("lockfile is missing or unreadable: %w", err)
			}
			return fmt.Errorf("generated output is stale: %s is missing", rel)
		}
		if !bytes.Equal(existing, files[rel]) {
			if frozen && rel == "actlane.lock" {
				return fmt.Errorf("lockfile would change")
			}
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

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func relToRoot(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}
