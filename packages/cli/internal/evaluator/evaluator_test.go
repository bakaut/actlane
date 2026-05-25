package evaluator

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/actlane/actlane/packages/cli/internal/pack"
)

func TestEvaluateResponsibilityRiskAndEvidence(t *testing.T) {
	loaded := testLoadedPack()
	cases := []struct {
		name   string
		input  map[string]any
		want   string
		denied bool
	}{
		{
			name:  "docs-only",
			input: map[string]any{"files": []any{"README.md"}},
			want:  `"risk":"low"`,
		},
		{
			name:  "package-code",
			input: map[string]any{"files": []any{"packages/cli/main.go"}},
			want:  `"risk":"medium"`,
		},
		{
			name:   "ci-workflow",
			input:  map[string]any{"files": []any{".github/workflows/release.yml"}},
			want:   `"policyDecision":"requires_approval"`,
			denied: true,
		},
		{
			name:   "secret-path",
			input:  map[string]any{"files": []any{"secrets/token.txt"}},
			want:   `"stop":true`,
			denied: true,
		},
		{
			name:   "missing-evidence",
			input:  map[string]any{"files": []any{"README.md"}, "checkEvidence": true, "evidence": map[string]any{"summary": "ok"}},
			want:   "missing evidence: changedFiles",
			denied: true,
		},
		{
			name:   "unregistered-write-tool",
			input:  map[string]any{"files": []any{"README.md"}, "toolsRequested": []any{"github_delete_repository"}},
			want:   "unregistered write MCP tool: github_delete_repository",
			denied: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eval := Evaluate(loaded, Request{Capability: "create-github-draft-pr", Input: tc.input})
			got := compactEvaluation(t, eval)
			if !strings.Contains(got, tc.want) {
				t.Fatalf("evaluation missing %q:\n%s", tc.want, got)
			}
			if tc.denied && eval.Allowed {
				t.Fatalf("expected evaluation to block:\n%s", got)
			}
		})
	}
}

func testLoadedPack() *pack.LoadedPack {
	return &pack.LoadedPack{
		Contracts: []pack.ResponsibilityContract{{
			Document: pack.Document{Metadata: pack.Metadata{Name: "create-github-draft-pr"}},
			Spec: map[string]any{
				"scopes": []any{
					map[string]any{"name": "docs", "paths": []any{"README.md", "docs/**"}, "riskFloor": "low"},
					map[string]any{"name": "code", "paths": []any{"packages/**"}, "riskFloor": "medium"},
					map[string]any{"name": "ci", "paths": []any{".github/workflows/**"}, "riskFloor": "high"},
					map[string]any{"name": "secrets", "paths": []any{"secrets/**"}, "riskFloor": "critical"},
				},
				"risk": map[string]any{"classes": map[string]any{
					"low":      map[string]any{"requiredChecks": []any{"lint"}},
					"medium":   map[string]any{"requiredChecks": []any{"lint", "unit-tests"}},
					"high":     map[string]any{"requiredChecks": []any{"lint", "unit-tests", "security-scan"}, "humanApprovalRequired": true},
					"critical": map[string]any{"humanApprovalRequired": true, "agentMustStop": true},
				}},
				"evidence": map[string]any{"requiredForHandoff": []any{"summary", "changedFiles"}},
				"tools":    map[string]any{"mcp": map[string]any{"denyUnregisteredWriteTools": true}},
			},
		}},
		MCPBindings: []pack.MCPBinding{{
			Spec: pack.MCPBindingSpec{
				CapabilityRef: pack.LocalRef{Name: "create-github-draft-pr"},
				RequiredTools: []pack.MCPToolBinding{{Server: "github", Name: "create_pull_request"}},
			},
		}},
	}
}

func compactEvaluation(t *testing.T, eval Evaluation) string {
	t.Helper()
	data, err := json.Marshal(eval)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
