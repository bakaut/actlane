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

const lockfilePath = "generated/actlane.lock"

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
	files[lockfilePath] = mustJSON(buildLockfile(loaded, files, opts.Target))

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
	renderer, err := rendererFor(targetProfile.Spec.Target)
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	if err := renderGuidance(files, loaded); err != nil {
		return nil, err
	}
	if err := renderer.Render(files, loaded, capability, targetProfile); err != nil {
		return nil, err
	}
	if err := renderCapabilityProfile(files, loaded.Root, capability, targetProfile, target); err != nil {
		return nil, err
	}
	if targetProfile.Spec.Generate.MCP {
		renderMCPBindingArtifacts(files, loaded)
	}
	files["generated/policies/policy-bundle.json"] = mustJSON(policyBundle{
		Pack:         loaded.Manifest.Metadata.Name,
		Version:      loaded.Manifest.Metadata.Version,
		Target:       target,
		Capabilities: []string{capability.Metadata.Name},
		Decisions:    []string{"allow", "deny", "mutate", "requires-approval"},
		Rules:        rules,
	})
	return files, nil
}

func renderGuidance(files map[string][]byte, loaded *pack.LoadedPack) error {
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
	files[filepath.ToSlash(compose.Output)] = []byte(strings.TrimRight(b.String(), "\n") + "\n")
	return nil
}

func renderCapabilityProfile(files map[string][]byte, packRoot string, capability pack.Capability, targetProfile pack.TargetProfile, target string) error {
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

func compareExisting(outDir string, files map[string][]byte, frozen bool) error {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, rel := range paths {
		existing, err := os.ReadFile(filepath.Join(outDir, rel))
		if err != nil {
			if frozen && rel == lockfilePath {
				return fmt.Errorf("lockfile is missing or unreadable: %w", err)
			}
			return fmt.Errorf("generated output is stale: %s is missing", rel)
		}
		if !bytes.Equal(existing, files[rel]) {
			if frozen && rel == lockfilePath {
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
