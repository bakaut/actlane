package profile

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"

	"github.com/actlane/actlane/packages/cli/internal/pack"
)

const generatorVersion = "actlane-go-profile-0.3.0-alpha.5"

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

func profileSources(loaded *pack.LoadedPack) []string {
	seen := map[string]bool{}
	var sources []string
	if len(loaded.Capabilities) == 0 {
		return nil
	}
	capability := loaded.Capabilities[0]
	for _, targetProfile := range loaded.TargetProfiles {
		for _, file := range targetProfileFiles(targetProfile) {
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
	sort.Strings(sources)
	return sources
}

func guidanceSources(loaded *pack.LoadedPack) []string {
	seen := map[string]bool{}
	var sources []string
	for _, source := range loaded.Manifest.Spec.Guidance.Sources {
		if source.Path == "" {
			continue
		}
		cleaned, err := cleanRelativePath(source.Path)
		if err != nil {
			continue
		}
		path := filepath.Join(loaded.Root, filepath.FromSlash(cleaned))
		if err := ensureInsideRoot(loaded.Root, path); err != nil || seen[path] {
			continue
		}
		seen[path] = true
		sources = append(sources, path)
	}
	sort.Strings(sources)
	return sources
}

func skillResourceSources(loaded *pack.LoadedPack) []string {
	seen := map[string]bool{}
	var sources []string
	for _, skill := range loaded.Skills {
		for _, resource := range append(append([]pack.SkillResource{}, skill.Spec.Scripts...), append(skill.Spec.References, skill.Spec.Assets...)...) {
			if resource.Source == "" {
				continue
			}
			source, err := profileSourcePath(loaded.Root, skill.Path, resource.Source)
			if err != nil || seen[source] {
				continue
			}
			seen[source] = true
			sources = append(sources, source)
		}
	}
	sort.Strings(sources)
	return sources
}

func lockfilePath(target string) string {
	return "generated/" + target + "/actlane.lock"
}

func buildLockfile(loaded *pack.LoadedPack, files map[string][]byte, target, lockPath string) lockfile {
	sourceDigests := map[string]string{
		"actlane.yaml": digest(loaded.ManifestRaw),
	}
	for _, capability := range loaded.Capabilities {
		sourceDigests[relToRoot(loaded.Root, capability.Path)] = digest(capability.Raw)
	}
	for _, skill := range loaded.Skills {
		sourceDigests[relToRoot(loaded.Root, skill.Path)] = digest(skill.Raw)
	}
	for _, command := range loaded.Commands {
		sourceDigests[relToRoot(loaded.Root, command.Path)] = digest(command.Raw)
	}
	for _, agent := range loaded.Agents {
		sourceDigests[relToRoot(loaded.Root, agent.Path)] = digest(agent.Raw)
	}
	for _, contract := range loaded.Contracts {
		sourceDigests[relToRoot(loaded.Root, contract.Path)] = digest(contract.Raw)
	}
	for _, policy := range loaded.Policies {
		sourceDigests[relToRoot(loaded.Root, policy.Path)] = digest(policy.Raw)
	}
	for _, binding := range loaded.MCPBindings {
		sourceDigests[relToRoot(loaded.Root, binding.Path)] = digest(binding.Raw)
	}
	for _, targetProfile := range loaded.TargetProfiles {
		sourceDigests[relToRoot(loaded.Root, targetProfile.Path)] = digest(targetProfile.Raw)
	}
	sources := append(profileSources(loaded), guidanceSources(loaded)...)
	sources = append(sources, skillResourceSources(loaded)...)
	for _, source := range sources {
		data, err := os.ReadFile(source)
		if err != nil {
			continue
		}
		sourceDigests[relToRoot(loaded.Root, source)] = digest(data)
	}

	paths := make([]string, 0, len(files))
	for path := range files {
		if path != lockPath {
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
