package mcpserver

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/actlane/actlane/packages/cli/internal/pack"
)

type Server struct {
	loaded *pack.LoadedPack
}

type PolicyBundle struct {
	Pack         string             `json:"pack"`
	Version      string             `json:"version"`
	Target       string             `json:"target"`
	Capabilities []string           `json:"capabilities"`
	Decisions    []string           `json:"decisions"`
	Rules        PolicyBundleRules  `json:"rules"`
	MCPBindings  []BundleMCPBinding `json:"mcpBindings"`
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

type evaluation struct {
	Tool           string         `json:"tool"`
	Mode           string         `json:"mode"`
	Capability     string         `json:"capability"`
	PolicyDecision string         `json:"policyDecision"`
	Allowed        bool           `json:"allowed"`
	Reasons        []string       `json:"reasons,omitempty"`
	OriginalInput  map[string]any `json:"originalInput"`
	MutatedInput   map[string]any `json:"mutatedInput"`
	Next           []nextCall     `json:"next,omitempty"`
	PolicyRefs     []string       `json:"policyRefs"`
}

type nextCall struct {
	Server    string         `json:"server"`
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"arguments"`
}

func New(loaded *pack.LoadedPack) *Server {
	return &Server{loaded: loaded}
}

func NewFromPolicyBundle(bundle PolicyBundle) *Server {
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
	return New(loaded)
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
				"version": "0.2.0-alpha.1",
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
	binding, tool, ok := s.generatedTool(name)
	if !ok {
		return toolResult{}, fmt.Errorf("unknown tool %q", name)
	}
	mode := tool.Mode
	if mode == "" {
		mode = "audit"
	}
	eval := s.evaluate(binding, name, mode, args)
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

func (s *Server) evaluate(binding pack.MCPBinding, toolName, mode string, input map[string]any) evaluation {
	original := cloneMap(input)
	mutated := cloneMap(input)
	policies := s.policiesFor(binding.Spec.CapabilityRef.Name)
	reasons := mutate(mutated, policies)
	reasons = append(reasons, validate(mutated, policies)...)
	allowed := len(reasons) == 0
	decision := "allow"
	if !allowed {
		decision = "deny"
	}
	eval := evaluation{
		Tool:           toolName,
		Mode:           mode,
		Capability:     binding.Spec.CapabilityRef.Name,
		PolicyDecision: decision,
		Allowed:        allowed,
		Reasons:        reasons,
		OriginalInput:  original,
		MutatedInput:   mutated,
		PolicyRefs:     policyNames(policies),
	}
	if allowed {
		eval.Next = s.nextCalls(binding.Spec.CapabilityRef.Name, mutated)
	}
	return eval
}

func (s *Server) nextCalls(capability string, input map[string]any) []nextCall {
	var calls []nextCall
	for _, binding := range s.loaded.MCPBindings {
		if binding.Spec.CapabilityRef.Name != capability || binding.Spec.Strategy.Handler == "actlane.policy.evaluate" {
			continue
		}
		for _, tool := range binding.Spec.RequiredTools {
			call := nextCall{
				Server:    tool.Server,
				Tool:      tool.Server + "_" + tool.Name,
				Arguments: map[string]any{},
			}
			switch tool.Name {
			case "create_branch":
				call.Arguments = pick(input, "repo", "baseBranch", "branch")
			case "push_files":
				call.Arguments = pick(input, "repo", "branch", "files")
			case "create_pull_request":
				call.Arguments = pick(input, "repo", "baseBranch", "branch", "title", "summary", "draft")
			default:
				call.Arguments = cloneMap(input)
			}
			calls = append(calls, call)
		}
	}
	return calls
}

func (s *Server) policiesFor(capability string) []pack.Policy {
	var policies []pack.Policy
	for _, policy := range s.loaded.Policies {
		for _, name := range policy.Spec.Match.Capabilities {
			if name == capability {
				policies = append(policies, policy)
				break
			}
		}
	}
	return policies
}

func mutate(input map[string]any, policies []pack.Policy) []string {
	var reasons []string
	for _, policy := range policies {
		for key, value := range policy.Spec.Mutate.Defaults {
			if _, exists := input[key]; !exists {
				input[key] = value
			}
		}
		prefix := policy.Spec.Mutate.Ensure.BranchPrefix
		if prefix != "" {
			branch, _ := input["branch"].(string)
			if branch == "" {
				reasons = append(reasons, "branch is required")
				continue
			}
			if !strings.HasPrefix(branch, prefix) {
				input["branch"] = prefix + branch
			}
		}
	}
	return reasons
}

func validate(input map[string]any, policies []pack.Policy) []string {
	var reasons []string
	for _, policy := range policies {
		confirmation := policy.Spec.Validate.Confirmation
		if confirmation.Field != "" && input[confirmation.Field] != confirmation.MustBe {
			reasons = append(reasons, fmt.Sprintf("%s must be %v", confirmation.Field, confirmation.MustBe))
		}
		if len(policy.Spec.Validate.RepoAllowlist) > 0 {
			repo, _ := input["repo"].(string)
			if !contains(policy.Spec.Validate.RepoAllowlist, repo) {
				reasons = append(reasons, "repo is not allowlisted")
			}
		}
		files := stringSlice(input["files"])
		for _, file := range files {
			if forbidden(file, policy.Spec.Validate.ForbidPaths) {
				reasons = append(reasons, "file is forbidden: "+file)
			}
		}
		if policy.Spec.Validate.Limits.MaxFiles > 0 && len(files) > policy.Spec.Validate.Limits.MaxFiles {
			reasons = append(reasons, "too many files")
		}
		if policy.Spec.Validate.Limits.MaxDiffKB > 0 {
			if diffKB, ok := number(input["diffKb"]); ok && int(diffKB) > policy.Spec.Validate.Limits.MaxDiffKB {
				reasons = append(reasons, "diff is too large")
			}
		}
	}
	return reasons
}

func inputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"repo":       map[string]any{"type": "string"},
			"baseBranch": map[string]any{"type": "string"},
			"branch":     map[string]any{"type": "string"},
			"title":      map[string]any{"type": "string"},
			"summary":    map[string]any{"type": "string"},
			"files":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"confirmed":  map[string]any{"type": "boolean"},
			"diffKb":     map[string]any{"type": "number"},
		},
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

func cloneMap(input map[string]any) map[string]any {
	clone := map[string]any{}
	for key, value := range input {
		clone[key] = value
	}
	return clone
}

func pick(input map[string]any, keys ...string) map[string]any {
	selected := map[string]any{}
	for _, key := range keys {
		if value, ok := input[key]; ok {
			selected[key] = value
		}
	}
	return selected
}

func policyNames(policies []pack.Policy) []string {
	names := make([]string, 0, len(policies))
	for _, policy := range policies {
		names = append(names, policy.Metadata.Name)
	}
	return names
}

func stringSlice(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	values := make([]string, 0, len(raw))
	for _, item := range raw {
		text, ok := item.(string)
		if ok {
			values = append(values, text)
		}
	}
	return values
}

func forbidden(file string, patterns []string) bool {
	for _, pattern := range patterns {
		if pattern == file {
			return true
		}
		if strings.HasSuffix(pattern, "/**") && strings.HasPrefix(file, strings.TrimSuffix(pattern, "/**")+"/") {
			return true
		}
		if ok, _ := path.Match(pattern, file); ok {
			return true
		}
	}
	return false
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func number(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	default:
		return 0, false
	}
}
