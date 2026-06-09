package authoringmcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/actlane/actlane/packages/cli/internal/generator/profile"
	"github.com/actlane/actlane/packages/cli/internal/pack"
	"github.com/actlane/actlane/packages/cli/internal/scaffold"
)

const serverVersion = "0.3.0-alpha.16"

type Server struct {
	packDir string
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

func New(packDir string) *Server {
	if packDir == "" {
		packDir = "."
	}
	return &Server{packDir: packDir}
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
		if err := writeResponse(w, s.handle(req)); err != nil {
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
				"name":    "actlane-pack-author",
				"version": serverVersion,
			},
		})
	case "tools/list":
		return responseOK(req.ID, map[string]any{"tools": tools()})
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

func tools() []map[string]any {
	names := []struct {
		name        string
		description string
	}{
		{"actlane_pack_inspect", "Inspect an Actlane pack source tree without modifying files."},
		{"actlane_pack_validate", "Validate an Actlane pack and return structured diagnostics."},
		{"actlane_pack_plan_change", "Plan source contract files for a new pack or capability without writing files."},
		{"actlane_pack_apply_change", "Apply explicitly confirmed source contract files from a prior plan."},
		{"actlane_pack_generate_preview", "Preview generated target files by delegating to the Actlane generator in a temporary directory."},
		{"actlane_pack_explain_errors", "Explain pack load or validation errors in human-readable form."},
	}
	result := make([]map[string]any, 0, len(names))
	for _, tool := range names {
		result = append(result, map[string]any{
			"name":        tool.name,
			"description": tool.description,
			"inputSchema": map[string]any{
				"type":                 "object",
				"additionalProperties": true,
			},
		})
	}
	return result
}

func (s *Server) callTool(name string, args map[string]any) (toolResult, error) {
	switch name {
	case "actlane_pack_inspect":
		return jsonToolResult(s.inspect(args), false)
	case "actlane_pack_validate":
		result := s.validate(args)
		return jsonToolResult(result, !result.Valid)
	case "actlane_pack_plan_change":
		return jsonToolResult(planChange(args), false)
	case "actlane_pack_apply_change":
		result := s.applyChange(args)
		return jsonToolResult(result, !result.Applied)
	case "actlane_pack_generate_preview":
		result := s.generatePreview(args)
		return jsonToolResult(result, !result.Valid)
	case "actlane_pack_explain_errors":
		return jsonToolResult(s.explainErrors(args), false)
	default:
		return toolResult{}, fmt.Errorf("unknown tool %q", name)
	}
}

type inspectResult struct {
	Pack              packSummary    `json:"pack"`
	Counts            map[string]int `json:"counts"`
	Capabilities      []string       `json:"capabilities"`
	Policies          []string       `json:"policies"`
	MCPBindings       []string       `json:"mcpBindings"`
	Skills            []string       `json:"skills"`
	Commands          []string       `json:"commands"`
	Agents            []string       `json:"agents"`
	Responsibilities  []string       `json:"responsibilities"`
	TargetProfiles    []string       `json:"targetProfiles"`
	SupportedTargets  []string       `json:"supportedTargets"`
	Validation        validationInfo `json:"validation"`
	SourceOfTruth     string         `json:"sourceOfTruth"`
	MutationPermitted bool           `json:"mutationPermitted"`
}

type packSummary struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Description  string `json:"description,omitempty"`
	Root         string `json:"root"`
	ManifestPath string `json:"manifestPath"`
}

type validationInfo struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

func (s *Server) inspect(args map[string]any) inspectResult {
	loaded, err := pack.Load(packDirArg(s.packDir, args))
	if err != nil {
		return inspectResult{
			Validation:        validationInfo{Valid: false, Error: err.Error()},
			SourceOfTruth:     "Actlane YAML contracts and adjacent source files",
			MutationPermitted: false,
		}
	}
	validation := validationInfo{Valid: true}
	if err := pack.Validate(loaded); err != nil {
		validation = validationInfo{Valid: false, Error: err.Error()}
	}
	return inspectResult{
		Pack: packSummary{
			Name:         loaded.Manifest.Metadata.Name,
			Version:      loaded.Manifest.Metadata.Version,
			Description:  loaded.Manifest.Metadata.Description,
			Root:         loaded.Root,
			ManifestPath: loaded.ManifestPath,
		},
		Counts: map[string]int{
			"capabilities":     len(loaded.Capabilities),
			"policies":         len(loaded.Policies),
			"mcpBindings":      len(loaded.MCPBindings),
			"skills":           len(loaded.Skills),
			"commands":         len(loaded.Commands),
			"agents":           len(loaded.Agents),
			"responsibilities": len(loaded.Contracts),
			"targetProfiles":   len(loaded.TargetProfiles),
		},
		Capabilities:      capabilityNames(loaded),
		Policies:          policyNames(loaded),
		MCPBindings:       mcpBindingNames(loaded),
		Skills:            skillNames(loaded),
		Commands:          commandNames(loaded),
		Agents:            agentNames(loaded),
		Responsibilities:  responsibilityNames(loaded),
		TargetProfiles:    targetProfileNames(loaded),
		SupportedTargets:  supportedTargets(loaded),
		Validation:        validation,
		SourceOfTruth:     "Actlane YAML contracts and adjacent source files",
		MutationPermitted: false,
	}
}

type validateResult struct {
	Valid       bool     `json:"valid"`
	Pack        string   `json:"pack,omitempty"`
	Diagnostics []string `json:"diagnostics,omitempty"`
	Next        []string `json:"next,omitempty"`
}

func (s *Server) validate(args map[string]any) validateResult {
	loaded, err := pack.Load(packDirArg(s.packDir, args))
	if err != nil {
		return validateResult{Valid: false, Diagnostics: []string{err.Error()}, Next: []string{"Fix pack manifest or referenced source file paths."}}
	}
	if err := pack.Validate(loaded); err != nil {
		return validateResult{Valid: false, Pack: loaded.Manifest.Metadata.Name, Diagnostics: []string{err.Error()}, Next: []string{"Move misplaced fields to their owning Actlane contract.", "Run actlane validate <pack> after editing."}}
	}
	return validateResult{Valid: true, Pack: loaded.Manifest.Metadata.Name, Next: []string{"Use actlane generate <pack> --target <target> or actlane_pack_generate_preview."}}
}

type planChangeResult struct {
	MutationPermitted bool              `json:"mutationPermitted"`
	RequiresApply     bool              `json:"requiresApply"`
	SourceOfTruth     string            `json:"sourceOfTruth"`
	Files             []scaffold.File   `json:"files"`
	Notes             []string          `json:"notes"`
	Boundaries        map[string]string `json:"boundaries"`
}

func planChange(args map[string]any) planChangeResult {
	name := scaffold.CleanName(stringArg(args, "name", "new-capability"))
	targets := stringSliceArg(args, "targets")
	if len(targets) == 0 {
		targets = []string{"codex"}
	}
	contracts := stringSliceArg(args, "contracts")
	files, err := scaffold.Plan(scaffold.Options{Name: name, Targets: targets, Contracts: contracts})
	if err != nil {
		return planChangeResult{
			MutationPermitted: false,
			RequiresApply:     false,
			SourceOfTruth:     "Actlane YAML contracts and adjacent source files",
			Notes:             []string{err.Error()},
			Boundaries:        contractBoundaries(),
		}
	}
	return planChangeResult{
		MutationPermitted: false,
		RequiresApply:     true,
		SourceOfTruth:     "Actlane YAML contracts and adjacent source files",
		Files:             files,
		Notes: []string{
			"This tool returns a proposal only; it does not write files.",
			"Generated target output must be produced by the normal Actlane generator.",
			"Review and confirm before applying any proposed source files.",
		},
		Boundaries: contractBoundaries(),
	}
}

func contractBoundaries() map[string]string {
	return map[string]string{
		"Capability":             "safe action contract",
		"ToolCallPolicy":         "safety behavior",
		"MCPBinding":             "real runtime tools",
		"SkillContract":          "portable skill directory",
		"CommandContract":        "portable command entrypoint",
		"AgentContract":          "portable agent role and activation",
		"ResponsibilityContract": "risk, evidence, and approval semantics",
		"TargetProfile":          "target runtime file layout",
	}
}

type applyChangeResult struct {
	Applied     bool     `json:"applied"`
	Pack        string   `json:"pack"`
	Written     []string `json:"written,omitempty"`
	Skipped     []string `json:"skipped,omitempty"`
	Diagnostics []string `json:"diagnostics,omitempty"`
	Next        []string `json:"next,omitempty"`
}

func (s *Server) applyChange(args map[string]any) applyChangeResult {
	packDir := packDirArg(s.packDir, args)
	if !boolArg(args, "confirmed", false) {
		return applyChangeResult{
			Applied:     false,
			Pack:        packDir,
			Diagnostics: []string{"confirmed must be true before writing source files"},
			Next:        []string{"Show the proposed files to the user, get explicit confirmation, then call actlane_pack_apply_change with confirmed=true."},
		}
	}
	files, err := plannedFilesArg(args)
	if err != nil {
		return applyChangeResult{Applied: false, Pack: packDir, Diagnostics: []string{err.Error()}}
	}
	overwrite := boolArg(args, "overwrite", false)
	written, skipped, err := scaffold.Write(packDir, files, overwrite)
	if len(skipped) > 0 {
		return applyChangeResult{
			Applied:     false,
			Pack:        packDir,
			Skipped:     sorted(skipped),
			Diagnostics: []string{"one or more files already exist; pass overwrite=true only after explicit confirmation"},
		}
	}
	if err != nil {
		return applyChangeResult{Applied: false, Pack: packDir, Written: written, Diagnostics: []string{err.Error()}}
	}
	applied := len(written) > 0
	return applyChangeResult{
		Applied: applied,
		Pack:    packDir,
		Written: sorted(written),
		Next: []string{
			"Run actlane_pack_validate.",
			"Run actlane_pack_generate_preview for the target before writing generated output.",
		},
	}
}

type generatePreviewResult struct {
	Valid       bool                   `json:"valid"`
	Pack        string                 `json:"pack,omitempty"`
	Target      string                 `json:"target,omitempty"`
	Files       []generatedPreviewFile `json:"files,omitempty"`
	Diagnostics []string               `json:"diagnostics,omitempty"`
}

type generatedPreviewFile struct {
	Path  string `json:"path"`
	Bytes int    `json:"bytes"`
}

func (s *Server) generatePreview(args map[string]any) generatePreviewResult {
	target := stringArg(args, "target", "")
	if target == "" {
		return generatePreviewResult{Valid: false, Diagnostics: []string{"target is required"}}
	}
	loaded, err := pack.Load(packDirArg(s.packDir, args))
	if err != nil {
		return generatePreviewResult{Valid: false, Target: target, Diagnostics: []string{err.Error()}}
	}
	if err := pack.Validate(loaded); err != nil {
		return generatePreviewResult{Valid: false, Pack: loaded.Manifest.Metadata.Name, Target: target, Diagnostics: []string{err.Error()}}
	}
	tempDir, err := os.MkdirTemp("", "actlane-authoring-preview-*")
	if err != nil {
		return generatePreviewResult{Valid: false, Pack: loaded.Manifest.Metadata.Name, Target: target, Diagnostics: []string{err.Error()}}
	}
	defer os.RemoveAll(tempDir)
	result, err := profile.Generate(loaded, profile.Options{Target: target, OutDir: tempDir})
	if err != nil {
		return generatePreviewResult{Valid: false, Pack: loaded.Manifest.Metadata.Name, Target: target, Diagnostics: []string{err.Error()}}
	}
	files := make([]generatedPreviewFile, 0, len(result.Files))
	for path, data := range result.Files {
		files = append(files, generatedPreviewFile{Path: path, Bytes: len(data)})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return generatePreviewResult{Valid: true, Pack: loaded.Manifest.Metadata.Name, Target: target, Files: files}
}

type explainErrorsResult struct {
	Valid       bool     `json:"valid"`
	Diagnostics []string `json:"diagnostics,omitempty"`
	Explanation []string `json:"explanation"`
	Next        []string `json:"next"`
}

func (s *Server) explainErrors(args map[string]any) explainErrorsResult {
	validation := s.validate(args)
	if validation.Valid {
		return explainErrorsResult{
			Valid:       true,
			Explanation: []string{"Pack loads and validates successfully."},
			Next:        []string{"Run actlane_pack_generate_preview for a target or actlane generate <pack> --target <target>."},
		}
	}
	return explainErrorsResult{
		Valid:       false,
		Diagnostics: validation.Diagnostics,
		Explanation: []string{
			"Actlane pack authoring errors usually mean one source contract owns fields that belong to another contract, or actlane.yaml references a missing source file.",
			"Keep runtime tools in MCPBinding, safety in ToolCallPolicy, target file layout in TargetProfile, and portable skill content in SkillContract.",
		},
		Next: validation.Next,
	}
}

func jsonToolResult(value any, isError bool) (toolResult, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return toolResult{}, err
	}
	return toolResult{
		IsError: isError,
		Content: []toolContent{{
			Type: "text",
			Text: string(data),
		}},
	}, nil
}

func packDirArg(defaultPackDir string, args map[string]any) string {
	return stringArg(args, "pack", defaultPackDir)
}

func stringArg(args map[string]any, key, fallback string) string {
	if args == nil {
		return fallback
	}
	value, ok := args[key]
	if !ok {
		return fallback
	}
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return fallback
	}
	return strings.TrimSpace(text)
}

func stringSliceArg(args map[string]any, key string) []string {
	if args == nil {
		return nil
	}
	raw, ok := args[key]
	if !ok {
		return nil
	}
	values, ok := raw.([]any)
	if !ok {
		return nil
	}
	var result []string
	for _, value := range values {
		text, ok := value.(string)
		if ok && strings.TrimSpace(text) != "" {
			result = append(result, strings.TrimSpace(text))
		}
	}
	return result
}

func boolArg(args map[string]any, key string, fallback bool) bool {
	if args == nil {
		return fallback
	}
	value, ok := args[key]
	if !ok {
		return fallback
	}
	boolean, ok := value.(bool)
	if !ok {
		return fallback
	}
	return boolean
}

func plannedFilesArg(args map[string]any) ([]scaffold.File, error) {
	raw, ok := args["files"]
	if !ok {
		return nil, fmt.Errorf("files are required")
	}
	values, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("files must be an array")
	}
	files := make([]scaffold.File, 0, len(values))
	for _, value := range values {
		object, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("each file must be an object")
		}
		path, _ := object["path"].(string)
		content, _ := object["content"].(string)
		if strings.TrimSpace(path) == "" {
			return nil, fmt.Errorf("each file must include path")
		}
		files = append(files, scaffold.File{Path: path, Content: content})
	}
	return files, nil
}

func capabilityNames(loaded *pack.LoadedPack) []string {
	values := make([]string, 0, len(loaded.Capabilities))
	for _, item := range loaded.Capabilities {
		values = append(values, item.Metadata.Name)
	}
	return sorted(values)
}

func policyNames(loaded *pack.LoadedPack) []string {
	values := make([]string, 0, len(loaded.Policies))
	for _, item := range loaded.Policies {
		values = append(values, item.Metadata.Name)
	}
	return sorted(values)
}

func mcpBindingNames(loaded *pack.LoadedPack) []string {
	values := make([]string, 0, len(loaded.MCPBindings))
	for _, item := range loaded.MCPBindings {
		values = append(values, item.Metadata.Name)
	}
	return sorted(values)
}

func skillNames(loaded *pack.LoadedPack) []string {
	values := make([]string, 0, len(loaded.Skills))
	for _, item := range loaded.Skills {
		values = append(values, item.Metadata.Name)
	}
	return sorted(values)
}

func commandNames(loaded *pack.LoadedPack) []string {
	values := make([]string, 0, len(loaded.Commands))
	for _, item := range loaded.Commands {
		values = append(values, item.Metadata.Name)
	}
	return sorted(values)
}

func agentNames(loaded *pack.LoadedPack) []string {
	values := make([]string, 0, len(loaded.Agents))
	for _, item := range loaded.Agents {
		values = append(values, item.Metadata.Name)
	}
	return sorted(values)
}

func responsibilityNames(loaded *pack.LoadedPack) []string {
	values := make([]string, 0, len(loaded.Contracts))
	for _, item := range loaded.Contracts {
		values = append(values, item.Metadata.Name)
	}
	return sorted(values)
}

func targetProfileNames(loaded *pack.LoadedPack) []string {
	values := make([]string, 0, len(loaded.TargetProfiles))
	for _, item := range loaded.TargetProfiles {
		values = append(values, item.Metadata.Name)
	}
	return sorted(values)
}

func supportedTargets(loaded *pack.LoadedPack) []string {
	values := make([]string, 0, len(loaded.TargetProfiles))
	for _, item := range loaded.TargetProfiles {
		if item.Spec.Target != "" {
			values = append(values, item.Spec.Target)
		}
	}
	return sorted(values)
}

func sorted(values []string) []string {
	sort.Strings(values)
	return values
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
