package profile

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/actlane/actlane/packages/cli/internal/pack"
)

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

func targetProfileGeneratedPath(targetProfile pack.TargetProfile, file pack.TargetProfileFile) (string, error) {
	if file.GeneratedPath != "" {
		return cleanRelativePath(file.GeneratedPath)
	}
	return targetOutputPath(targetProfile, file.TargetPath)
}

func targetConfigPath(targetProfile pack.TargetProfile) (string, error) {
	if targetProfile.Spec.Output.Config != "" {
		return targetOutputPath(targetProfile, targetProfile.Spec.Output.Config)
	}
	if filename := targetProfile.Spec.OpenCode.Config.Filename; filename != "" {
		return targetOutputPath(targetProfile, filename)
	}
	return "", fmt.Errorf("target profile %s config filename is required", targetProfile.Metadata.Name)
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
