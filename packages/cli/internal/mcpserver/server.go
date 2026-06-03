package mcpserver

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/actlane/actlane/packages/cli/internal/evaluator"
	"github.com/actlane/actlane/packages/cli/internal/pack"
)

type Server struct {
	loaded           *pack.LoadedPack
	evidenceStore    map[string]evidenceSummary
	runStore         map[string]runCapabilityResult
	latestEvidenceID string
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
	return &Server{
		loaded:        loaded,
		evidenceStore: map[string]evidenceSummary{},
		runStore:      map[string]runCapabilityResult{},
	}
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
				"version": "0.3.0-alpha.10",
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
	if len(s.loaded.Capabilities) > 0 {
		tools = append(tools, map[string]any{
			"name":        "actlane_load_capability",
			"description": "Return a compact Actlane capability view derived from source contracts without mutation or execution.",
			"inputSchema": loadCapabilityInputSchema(),
		})
		tools = append(tools, map[string]any{
			"name":        "actlane_run_capability",
			"description": "Evaluate and prepare a guarded capability run through Actlane policy and MCPBinding-derived downstream plan.",
			"inputSchema": runCapabilityInputSchema(),
		})
		tools = append(tools, map[string]any{
			"name":        "actlane_get_evidence",
			"description": "Return compact evidence captured during this Actlane MCP session by id or latest marker.",
			"inputSchema": getEvidenceInputSchema(),
		})
		tools = append(tools, map[string]any{
			"name":        "actlane_prepare_delivery",
			"description": "Prepare a final read-only delivery summary from the latest Actlane run and compact evidence.",
			"inputSchema": prepareDeliveryInputSchema(),
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
	if name == "actlane_load_capability" {
		result, err := s.loadCapability(args)
		if err != nil {
			return toolResult{}, err
		}
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
	if name == "actlane_run_capability" {
		result, err := s.runCapability(args)
		if err != nil {
			return toolResult{}, err
		}
		text, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return toolResult{}, err
		}
		return toolResult{
			IsError: result.Mode == "enforce" && !result.Allowed,
			Content: []toolContent{{
				Type: "text",
				Text: string(text),
			}},
		}, nil
	}
	if name == "actlane_get_evidence" {
		result, err := s.getEvidence(args)
		if err != nil {
			return toolResult{}, err
		}
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
	if name == "actlane_prepare_delivery" {
		result, err := s.prepareDelivery(args)
		if err != nil {
			return toolResult{}, err
		}
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

type runCapabilityResult struct {
	Capability            string               `json:"capability"`
	Mode                  string               `json:"mode"`
	PolicyDecision        string               `json:"policyDecision"`
	Allowed               bool                 `json:"allowed"`
	Risk                  string               `json:"risk,omitempty"`
	ImpactedScopes        []string             `json:"impactedScopes,omitempty"`
	RequiredChecks        []string             `json:"requiredChecks,omitempty"`
	RequiredEvidence      []string             `json:"requiredEvidence,omitempty"`
	HumanApprovalRequired bool                 `json:"humanApprovalRequired,omitempty"`
	Stop                  bool                 `json:"stop,omitempty"`
	Reasons               []string             `json:"reasons,omitempty"`
	OriginalInput         map[string]any       `json:"originalInput"`
	MutatedInput          map[string]any       `json:"mutatedInput"`
	PolicyRefs            []string             `json:"policyRefs"`
	ResponsibilityRefs    []string             `json:"responsibilityRefs,omitempty"`
	DownstreamPlan        []evaluator.NextCall `json:"downstreamPlan,omitempty"`
	AdapterSource         string               `json:"adapterSource"`
	AdapterExecutions     []adapterExecution   `json:"adapterExecutions,omitempty"`
	Evidence              evidenceSummary      `json:"evidence"`
	Execution             executionResult      `json:"execution"`
}

type executionResult struct {
	Performed bool   `json:"performed"`
	Reason    string `json:"reason"`
}

type adapterExecution struct {
	ID             string   `json:"id"`
	Binding        string   `json:"binding"`
	Server         string   `json:"server"`
	Tool           string   `json:"tool"`
	Status         string   `json:"status"`
	Performed      bool     `json:"performed"`
	Reason         string   `json:"reason"`
	RequiredScopes []string `json:"requiredScopes,omitempty"`
	EvidenceID     string   `json:"evidenceId"`
}

type evidenceSummary struct {
	ID                string         `json:"id"`
	Contract          string         `json:"contract,omitempty"`
	SummaryFields     []string       `json:"summaryFields"`
	Values            map[string]any `json:"values"`
	MissingFields     []string       `json:"missingFields,omitempty"`
	RawOutput         string         `json:"rawOutput,omitempty"`
	Redacted          bool           `json:"redacted"`
	DeliveryChecklist []string       `json:"deliveryChecklist,omitempty"`
}

type evidenceLookupResult struct {
	Source   string          `json:"source"`
	Evidence evidenceSummary `json:"evidence"`
}

type deliveryResult struct {
	Delivery deliverySummary `json:"delivery"`
}

type deliverySummary struct {
	Capability               string             `json:"capability"`
	Summary                  string             `json:"summary"`
	PolicyDecision           string             `json:"policyDecision"`
	Allowed                  bool               `json:"allowed"`
	Risk                     string             `json:"risk,omitempty"`
	ResidualRisk             string             `json:"residualRisk,omitempty"`
	HumanApprovalRequired    bool               `json:"humanApprovalRequired"`
	RequiresApproval         bool               `json:"requiresApproval"`
	Stop                     bool               `json:"stop,omitempty"`
	WhatChanged              []string           `json:"whatChanged"`
	WhatWasChecked           []string           `json:"whatWasChecked"`
	Risky                    []string           `json:"risky,omitempty"`
	RequiresHumanResolution  []string           `json:"requiresHumanResolution,omitempty"`
	EvidenceID               string             `json:"evidenceId"`
	Evidence                 evidenceSummary    `json:"evidence"`
	AdapterExecutions        []adapterExecution `json:"adapterExecutions,omitempty"`
	ExternalExecutionPlanned bool               `json:"externalExecutionPlanned"`
	ExternalExecutionDone    bool               `json:"externalExecutionDone"`
}

func (s *Server) runCapability(args map[string]any) (runCapabilityResult, error) {
	name := stringArg(args, "name")
	if name == "" {
		name = stringArg(args, "capability")
	}
	capability, ok := s.capabilityByName(name)
	if !ok {
		if name == "" && len(s.loaded.Capabilities) == 1 {
			capability = s.loaded.Capabilities[0]
			ok = true
		}
	}
	if !ok {
		return runCapabilityResult{}, fmt.Errorf("unknown capability %q", name)
	}
	mode := stringArg(args, "mode")
	if mode == "" {
		mode = "audit"
	}
	input := mapArg(args, "input")
	if len(input) == 0 {
		input = capabilityInputFromTopLevel(args)
	}
	eval := evaluator.Evaluate(s.loaded, evaluator.Request{
		Tool:       "actlane_run_capability",
		Mode:       mode,
		Capability: capability.Metadata.Name,
		Input:      input,
	})
	evidence := s.buildEvidenceSummary(capability.Metadata.Name, eval)
	s.storeEvidence(evidence)
	adapterExecutions := s.buildAdapterExecutions(capability.Metadata.Name, eval.Next, evidence.ID)
	result := runCapabilityResult{
		Capability:            eval.Capability,
		Mode:                  eval.Mode,
		PolicyDecision:        eval.PolicyDecision,
		Allowed:               eval.Allowed,
		Risk:                  eval.Risk,
		ImpactedScopes:        nonNilStrings(eval.ImpactedScopes),
		RequiredChecks:        nonNilStrings(eval.RequiredChecks),
		RequiredEvidence:      nonNilStrings(eval.RequiredEvidence),
		HumanApprovalRequired: eval.HumanApprovalRequired,
		Stop:                  eval.Stop,
		Reasons:               nonNilStrings(eval.Reasons),
		OriginalInput:         eval.OriginalInput,
		MutatedInput:          eval.MutatedInput,
		PolicyRefs:            nonNilStrings(eval.PolicyRefs),
		ResponsibilityRefs:    nonNilStrings(eval.ResponsibilityRefs),
		DownstreamPlan:        eval.Next,
		AdapterSource:         "MCPBinding",
		AdapterExecutions:     adapterExecutions,
		Evidence:              evidence,
		Execution: executionResult{
			Performed: false,
			Reason:    "adapter executions are recorded but external MCP calls are not executed by this MVP",
		},
	}
	s.storeRun(result)
	return result, nil
}

func (s *Server) getEvidence(args map[string]any) (evidenceLookupResult, error) {
	id := stringArg(args, "id")
	latest, _ := args["latest"].(bool)
	if latest || id == "" {
		id = s.latestEvidenceID
	}
	if id == "" {
		return evidenceLookupResult{}, fmt.Errorf("no evidence has been captured in this MCP session")
	}
	evidence, ok := s.evidenceStore[id]
	if !ok {
		return evidenceLookupResult{}, fmt.Errorf("unknown evidence %q", id)
	}
	return evidenceLookupResult{
		Source:   "evidenceStore",
		Evidence: evidence,
	}, nil
}

func (s *Server) prepareDelivery(args map[string]any) (deliveryResult, error) {
	run, err := s.runForArgs(args)
	if err != nil {
		return deliveryResult{}, err
	}
	return deliveryResult{Delivery: deliveryFromRun(run)}, nil
}

type capabilityView struct {
	Name                string                  `json:"name"`
	Title               string                  `json:"title,omitempty"`
	Description         string                  `json:"description,omitempty"`
	Intent              intentView              `json:"intent"`
	Inputs              any                     `json:"inputs,omitempty"`
	Outputs             any                     `json:"outputs,omitempty"`
	PolicyRef           string                  `json:"policyRef,omitempty"`
	ExecutionRef        string                  `json:"executionRef,omitempty"`
	ResponsibilityRef   string                  `json:"responsibilityRef,omitempty"`
	RuntimeRef          string                  `json:"runtimeRef,omitempty"`
	EvidenceRef         string                  `json:"evidenceRef,omitempty"`
	Policy              capabilityPolicyView    `json:"policy"`
	Responsibility      responsibilityView      `json:"responsibility"`
	RequiredEvidence    []string                `json:"requiredEvidence"`
	DownstreamTools     []downstreamToolView    `json:"downstreamTools"`
	PolicyGateTools     []pack.MCPGeneratedTool `json:"policyGateTools"`
	Runtime             runtimeView             `json:"runtime"`
	RawYAMLIncluded     bool                    `json:"rawYamlIncluded"`
	MutationOrExecution bool                    `json:"mutationOrExecution"`
}

type intentView struct {
	Type         string   `json:"type,omitempty"`
	WhenToUse    []string `json:"whenToUse,omitempty"`
	WhenNotToUse []string `json:"whenNotToUse,omitempty"`
}

type capabilityPolicyView struct {
	Name         string                  `json:"name,omitempty"`
	Confirmation pack.PolicyConfirmation `json:"confirmation,omitempty"`
	ForbidPaths  []string                `json:"forbidPaths,omitempty"`
	Limits       pack.PolicyLimits       `json:"limits,omitempty"`
	Approval     pack.PolicyApproval     `json:"approval,omitempty"`
	Defaults     map[string]any          `json:"defaults,omitempty"`
	BranchPrefix string                  `json:"branchPrefix,omitempty"`
	Audit        pack.PolicyAudit        `json:"audit,omitempty"`
}

type responsibilityView struct {
	Name          string `json:"name,omitempty"`
	HumanBoundary any    `json:"humanBoundary,omitempty"`
	Evidence      any    `json:"evidence,omitempty"`
	Risk          any    `json:"risk,omitempty"`
	Checks        any    `json:"checks,omitempty"`
}

type downstreamToolView struct {
	Binding        string   `json:"binding"`
	Server         string   `json:"server,omitempty"`
	Name           string   `json:"name"`
	Toolset        string   `json:"toolset,omitempty"`
	RequiredScopes []string `json:"requiredScopes,omitempty"`
}

type runtimeView struct {
	Name                  string   `json:"name,omitempty"`
	DefaultMode           string   `json:"defaultMode,omitempty"`
	WorkTypes             []string `json:"workTypes,omitempty"`
	RiskFlags             []string `json:"riskFlags,omitempty"`
	CandidateCapabilities []string `json:"candidateCapabilities,omitempty"`
}

func (s *Server) loadCapability(args map[string]any) (capabilityView, error) {
	name := stringArg(args, "name")
	if name == "" {
		name = stringArg(args, "capability")
	}
	capability, ok := s.capabilityByName(name)
	if !ok {
		if name == "" && len(s.loaded.Capabilities) == 1 {
			capability = s.loaded.Capabilities[0]
			ok = true
		}
	}
	if !ok {
		return capabilityView{}, fmt.Errorf("unknown capability %q", name)
	}
	policy := s.policyForCapability(capability)
	responsibility := s.responsibilityForCapability(capability.Spec.ResponsibilityRef.Name)
	evidence := s.evidenceForCapability(capability.Metadata.Name)
	runtimeProfile := s.runtimeProfileForCapability(capability.Metadata.Name)
	downstreamTools, policyGateTools := s.toolSummariesForCapability(capability.Metadata.Name)
	return capabilityView{
		Name:                capability.Metadata.Name,
		Title:               capability.Metadata.Title,
		Description:         capability.Metadata.Description,
		Intent:              capabilityIntentView(capability.Spec.Intent),
		Inputs:              capabilityInputs(capability),
		Outputs:             capabilityOutputs(capability),
		PolicyRef:           capability.Spec.PolicyRef.Name,
		ExecutionRef:        capability.Spec.ExecutionRef.Name,
		ResponsibilityRef:   capability.Spec.ResponsibilityRef.Name,
		RuntimeRef:          capability.Spec.RuntimeRef.Name,
		EvidenceRef:         capability.Spec.EvidenceRef.Name,
		Policy:              policy,
		Responsibility:      responsibility,
		RequiredEvidence:    nonNilStrings(evidence.Spec.SummaryFields),
		DownstreamTools:     downstreamTools,
		PolicyGateTools:     policyGateTools,
		Runtime:             runtimeProfile,
		RawYAMLIncluded:     false,
		MutationOrExecution: false,
	}, nil
}

func capabilityIntentView(intent pack.CapabilityIntent) intentView {
	return intentView{
		Type:         intent.Type,
		WhenToUse:    nonNilStrings(intent.WhenToUse),
		WhenNotToUse: nonNilStrings(intent.WhenNotToUse),
	}
}

func (s *Server) capabilityByName(name string) (pack.Capability, bool) {
	for _, capability := range s.loaded.Capabilities {
		if capability.Metadata.Name == name {
			return capability, true
		}
	}
	return pack.Capability{}, false
}

func (s *Server) policyForCapability(capability pack.Capability) capabilityPolicyView {
	for _, policy := range s.loaded.Policies {
		if policy.Metadata.Name == capability.Spec.PolicyRef.Name || policyMatchesCapability(policy, capability.Metadata.Name) {
			return capabilityPolicyView{
				Name:         policy.Metadata.Name,
				Confirmation: policy.Spec.Validate.Confirmation,
				ForbidPaths:  nonNilStrings(policy.Spec.Validate.ForbidPaths),
				Limits:       policy.Spec.Validate.Limits,
				Approval:     policy.Spec.Approval,
				Defaults:     policy.Spec.Mutate.Defaults,
				BranchPrefix: policy.Spec.Mutate.Ensure.BranchPrefix,
				Audit:        policy.Spec.Audit,
			}
		}
	}
	return capabilityPolicyView{}
}

func policyMatchesCapability(policy pack.Policy, capability string) bool {
	for _, value := range policy.Spec.Match.Capabilities {
		if value == capability {
			return true
		}
	}
	return false
}

func (s *Server) responsibilityForCapability(name string) responsibilityView {
	for _, responsibility := range s.loaded.Contracts {
		if responsibility.Metadata.Name != name {
			continue
		}
		return responsibilityView{
			Name:          responsibility.Metadata.Name,
			HumanBoundary: responsibility.Spec["humanBoundary"],
			Evidence:      responsibility.Spec["evidence"],
			Risk:          responsibility.Spec["risk"],
			Checks:        responsibility.Spec["checks"],
		}
	}
	return responsibilityView{}
}

func (s *Server) runtimeProfileForCapability(capability string) runtimeView {
	for _, runtimeProfile := range s.loaded.RuntimeProfiles {
		if runtimeProfile.Spec.CapabilityRef.Name != capability {
			continue
		}
		return runtimeView{
			Name:                  runtimeProfile.Metadata.Name,
			DefaultMode:           runtimeProfile.Spec.DefaultMode,
			WorkTypes:             nonNilStrings(runtimeProfile.Spec.WorkTypes),
			RiskFlags:             nonNilStrings(runtimeProfile.Spec.RiskFlags),
			CandidateCapabilities: nonNilStrings(runtimeProfile.Spec.CandidateCapabilities),
		}
	}
	return runtimeView{}
}

func (s *Server) toolSummariesForCapability(capability string) ([]downstreamToolView, []pack.MCPGeneratedTool) {
	var downstream []downstreamToolView
	var policyGate []pack.MCPGeneratedTool
	for _, binding := range s.loaded.MCPBindings {
		if binding.Spec.CapabilityRef.Name != capability {
			continue
		}
		if binding.Spec.Strategy.Handler == "actlane.policy.evaluate" {
			policyGate = append(policyGate, generatedTools(binding)...)
			continue
		}
		for _, tool := range binding.Spec.RequiredTools {
			downstream = append(downstream, downstreamToolView{
				Binding:        binding.Metadata.Name,
				Server:         tool.Server,
				Name:           tool.Name,
				Toolset:        tool.Toolset,
				RequiredScopes: nonNilStrings(tool.RequiredScopes),
			})
		}
	}
	return downstream, policyGate
}

func capabilityInputs(capability pack.Capability) any {
	if len(capability.Spec.Interface.Input) > 0 {
		return capability.Spec.Interface.Input
	}
	if len(capability.Spec.Inputs) > 0 {
		return capability.Spec.Inputs
	}
	return nil
}

func capabilityOutputs(capability pack.Capability) any {
	if len(capability.Spec.Interface.Output) > 0 {
		return capability.Spec.Interface.Output
	}
	if len(capability.Spec.Outputs) > 0 {
		return capability.Spec.Outputs
	}
	return nil
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

func stringArg(args map[string]any, key string) string {
	value, _ := args[key].(string)
	return value
}

func mapArg(args map[string]any, key string) map[string]any {
	value, _ := args[key].(map[string]any)
	if value == nil {
		return map[string]any{}
	}
	return value
}

func capabilityInputFromTopLevel(args map[string]any) map[string]any {
	input := map[string]any{}
	for key, value := range args {
		switch key {
		case "name", "capability", "mode":
			continue
		default:
			input[key] = value
		}
	}
	return input
}

func (s *Server) buildEvidenceSummary(capability string, eval evaluator.Evaluation) evidenceSummary {
	contract := s.evidenceForCapability(capability)
	fields := nonNilStrings(contract.Spec.SummaryFields)
	values := evidenceValues(eval)
	filtered := map[string]any{}
	var missing []string
	for _, field := range fields {
		value, ok := values[field]
		if !ok || isEmptyEvidenceValue(value) {
			missing = append(missing, field)
			continue
		}
		filtered[field] = value
	}
	rawOutput := contract.Spec.RawOutput.Default
	if rawOutput == "" {
		rawOutput = "summary"
	}
	return evidenceSummary{
		ID:                evidenceID(contract, capability, eval),
		Contract:          contract.Metadata.Name,
		SummaryFields:     fields,
		Values:            filtered,
		MissingFields:     missing,
		RawOutput:         rawOutput,
		Redacted:          contract.Spec.Redaction.Secrets || contract.Spec.Redaction.Tokens,
		DeliveryChecklist: nonNilStrings(contract.Spec.DeliveryChecklist),
	}
}

func (s *Server) storeEvidence(evidence evidenceSummary) {
	if evidence.ID == "" {
		return
	}
	s.evidenceStore[evidence.ID] = evidence
	s.latestEvidenceID = evidence.ID
}

func (s *Server) storeRun(run runCapabilityResult) {
	if run.Evidence.ID == "" {
		return
	}
	s.runStore[run.Evidence.ID] = run
	s.latestEvidenceID = run.Evidence.ID
}

func (s *Server) runForArgs(args map[string]any) (runCapabilityResult, error) {
	id := stringArg(args, "evidenceId")
	if id == "" {
		id = stringArg(args, "id")
	}
	latest, _ := args["latest"].(bool)
	if latest || id == "" {
		id = s.latestEvidenceID
	}
	if id == "" {
		return runCapabilityResult{}, fmt.Errorf("no capability run has been captured in this MCP session")
	}
	run, ok := s.runStore[id]
	if !ok {
		return runCapabilityResult{}, fmt.Errorf("unknown capability run for evidence %q", id)
	}
	return run, nil
}

func deliveryFromRun(run runCapabilityResult) deliverySummary {
	checks := append([]string{}, run.RequiredChecks...)
	if len(checks) == 0 {
		checks = stringSliceArg(run.Evidence.Values, "checks_run")
	}
	changed := stringSliceArg(run.MutatedInput, "files")
	if len(changed) == 0 {
		changed = stringSliceArg(run.Evidence.Values, "changed_files")
	}
	return deliverySummary{
		Capability:               run.Capability,
		Summary:                  fmt.Sprintf("Actlane broker prepared %s with policy decision %s.", run.Capability, run.PolicyDecision),
		PolicyDecision:           run.PolicyDecision,
		Allowed:                  run.Allowed,
		Risk:                     run.Risk,
		ResidualRisk:             run.Risk,
		HumanApprovalRequired:    run.HumanApprovalRequired,
		RequiresApproval:         run.HumanApprovalRequired || !run.Allowed,
		Stop:                     run.Stop,
		WhatChanged:              nonNilStrings(changed),
		WhatWasChecked:           nonNilStrings(checks),
		Risky:                    riskyItems(run),
		RequiresHumanResolution:  humanResolutionItems(run),
		EvidenceID:               run.Evidence.ID,
		Evidence:                 run.Evidence,
		AdapterExecutions:        run.AdapterExecutions,
		ExternalExecutionPlanned: len(run.AdapterExecutions) > 0,
		ExternalExecutionDone:    run.Execution.Performed,
	}
}

func riskyItems(run runCapabilityResult) []string {
	items := append([]string{}, run.Reasons...)
	for _, path := range stringSliceArg(run.Evidence.Values, "blocked_paths") {
		items = append(items, "blocked path: "+path)
	}
	return nonNilStrings(items)
}

func humanResolutionItems(run runCapabilityResult) []string {
	if run.Allowed && !run.HumanApprovalRequired {
		return []string{}
	}
	var items []string
	if run.HumanApprovalRequired {
		items = append(items, "human approval required")
	}
	if !run.Allowed {
		items = append(items, "policy decision must be resolved before delivery")
	}
	if run.Stop {
		items = append(items, "agent must stop")
	}
	return items
}

func evidenceValues(eval evaluator.Evaluation) map[string]any {
	return map[string]any{
		"policy_decision": eval.PolicyDecision,
		"changed_files":   stringSliceArg(eval.MutatedInput, "files"),
		"branch":          eval.MutatedInput["branch"],
		"draft_pr_url":    firstNonEmpty(eval.MutatedInput["draft_pr_url"], eval.MutatedInput["draftPrUrl"]),
		"checks_run":      nonNilStrings(eval.RequiredChecks),
		"blocked_paths":   blockedPaths(eval.Reasons),
		"residual_risk":   eval.Risk,
	}
}

func evidenceID(contract pack.EvidenceContract, capability string, eval evaluator.Evaluation) string {
	prefix := contract.Spec.EvidenceID.Prefix
	if prefix == "" {
		prefix = capability
	}
	data, _ := json.Marshal(map[string]any{
		"capability": capability,
		"decision":   eval.PolicyDecision,
		"input":      eval.MutatedInput,
	})
	sum := sha256.Sum256(data)
	return prefix + "-" + hex.EncodeToString(sum[:])[:12]
}

func firstNonEmpty(values ...any) any {
	for _, value := range values {
		if !isEmptyEvidenceValue(value) {
			return value
		}
	}
	return nil
}

func blockedPaths(reasons []string) []string {
	var paths []string
	for _, reason := range reasons {
		const prefix = "file is forbidden: "
		if strings.HasPrefix(reason, prefix) {
			paths = append(paths, strings.TrimPrefix(reason, prefix))
		}
	}
	return paths
}

func isEmptyEvidenceValue(value any) bool {
	if value == nil {
		return true
	}
	switch typed := value.(type) {
	case string:
		return typed == ""
	case []string:
		return len(typed) == 0
	case []any:
		return len(typed) == 0
	default:
		return false
	}
}

func (s *Server) buildAdapterExecutions(capability string, calls []evaluator.NextCall, evidenceID string) []adapterExecution {
	if len(calls) == 0 {
		return nil
	}
	var executions []adapterExecution
	for index, call := range calls {
		binding, tool := s.bindingToolForCall(capability, call)
		executions = append(executions, adapterExecution{
			ID:             fmt.Sprintf("%s-adapter-%d", evidenceID, index+1),
			Binding:        binding.Metadata.Name,
			Server:         call.Server,
			Tool:           call.Tool,
			Status:         "planned",
			Performed:      false,
			Reason:         "external MCP adapter execution is disabled in this MVP",
			RequiredScopes: nonNilStrings(tool.RequiredScopes),
			EvidenceID:     evidenceID,
		})
	}
	return executions
}

func (s *Server) bindingToolForCall(capability string, call evaluator.NextCall) (pack.MCPBinding, pack.MCPToolBinding) {
	for _, binding := range s.loaded.MCPBindings {
		if binding.Spec.CapabilityRef.Name != capability || binding.Spec.Strategy.Handler == "actlane.policy.evaluate" {
			continue
		}
		for _, tool := range binding.Spec.RequiredTools {
			if tool.Server == call.Server && tool.Server+"_"+tool.Name == call.Tool {
				return binding, tool
			}
		}
	}
	return pack.MCPBinding{}, pack.MCPToolBinding{}
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
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

func runCapabilityInputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name":       map[string]any{"type": "string"},
			"capability": map[string]any{"type": "string"},
			"mode":       map[string]any{"type": "string", "enum": []string{"audit", "enforce"}},
			"input": map[string]any{
				"type":                 "object",
				"additionalProperties": true,
			},
		},
		"additionalProperties": true,
	}
}

func getEvidenceInputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id":     map[string]any{"type": "string"},
			"latest": map[string]any{"type": "boolean"},
		},
		"additionalProperties": false,
	}
}

func prepareDeliveryInputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id":         map[string]any{"type": "string"},
			"evidenceId": map[string]any{"type": "string"},
			"latest":     map[string]any{"type": "boolean"},
		},
		"additionalProperties": false,
	}
}

func loadCapabilityInputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name":       map[string]any{"type": "string"},
			"capability": map[string]any{"type": "string"},
		},
		"additionalProperties": false,
	}
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
