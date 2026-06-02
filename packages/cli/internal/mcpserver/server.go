package mcpserver

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/actlane/actlane/packages/cli/internal/evaluator"
	"github.com/actlane/actlane/packages/cli/internal/pack"
)

type Server struct {
	loaded *pack.LoadedPack
}

type PolicyBundle struct {
	Pack           string                 `json:"pack"`
	Version        string                 `json:"version"`
	Target         string                 `json:"target"`
	Capabilities   []string               `json:"capabilities"`
	Decisions      []string               `json:"decisions"`
	Rules          PolicyBundleRules      `json:"rules"`
	MCPBindings    []BundleMCPBinding     `json:"mcpBindings"`
	Responsibility []BundleResponsibility `json:"responsibility,omitempty"`
}

type PolicyBundleRules struct {
	Capabilities  []string                `json:"capabilities"`
	Defaults      map[string]any          `json:"defaults"`
	BranchPrefix  string                  `json:"branchPrefix"`
	Confirmation  pack.PolicyConfirmation `json:"confirmation"`
	RepoAllowlist []string                `json:"repoAllowlist"`
	ForbidPaths   []string                `json:"forbidPaths"`
	MaxFiles      int                     `json:"maxFiles"`
	MaxDiffKB     int                     `json:"maxDiffKb"`
	Approval      pack.PolicyApproval     `json:"approval"`
	Audit         pack.PolicyAudit        `json:"audit"`
}

type BundleMCPBinding struct {
	Name           string                  `json:"name"`
	Capability     string                  `json:"capability"`
	Handler        string                  `json:"handler"`
	GeneratedTools []pack.MCPGeneratedTool `json:"generatedTools"`
	RequiredTools  []pack.MCPToolBinding   `json:"requiredTools"`
}

type BundleResponsibility struct {
	Name string         `json:"name"`
	Spec map[string]any `json:"spec"`
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id,omitempty"`
	Result  any            `json:"result,omitempty"`
	Error   *responseError `json:"error,omitempty"`
}

type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type callParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type toolResult struct {
	Content []toolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func New(loaded *pack.LoadedPack) *Server {
	return &Server{loaded: loaded}
}

func NewFromPolicyBundle(bundle PolicyBundle) *Server {
	return New(LoadedFromPolicyBundle(bundle))
}

func LoadedFromPolicyBundle(bundle PolicyBundle) *pack.LoadedPack {
	capabilities := bundle.Rules.Capabilities
	if len(capabilities) == 0 {
		capabilities = bundle.Capabilities
	}
	loaded := &pack.LoadedPack{
		Manifest: pack.CapabilityPack{
			Document: pack.Document{
				Metadata: pack.Metadata{
					Name:    bundle.Pack,
					Version: bundle.Version,
				},
			},
		},
		Policies: []pack.Policy{{
			Document: pack.Document{
				Metadata: pack.Metadata{Name: "policy-bundle"},
			},
			Spec: pack.PolicySpec{
				Match: pack.PolicyMatch{Capabilities: capabilities},
				Mutate: pack.PolicyMutateSpec{
					Defaults: bundle.Rules.Defaults,
					Ensure:   pack.PolicyEnsure{BranchPrefix: bundle.Rules.BranchPrefix},
				},
				Validate: pack.PolicyValidate{
					Confirmation:  bundle.Rules.Confirmation,
					RepoAllowlist: bundle.Rules.RepoAllowlist,
					ForbidPaths:   bundle.Rules.ForbidPaths,
					Limits: pack.PolicyLimits{
						MaxFiles:  bundle.Rules.MaxFiles,
						MaxDiffKB: bundle.Rules.MaxDiffKB,
					},
				},
				Approval: bundle.Rules.Approval,
				Audit:    bundle.Rules.Audit,
			},
		}},
	}
	for _, binding := range bundle.MCPBindings {
		loaded.MCPBindings = append(loaded.MCPBindings, pack.MCPBinding{
			Document: pack.Document{
				Metadata: pack.Metadata{Name: binding.Name},
			},
			Spec: pack.MCPBindingSpec{
				CapabilityRef:  pack.LocalRef{Name: binding.Capability},
				Strategy:       pack.MCPBindingStrategy{Handler: binding.Handler},
				GeneratedTools: binding.GeneratedTools,
				RequiredTools:  binding.RequiredTools,
			},
		})
	}
	for _, responsibility := range bundle.Responsibility {
		loaded.Contracts = append(loaded.Contracts, pack.ResponsibilityContract{
			Document: pack.Document{
				Metadata: pack.Metadata{Name: responsibility.Name},
			},
			Spec: responsibility.Spec,
		})
	}
	return loaded
}

func (s *Server) Serve(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var req request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			if err := writeResponse(w, responseErrorOnly(nil, -32700, "parse error")); err != nil {
				return err
			}
			continue
		}
		if req.ID == nil {
			continue
		}
		resp := s.handle(req)
		if err := writeResponse(w, resp); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (s *Server) handle(req request) response {
	switch req.Method {
	case "initialize":
		return responseOK(req.ID, map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "actlane-safe-gitops",
				"version": "0.3.0-alpha.6",
			},
		})
	case "tools/list":
		return responseOK(req.ID, map[string]any{"tools": s.tools()})
	case "tools/call":
		var params callParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return responseErrorOnly(req.ID, -32602, "invalid tools/call params")
		}
		result, err := s.callTool(params.Name, params.Arguments)
		if err != nil {
			return responseErrorOnly(req.ID, -32602, err.Error())
		}
		return responseOK(req.ID, result)
	default:
		return responseErrorOnly(req.ID, -32601, "method not found")
	}
}

func (s *Server) tools() []map[string]any {
	var tools []map[string]any
	if len(s.loaded.RuntimeProfiles) > 0 {
		tools = append(tools, map[string]any{
			"name":        "actlane_classify",
			"description": "Classify a user task against Actlane runtime profiles and return advisory mode, risk flags, candidate capabilities, and required evidence.",
			"inputSchema": classifyInputSchema(),
		})
	}
	for _, binding := range s.loaded.MCPBindings {
		if binding.Spec.Strategy.Handler != "actlane.policy.evaluate" {
			continue
		}
		for _, tool := range generatedTools(binding) {
			if tool.Name == "" {
				continue
			}
			tools = append(tools, map[string]any{
				"name":        tool.Name,
				"description": tool.Description,
				"inputSchema": inputSchema(),
			})
		}
	}
	return tools
}

func (s *Server) callTool(name string, args map[string]any) (toolResult, error) {
	if name == "actlane_classify" {
		result := s.classify(args)
		text, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return toolResult{}, err
		}
		return toolResult{
			Content: []toolContent{{
				Type: "text",
				Text: string(text),
			}},
		}, nil
	}
	binding, tool, ok := s.generatedTool(name)
	if !ok {
		return toolResult{}, fmt.Errorf("unknown tool %q", name)
	}
	mode := tool.Mode
	if mode == "" {
		mode = "audit"
	}
	eval := evaluator.Evaluate(s.loaded, evaluator.Request{
		Tool:       name,
		Mode:       mode,
		Capability: binding.Spec.CapabilityRef.Name,
		Input:      args,
	})
	text, err := json.MarshalIndent(eval, "", "  ")
	if err != nil {
		return toolResult{}, err
	}
	return toolResult{
		IsError: mode == "enforce" && !eval.Allowed,
		Content: []toolContent{{
			Type: "text",
			Text: string(text),
		}},
	}, nil
}

func (s *Server) generatedTool(name string) (pack.MCPBinding, pack.MCPGeneratedTool, bool) {
	for _, binding := range s.loaded.MCPBindings {
		if binding.Spec.Strategy.Handler != "actlane.policy.evaluate" {
			continue
		}
		for _, tool := range generatedTools(binding) {
			if tool.Name == name {
				return binding, tool, true
			}
		}
	}
	return pack.MCPBinding{}, pack.MCPGeneratedTool{}, false
}

type classificationResult struct {
	WorkType              string   `json:"workType"`
	RiskFlags             []string `json:"riskFlags"`
	TechHints             []string `json:"techHints"`
	Mode                  string   `json:"mode"`
	Confidence            string   `json:"confidence"`
	CandidateCapabilities []string `json:"candidateCapabilities"`
	RequiredEvidence      []string `json:"requiredEvidence"`
	NextStep              string   `json:"nextStep"`
	RuntimeProfile        string   `json:"runtimeProfile,omitempty"`
	EvidenceContract      string   `json:"evidenceContract,omitempty"`
}

func (s *Server) classify(args map[string]any) classificationResult {
	task := lowerStringArg(args, "task")
	diffSummary := lowerStringArg(args, "diff_summary")
	if diffSummary == "" {
		diffSummary = lowerStringArg(args, "diffSummary")
	}
	files := stringSliceArg(args, "changed_files")
	if len(files) == 0 {
		files = stringSliceArg(args, "changedFiles")
	}
	profile := s.runtimeProfileForTask(task, files, diffSummary)
	capabilities := append([]string{}, profile.Spec.CandidateCapabilities...)
	if len(capabilities) == 0 && profile.Spec.CapabilityRef.Name != "" {
		capabilities = []string{profile.Spec.CapabilityRef.Name}
	}
	workType := detectWorkType(task, diffSummary, files, profile)
	riskFlags := detectRiskFlags(task, diffSummary, files, profile)
	if riskFlags == nil {
		riskFlags = []string{}
	}
	mode := profile.Spec.DefaultMode
	if mode == "" {
		mode = "advise"
	}
	humanBoundary := false
	if hasAny(riskFlags, profile.Spec.HighRisk.Flags) {
		if profile.Spec.HighRisk.Mode != "" {
			mode = profile.Spec.HighRisk.Mode
		}
		humanBoundary = profile.Spec.HighRisk.RequireHumanBoundary
	}
	evidence := s.evidenceForCapability(profile.Spec.CapabilityRef.Name)
	requiredEvidence := append([]string{}, evidence.Spec.SummaryFields...)
	if requiredEvidence == nil {
		requiredEvidence = []string{}
	}
	nextStep := profile.Spec.Recommendations.NextStep
	if humanBoundary && profile.Spec.Recommendations.HumanBoundaryNextStep != "" {
		nextStep = profile.Spec.Recommendations.HumanBoundaryNextStep
	} else if nextStep == "" {
		nextStep = "Continue with existing tools; use Actlane policy checks before mutating downstream actions."
	}
	return classificationResult{
		WorkType:              workType,
		RiskFlags:             riskFlags,
		TechHints:             detectTechHints(task, diffSummary, files, profile),
		Mode:                  mode,
		Confidence:            confidence(workType, riskFlags),
		CandidateCapabilities: capabilities,
		RequiredEvidence:      requiredEvidence,
		NextStep:              nextStep,
		RuntimeProfile:        profile.Metadata.Name,
		EvidenceContract:      evidence.Metadata.Name,
	}
}

func (s *Server) runtimeProfileForTask(task string, files []string, diffSummary string) pack.RuntimeProfile {
	if len(s.loaded.RuntimeProfiles) == 0 {
		return pack.RuntimeProfile{Spec: pack.RuntimeProfileSpec{DefaultMode: "advise"}}
	}
	return s.loaded.RuntimeProfiles[0]
}

func (s *Server) evidenceForCapability(capability string) pack.EvidenceContract {
	for _, evidence := range s.loaded.Evidence {
		if evidence.Spec.CapabilityRef.Name == capability {
			return evidence
		}
	}
	if len(s.loaded.Evidence) > 0 {
		return s.loaded.Evidence[0]
	}
	return pack.EvidenceContract{}
}

func detectWorkType(task, diffSummary string, files []string, profile pack.RuntimeProfile) string {
	text := strings.Join(append([]string{task, diffSummary}, lowerFiles(files)...), " ")
	counts := map[string]int{}
	addIfMatch(counts, "ci_change", text, append([]string{".github/workflows", "workflow", "ci", "actions"}, profile.Spec.ClassificationKeywords.CI...))
	addIfMatch(counts, "docs_change", text, append([]string{".md", "readme", "docs", "documentation"}, profile.Spec.ClassificationKeywords.Docs...))
	addIfMatch(counts, "test_change", text, append([]string{"test", "_test", "spec", "pytest"}, profile.Spec.ClassificationKeywords.Tests...))
	addIfMatch(counts, "dependency_change", text, append([]string{"go.mod", "go.sum", "package.json", "package-lock", "dependency", "dependencies"}, profile.Spec.ClassificationKeywords.Dependency...))
	addIfMatch(counts, "config_change", text, append([]string{".yaml", ".yml", ".toml", ".json", "config"}, profile.Spec.ClassificationKeywords.Config...))
	addIfMatch(counts, "code_change", text, append([]string{".go", ".py", ".js", ".ts", ".java", "code"}, profile.Spec.ClassificationKeywords.Code...))
	if len(counts) == 0 {
		if len(profile.Spec.WorkTypes) > 0 {
			return profile.Spec.WorkTypes[0]
		}
		return "unknown_or_mixed"
	}
	best := ""
	bestCount := 0
	ties := 0
	for workType, count := range counts {
		if count > bestCount {
			best = workType
			bestCount = count
			ties = 1
		} else if count == bestCount {
			ties++
		}
	}
	if ties > 1 {
		return "unknown_or_mixed"
	}
	return best
}

func detectRiskFlags(task, diffSummary string, files []string, profile pack.RuntimeProfile) []string {
	text := strings.Join(append([]string{task, diffSummary}, lowerFiles(files)...), " ")
	var risks []string
	if containsAny(text, []string{".env", "secret", "token", "password", "credential", "private_key", "secrets/"}) {
		risks = append(risks, "secrets_sensitive")
	}
	if containsAny(text, []string{"prod", "production", "deploy", "release", "live"}) {
		risks = append(risks, "production_sensitive")
	}
	if containsAny(text, []string{"destroy", "delete", "drop", "force push", "terraform apply", "kubectl apply", "kubectl delete"}) {
		risks = append(risks, "destructive_operation")
	}
	if containsAny(text, []string{"security", "auth", "permission", "rbac", "oauth", "vulnerability"}) {
		risks = append(risks, "security_sensitive")
	}
	if containsAny(text, []string{"api", "public endpoint", "breaking change"}) {
		risks = append(risks, "public_api_sensitive")
	}
	return filterAllowedRisks(uniqueStrings(risks), profile.Spec.RiskFlags)
}

func detectTechHints(task, diffSummary string, files []string, profile pack.RuntimeProfile) []string {
	text := strings.Join(append([]string{task, diffSummary}, lowerFiles(files)...), " ")
	var hints []string
	for _, hint := range profile.Spec.TechHints {
		if strings.Contains(text, strings.ToLower(hint)) {
			hints = append(hints, hint)
		}
	}
	if containsAny(text, []string{"github", "pull request", "pr", ".github"}) {
		hints = append(hints, "github")
	}
	if containsAny(text, []string{"git", "branch", "commit"}) {
		hints = append(hints, "git")
	}
	return uniqueStrings(hints)
}

func addIfMatch(counts map[string]int, workType, text string, needles []string) {
	for _, needle := range needles {
		if needle != "" && strings.Contains(text, strings.ToLower(needle)) {
			counts[workType]++
		}
	}
}

func confidence(workType string, riskFlags []string) string {
	if workType == "unknown_or_mixed" {
		return "low"
	}
	if len(riskFlags) > 0 {
		return "medium"
	}
	return "high"
}

func filterAllowedRisks(values, allowed []string) []string {
	if len(allowed) == 0 {
		return values
	}
	allowedSet := map[string]bool{}
	for _, value := range allowed {
		allowedSet[value] = true
	}
	var filtered []string
	for _, value := range values {
		if allowedSet[value] {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func lowerFiles(files []string) []string {
	values := make([]string, 0, len(files))
	for _, file := range files {
		values = append(values, strings.ToLower(file))
	}
	return values
}

func containsAny(text string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func hasAny(values, needles []string) bool {
	if len(values) == 0 || len(needles) == 0 {
		return false
	}
	set := map[string]bool{}
	for _, value := range values {
		set[value] = true
	}
	for _, needle := range needles {
		if set[needle] {
			return true
		}
	}
	return false
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var unique []string
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		unique = append(unique, value)
	}
	return unique
}

func lowerStringArg(args map[string]any, key string) string {
	value, _ := args[key].(string)
	return strings.ToLower(value)
}

func stringSliceArg(args map[string]any, key string) []string {
	raw, ok := args[key]
	if !ok {
		return nil
	}
	values, ok := raw.([]any)
	if !ok {
		if stringsValue, ok := raw.([]string); ok {
			return stringsValue
		}
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if stringValue, ok := value.(string); ok {
			result = append(result, stringValue)
		}
	}
	return result
}

func classifyInputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task":           map[string]any{"type": "string"},
			"changed_files":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"changedFiles":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"branch":         map[string]any{"type": "string"},
			"diff_summary":   map[string]any{"type": "string"},
			"diffSummary":    map[string]any{"type": "string"},
			"existing_tools": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		"additionalProperties": true,
	}
}

func inputSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": true,
	}
}

func writeResponse(w io.Writer, resp response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

func responseOK(id any, result any) response {
	return response{JSONRPC: "2.0", ID: id, Result: result}
}

func responseErrorOnly(id any, code int, message string) response {
	return response{JSONRPC: "2.0", ID: id, Error: &responseError{Code: code, Message: message}}
}

func generatedTools(binding pack.MCPBinding) []pack.MCPGeneratedTool {
	if len(binding.Spec.GeneratedTools) > 0 {
		return binding.Spec.GeneratedTools
	}
	if binding.Spec.GeneratedTool.Name != "" {
		return []pack.MCPGeneratedTool{binding.Spec.GeneratedTool}
	}
	return nil
}
