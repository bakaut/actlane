package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateGithubDraftPROpenCodePack(t *testing.T) {
	root := repoRoot(t)
	var stdout, stderr bytes.Buffer

	code := Main([]string{"validate", filepath.Join(root, "packs/github-draft-pr-opencode")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("validate failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "valid") {
		t.Fatalf("validate output should include valid, got %q", stdout.String())
	}
}

func TestGenerateOpenCodeWritesOnlyGeneratedOutput(t *testing.T) {
	packDir := copyPackToTemp(t)
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "opencode"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}

	assertExists(t, filepath.Join(packDir, "generated/opencode/opencode.snippet.jsonc"))
	assertExists(t, filepath.Join(packDir, "generated/opencode/commands/create-github-draft-pr.md"))
	assertExists(t, filepath.Join(packDir, "generated/opencode/agents/github-draft-pr.md"))
	assertExists(t, filepath.Join(packDir, "generated/opencode/instructions/github-draft-pr.md"))
	assertExists(t, filepath.Join(packDir, "generated/policies/policy-bundle.json"))
	assertExists(t, filepath.Join(packDir, "generated/actlane.lock"))
	assertNotExists(t, filepath.Join(packDir, ".opencode"))

	snippet := readFile(t, filepath.Join(packDir, "generated/opencode/opencode.snippet.jsonc"))
	for _, want := range []string{
		`"$schema": "https://opencode.ai/config.json"`,
		`"github-draft-pr"`,
		`"create-github-draft-pr"`,
		`"edit": "ask"`,
		`"bash": "ask"`,
	} {
		if !strings.Contains(snippet, want) {
			t.Fatalf("generated snippet missing %q:\n%s", want, snippet)
		}
	}
}

func TestGenerateCheckDetectsStaleOutputWithoutWriting(t *testing.T) {
	packDir := copyPackToTemp(t)
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "opencode", "--check"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("--check should fail when generated output is missing")
	}
	assertNotExists(t, filepath.Join(packDir, "generated/opencode/opencode.snippet.jsonc"))

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"generate", packDir, "--target", "opencode"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed: %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"generate", packDir, "--target", "opencode", "--check"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("--check should pass after generation\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
}

func TestFrozenLockfileDetectsDrift(t *testing.T) {
	packDir := copyPackToTemp(t)
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "opencode"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed: %s", stderr.String())
	}

	capabilityPath := filepath.Join(packDir, "capabilities/create-github-draft-pr.yaml")
	content := readFile(t, capabilityPath)
	content = strings.Replace(content, "Safely prepare a GitHub draft pull request", "Prepare a GitHub draft pull request", 1)
	if err := os.WriteFile(capabilityPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"generate", packDir, "--target", "opencode", "--frozen-lockfile"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("--frozen-lockfile should fail after source drift")
	}
	if !strings.Contains(stderr.String(), "lockfile") {
		t.Fatalf("expected lockfile drift error, got %q", stderr.String())
	}
}

func TestUnsupportedTargetFailsClearly(t *testing.T) {
	root := repoRoot(t)
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", filepath.Join(root, "packs/github-draft-pr-opencode"), "--target", "mcp"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("unsupported target should fail")
	}
	if !strings.Contains(stderr.String(), "supported target: opencode") {
		t.Fatalf("unsupported target error should list opencode, got %q", stderr.String())
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "packs/github-draft-pr-opencode")); err == nil {
			return wd
		}
		next := filepath.Dir(wd)
		if next == wd {
			t.Fatal("repository root not found")
		}
		wd = next
	}
}

func copyPackToTemp(t *testing.T) string {
	t.Helper()
	src := filepath.Join(repoRoot(t), "packs/github-draft-pr-opencode")
	dst := filepath.Join(t.TempDir(), "github-draft-pr-opencode")
	copyDir(t, src, dst)
	if err := os.RemoveAll(filepath.Join(dst, "generated")); err != nil {
		t.Fatal(err)
	}
	return dst
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			copyDir(t, srcPath, dstPath)
			continue
		}
		data, err := os.ReadFile(srcPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dstPath, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("expected %s not to exist", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("unexpected stat error for %s: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
