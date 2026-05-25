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
				"version": "0.3.0-alpha.2",
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
