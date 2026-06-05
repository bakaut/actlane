package pack

type Document struct {
	Schema     string   `yaml:"$schema"`
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
}

type Metadata struct {
	Name        string            `yaml:"name"`
	Title       string            `yaml:"title"`
	Version     string            `yaml:"version"`
	Description string            `yaml:"description"`
	Labels      map[string]string `yaml:"labels"`
}

type CapabilityPack struct {
	Document `yaml:",inline"`
	Spec     PackSpec `yaml:"spec"`
}

type PackSpec struct {
	Capabilities    []string       `yaml:"capabilities"`
	Skills          []string       `yaml:"skills"`
	Commands        []string       `yaml:"commands"`
	Agents          []string       `yaml:"agents"`
	Contracts       []string       `yaml:"contracts"`
	RuntimeProfiles []string       `yaml:"runtimeProfiles"`
	Evidence        []string       `yaml:"evidence"`
	Policies        []string       `yaml:"policies"`
	MCPBindings     []string       `yaml:"mcpBindings"`
	Targets         []string       `yaml:"targets"`
	TargetProfiles  []string       `yaml:"targetProfiles"`
	Guidance        GuidanceSpec   `yaml:"guidance"`
	AdoptionProfile map[string]any `yaml:"adoptionProfile"`
}

type GuidanceSpec struct {
	Sources []GuidanceSource `yaml:"sources"`
	Compose GuidanceCompose  `yaml:"compose"`
}

type GuidanceSource struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
	Role string `yaml:"role"`
}

type GuidanceCompose struct {
	Enabled  bool     `yaml:"enabled"`
	Output   string   `yaml:"output"`
	Strategy string   `yaml:"strategy"`
	Order    []string `yaml:"order"`
}

type Capability struct {
	Document `yaml:",inline"`
	Spec     CapabilitySpec `yaml:"spec"`
	Path     string         `yaml:"-"`
	Raw      []byte         `yaml:"-"`
}

type CapabilitySpec struct {
	Intent            CapabilityIntent             `yaml:"intent"`
	Interface         CapabilityInterface          `yaml:"interface"`
	PolicyRef         LocalRef                     `yaml:"policyRef"`
	ExecutionRef      LocalRef                     `yaml:"executionRef"`
	ResponsibilityRef LocalRef                     `yaml:"responsibilityRef"`
	RuntimeRef        LocalRef                     `yaml:"runtimeRef"`
	EvidenceRef       LocalRef                     `yaml:"evidenceRef"`
	WorkflowHints     []WorkflowHint               `yaml:"workflowHints"`
	Projections       CapabilityProjections        `yaml:"projections"`
	Reporting         map[string]bool              `yaml:"reporting"`
	WhenToUse         string                       `yaml:"whenToUse"`
	Targets           []string                     `yaml:"targets"`
	Inputs            map[string]Field             `yaml:"inputs"`
	Outputs           map[string]Field             `yaml:"outputs"`
	Policies          []string                     `yaml:"policies"`
	ToolFlow          []ToolFlowStep               `yaml:"toolFlow"`
	MCP               MCPSpec                      `yaml:"mcp"`
	Profiles          map[string]CapabilityProfile `yaml:"profiles"`
	Extra             map[string]any               `yaml:",inline"`
}

type Field struct {
	Type     string `yaml:"type"`
	Items    *Field `yaml:"items"`
	Required bool   `yaml:"required"`
}

type CapabilityIntent struct {
	Type         string   `yaml:"type"`
	WhenToUse    []string `yaml:"whenToUse"`
	WhenNotToUse []string `yaml:"whenNotToUse"`
}

type CapabilityInterface struct {
	Input  map[string]any `yaml:"input"`
	Output map[string]any `yaml:"output"`
}

type LocalRef struct {
	Name string `yaml:"name"`
}

type WorkflowHint struct {
	Step    string `yaml:"step"`
	Purpose string `yaml:"purpose"`
}

type CapabilityProjections struct {
	MCP      CapabilityMCPProjection      `yaml:"mcp"`
	Codex    CapabilityCodexProjection    `yaml:"codex"`
	OpenCode CapabilityOpenCodeProjection `yaml:"opencode"`
}

type CapabilityMCPProjection struct {
	Enabled  bool   `yaml:"enabled"`
	ToolName string `yaml:"toolName"`
}

type CapabilityOpenCodeProjection struct {
	Enabled bool   `yaml:"enabled"`
	Command string `yaml:"command"`
	Agent   string `yaml:"agent"`
}

type CapabilityCodexProjection struct {
	Enabled bool   `yaml:"enabled"`
	Skill   string `yaml:"skill"`
}

type ToolFlowStep struct {
	Tool    string `yaml:"tool"`
	Purpose string `yaml:"purpose"`
}

type MCPSpec struct {
	Servers []MCPServerBinding `yaml:"servers"`
	Tools   []MCPToolBinding   `yaml:"tools"`
	Prompts []MCPPrompt        `yaml:"prompts"`
}

type MCPServerBinding struct {
	Name    string            `yaml:"name"`
	Source  string            `yaml:"source"`
	Type    string            `yaml:"type"`
	Command []string          `yaml:"command"`
	Env     map[string]string `yaml:"environment"`
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers"`
	OAuth   any               `yaml:"oauth"`
	Timeout int               `yaml:"timeout"`
	Enabled *bool             `yaml:"enabled"`
}

type MCPToolBinding struct {
	Name           string   `yaml:"name" json:"name"`
	Server         string   `yaml:"server" json:"server"`
	Toolset        string   `yaml:"toolset" json:"toolset"`
	Description    string   `yaml:"description" json:"description,omitempty"`
	RequiredScopes []string `yaml:"requiredScopes" json:"requiredScopes,omitempty"`
}

type MCPPrompt struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type CapabilityProfile struct {
	Files  []GeneratedFile `yaml:"files"`
	Config map[string]any  `yaml:"config"`
}

type GeneratedFile struct {
	Path            string `yaml:"path"`
	Source          string `yaml:"source"`
	SkillContract   string `yaml:"skillContract"`
	CommandContract string `yaml:"commandContract"`
	AgentContract   string `yaml:"agentContract"`
	Content         string `yaml:"content"`
}

type SkillContract struct {
	Document `yaml:",inline"`
	Spec     SkillContractSpec `yaml:"spec"`
	Path     string            `yaml:"-"`
	Raw      []byte            `yaml:"-"`
}

type SkillContractSpec struct {
	Body       string          `yaml:"body"`
	BodySource string          `yaml:"bodySource"`
	Scripts    []SkillResource `yaml:"scripts"`
	References []SkillResource `yaml:"references"`
	Assets     []SkillResource `yaml:"assets"`
}

type SkillResource struct {
	Source string `yaml:"source"`
	Path   string `yaml:"path"`
}

type CommandContract struct {
	Document `yaml:",inline"`
	Spec     CommandContractSpec `yaml:"spec"`
	Path     string              `yaml:"-"`
	Raw      []byte              `yaml:"-"`
}

type CommandContractSpec struct {
	Scope         string                   `yaml:"scope"`
	Invocation    CommandInvocation        `yaml:"invocation"`
	CapabilityRef LocalRef                 `yaml:"capabilityRef"`
	SkillRef      CommandPathRef           `yaml:"skillRef"`
	AgentRef      CommandAgentRef          `yaml:"agentRef"`
	Arguments     CommandArguments         `yaml:"arguments"`
	Prompt        CommandPrompt            `yaml:"prompt"`
	Output        CommandOutput            `yaml:"output"`
	Safety        CommandSafety            `yaml:"safety"`
	Projections   map[string]CommandTarget `yaml:"projections"`
}

type CommandInvocation struct {
	Slash   string   `yaml:"slash"`
	Aliases []string `yaml:"aliases"`
}

type CommandPathRef struct {
	Path string `yaml:"path"`
}

type CommandAgentRef struct {
	Name     string `yaml:"name"`
	Optional bool   `yaml:"optional"`
}

type CommandArguments struct {
	Mode        string `yaml:"mode"`
	Placeholder string `yaml:"placeholder"`
	Description string `yaml:"description"`
}

type CommandPrompt struct {
	Template string `yaml:"template"`
}

type CommandOutput struct {
	Expected []string `yaml:"expected"`
}

type CommandSafety struct {
	RequirePolicy         bool `yaml:"requirePolicy"`
	RequireConfirmation   bool `yaml:"requireConfirmation"`
	DoNotBypassCapability bool `yaml:"doNotBypassCapability"`
}

type CommandTarget struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
	Reason  string `yaml:"reason"`
}

type AgentContract struct {
	Document `yaml:",inline"`
	Spec     AgentContractSpec `yaml:"spec"`
	Path     string            `yaml:"-"`
	Raw      []byte            `yaml:"-"`
}

type AgentContractSpec struct {
	Scope        string                 `yaml:"scope"`
	Mode         string                 `yaml:"mode"`
	Role         AgentRole              `yaml:"role"`
	Activation   AgentActivation        `yaml:"activation"`
	Capabilities AgentCapabilityRefs    `yaml:"capabilities"`
	Skills       AgentSkillRefs         `yaml:"skills"`
	Tools        AgentTools             `yaml:"tools"`
	Permissions  map[string]string      `yaml:"permissions"`
	Output       AgentOutput            `yaml:"output"`
	Projections  map[string]AgentTarget `yaml:"projections"`
}

type AgentRole struct {
	Summary string `yaml:"summary"`
}

type AgentActivation struct {
	WhenToUse []string `yaml:"whenToUse"`
}

type AgentCapabilityRefs struct {
	Allowed []string `yaml:"allowed"`
}

type AgentSkillRefs struct {
	Allowed []string `yaml:"allowed"`
}

type AgentTools struct {
	Strategy    string           `yaml:"strategy"`
	RawMCPTools AgentRawMCPTools `yaml:"rawMcpTools"`
}

type AgentRawMCPTools struct {
	Default string `yaml:"default"`
}

type AgentOutput struct {
	MustInclude []string `yaml:"mustInclude"`
}

type AgentTarget struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
	Reason  string `yaml:"reason"`
}

type TargetProfile struct {
	Document `yaml:",inline"`
	Spec     TargetProfileSpec `yaml:"spec"`
	Path     string            `yaml:"-"`
	Raw      []byte            `yaml:"-"`
}

type TargetProfileSpec struct {
	Target     string                  `yaml:"target"`
	Scope      string                  `yaml:"scope"`
	Output     TargetProfileOutput     `yaml:"output"`
	Transforms TargetProfileTransforms `yaml:"transforms"`
	Install    TargetProfileInstall    `yaml:"install"`
	Generate   TargetProfileGenerate   `yaml:"generate"`
	Codex      TargetProfileCodex      `yaml:"codex"`
	OpenCode   TargetProfileOpenCode   `yaml:"opencode"`
}

type TargetProfileOutput struct {
	Root   string `yaml:"root"`
	Config string `yaml:"config"`
}

type TargetProfileInstall struct {
	Mode                 string            `yaml:"mode"`
	Scope                string            `yaml:"scope"`
	RequireExplicitApply bool              `yaml:"requireExplicitApply"`
	RequireDiffPreview   bool              `yaml:"requireDiffPreview"`
	ProjectPaths         map[string]string `yaml:"projectPaths"`
}

type TargetProfileGenerate struct {
	Config       bool `yaml:"config"`
	Instructions bool `yaml:"instructions"`
	Agents       bool `yaml:"agents"`
	Commands     bool `yaml:"commands"`
	Skills       bool `yaml:"skills"`
	MCP          bool `yaml:"mcp"`
}

type TargetProfileOpenCode struct {
	Config TargetProfileOpenCodeConfig `yaml:"config"`
	Files  []TargetProfileFile         `yaml:"files"`
}

type TargetProfileCodex struct {
	Config TargetProfileCodexConfig `yaml:"config"`
	Files  []TargetProfileFile      `yaml:"files"`
}

type TargetProfileCodexConfig struct {
	Filename string `yaml:"filename"`
}

type TargetProfileOpenCodeConfig struct {
	Filename     string                       `yaml:"filename"`
	Schema       string                       `yaml:"schema"`
	Instructions []string                     `yaml:"instructions"`
	Permission   map[string]string            `yaml:"permission"`
	MCP          TargetProfileMCPServerConfig `yaml:"mcp"`
}

type TargetProfileMCPServerConfig struct {
	ServerName string   `yaml:"serverName"`
	Type       string   `yaml:"type"`
	Command    []string `yaml:"command"`
}

type TargetProfileFile struct {
	TargetPath      string `yaml:"targetPath"`
	GeneratedPath   string `yaml:"generatedPath"`
	Source          string `yaml:"source"`
	SkillContract   string `yaml:"skillContract"`
	CommandContract string `yaml:"commandContract"`
	AgentContract   string `yaml:"agentContract"`
	Owned           bool   `yaml:"owned"`
	OwnedBlock      bool   `yaml:"ownedBlock"`
	MarkerStyle     string `yaml:"markerStyle"`
}

type TargetProfileTransforms struct {
	MCP TargetProfileMCPTransform `yaml:"mcp"`
}

type TargetProfileMCPTransform struct {
	Enabled   bool   `yaml:"enabled"`
	ConfigKey string `yaml:"configKey"`
}

type Policy struct {
	Document `yaml:",inline"`
	Spec     PolicySpec `yaml:"spec"`
	Path     string     `yaml:"-"`
	Raw      []byte     `yaml:"-"`
}

type PolicySpec struct {
	Allow            []map[string]any `yaml:"allow"`
	Deny             []PolicyDeny     `yaml:"deny"`
	Mutate           PolicyMutateSpec `yaml:"mutate"`
	RequiresApproval []map[string]any `yaml:"requiresApproval"`
	Match            PolicyMatch      `yaml:"match"`
	Validate         PolicyValidate   `yaml:"validate"`
	Approval         PolicyApproval   `yaml:"approval"`
	Audit            PolicyAudit      `yaml:"audit"`
}

type PolicyMatch struct {
	Capabilities []string `yaml:"capabilities"`
}

type PolicyDeny struct {
	Reason       string   `yaml:"reason"`
	Paths        []string `yaml:"paths"`
	MaxFiles     int      `yaml:"maxFiles"`
	MaxDiffBytes int      `yaml:"maxDiffBytes"`
}

type PolicyMutateSpec struct {
	Defaults map[string]any `yaml:"defaults"`
	Ensure   PolicyEnsure   `yaml:"ensure"`
}

type PolicyEnsure struct {
	BranchPrefix string `yaml:"branchPrefix"`
}

type PolicyValidate struct {
	Confirmation  PolicyConfirmation `yaml:"confirmation"`
	RepoAllowlist []string           `yaml:"repoAllowlist"`
	ForbidPaths   []string           `yaml:"forbidPaths"`
	Limits        PolicyLimits       `yaml:"limits"`
}

type PolicyConfirmation struct {
	Field  string `yaml:"field" json:"field"`
	MustBe any    `yaml:"mustBe" json:"mustBe"`
}

type PolicyLimits struct {
	MaxFiles  int `yaml:"maxFiles" json:"maxFiles"`
	MaxDiffKB int `yaml:"maxDiffKb" json:"maxDiffKb"`
}

type PolicyApproval struct {
	Required bool   `yaml:"required" json:"required"`
	Reason   string `yaml:"reason" json:"reason"`
}

type PolicyAudit struct {
	Level   string   `yaml:"level" json:"level"`
	Include []string `yaml:"include" json:"include"`
}

type ResponsibilityContract struct {
	Document `yaml:",inline"`
	Spec     map[string]any `yaml:"spec"`
	Path     string         `yaml:"-"`
	Raw      []byte         `yaml:"-"`
}

type RuntimeProfile struct {
	Document `yaml:",inline"`
	Spec     RuntimeProfileSpec `yaml:"spec"`
	Path     string             `yaml:"-"`
	Raw      []byte             `yaml:"-"`
}

type RuntimeProfileSpec struct {
	CapabilityRef          LocalRef                   `yaml:"capabilityRef"`
	DefaultMode            string                     `yaml:"defaultMode"`
	WorkTypes              []string                   `yaml:"workTypes"`
	RiskFlags              []string                   `yaml:"riskFlags"`
	TechHints              []string                   `yaml:"techHints"`
	CandidateCapabilities  []string                   `yaml:"candidateCapabilities"`
	HighRisk               RuntimeHighRisk            `yaml:"highRisk"`
	Recommendations        RuntimeRecommendations     `yaml:"recommendations"`
	ClassificationKeywords RuntimeClassificationHints `yaml:"classificationKeywords"`
}

type RuntimeHighRisk struct {
	Mode                 string   `yaml:"mode"`
	RequireHumanBoundary bool     `yaml:"requireHumanBoundary"`
	Flags                []string `yaml:"flags"`
}

type RuntimeRecommendations struct {
	NextStep              string `yaml:"nextStep"`
	HumanBoundaryNextStep string `yaml:"humanBoundaryNextStep"`
}

type RuntimeClassificationHints struct {
	Docs       []string `yaml:"docs"`
	Tests      []string `yaml:"tests"`
	Code       []string `yaml:"code"`
	Config     []string `yaml:"config"`
	CI         []string `yaml:"ci"`
	Dependency []string `yaml:"dependency"`
}

type EvidenceContract struct {
	Document `yaml:",inline"`
	Spec     EvidenceContractSpec `yaml:"spec"`
	Path     string               `yaml:"-"`
	Raw      []byte               `yaml:"-"`
}

type EvidenceContractSpec struct {
	CapabilityRef     LocalRef          `yaml:"capabilityRef"`
	Categories        []string          `yaml:"categories"`
	SummaryFields     []string          `yaml:"summaryFields"`
	RawOutput         EvidenceRawOutput `yaml:"rawOutput"`
	Redaction         EvidenceRedaction `yaml:"redaction"`
	DeliveryChecklist []string          `yaml:"deliveryChecklist"`
	EvidenceID        EvidenceID        `yaml:"evidenceId"`
	Extra             map[string]any    `yaml:",inline"`
}

type EvidenceRawOutput struct {
	Default string `yaml:"default"`
}

type EvidenceRedaction struct {
	Secrets bool `yaml:"secrets"`
	Tokens  bool `yaml:"tokens"`
}

type EvidenceID struct {
	Prefix string `yaml:"prefix"`
}

type MCPBinding struct {
	Document `yaml:",inline"`
	Spec     MCPBindingSpec `yaml:"spec"`
	Path     string         `yaml:"-"`
	Raw      []byte         `yaml:"-"`
}

type MCPBindingSpec struct {
	CapabilityRef  LocalRef           `yaml:"capabilityRef"`
	Servers        []MCPRuntimeServer `yaml:"mcpservers"`
	RequiredTools  []MCPToolBinding   `yaml:"requiredTools"`
	Strategy       MCPBindingStrategy `yaml:"strategy"`
	GeneratedTool  MCPGeneratedTool   `yaml:"generatedTool"`
	GeneratedTools []MCPGeneratedTool `yaml:"generatedTools"`
}

type MCPRuntimeServer struct {
	Name      string         `yaml:"name"`
	Provider  string         `yaml:"provider"`
	Source    string         `yaml:"source"`
	Transport string         `yaml:"transport"`
	Command   []string       `yaml:"command"`
	Args      []string       `yaml:"args"`
	Env       map[string]any `yaml:"env"`
	URL       string         `yaml:"url"`
	Headers   map[string]any `yaml:"headers"`
	OAuth     any            `yaml:"oauth"`
	Timeout   int            `yaml:"timeout"`
	Enabled   *bool          `yaml:"enabled"`
}

type MCPBindingStrategy struct {
	Type    string `yaml:"type"`
	Handler string `yaml:"handler"`
}

type MCPGeneratedTool struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
	Mode        string `yaml:"mode" json:"mode"`
}

type LoadedPack struct {
	Root            string
	ManifestPath    string
	Manifest        CapabilityPack
	ManifestRaw     []byte
	Capabilities    []Capability
	Skills          []SkillContract
	Commands        []CommandContract
	Agents          []AgentContract
	Contracts       []ResponsibilityContract
	RuntimeProfiles []RuntimeProfile
	Evidence        []EvidenceContract
	Policies        []Policy
	MCPBindings     []MCPBinding
	TargetProfiles  []TargetProfile
}
