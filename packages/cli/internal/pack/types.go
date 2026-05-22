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
	Intent        CapabilityIntent             `yaml:"intent"`
	Interface     CapabilityInterface          `yaml:"interface"`
	PolicyRef     LocalRef                     `yaml:"policyRef"`
	ExecutionRef  LocalRef                     `yaml:"executionRef"`
	WorkflowHints []WorkflowHint               `yaml:"workflowHints"`
	Projections   CapabilityProjections        `yaml:"projections"`
	Reporting     map[string]bool              `yaml:"reporting"`
	WhenToUse     string                       `yaml:"whenToUse"`
	Targets       []string                     `yaml:"targets"`
	Inputs        map[string]Field             `yaml:"inputs"`
	Outputs       map[string]Field             `yaml:"outputs"`
	Policies      []string                     `yaml:"policies"`
	ToolFlow      []ToolFlowStep               `yaml:"toolFlow"`
	MCP           MCPSpec                      `yaml:"mcp"`
	Profiles      map[string]CapabilityProfile `yaml:"profiles"`
	Extra         map[string]any               `yaml:",inline"`
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
	Name           string   `yaml:"name"`
	Server         string   `yaml:"server"`
	Toolset        string   `yaml:"toolset"`
	Description    string   `yaml:"description"`
	RequiredScopes []string `yaml:"requiredScopes"`
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
	Path    string `yaml:"path"`
	Source  string `yaml:"source"`
	Content string `yaml:"content"`
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
	TargetPath    string `yaml:"targetPath"`
	GeneratedPath string `yaml:"generatedPath"`
	Owned         bool   `yaml:"owned"`
	OwnedBlock    bool   `yaml:"ownedBlock"`
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
}

type MCPBindingStrategy struct {
	Type    string `yaml:"type"`
	Handler string `yaml:"handler"`
}

type MCPGeneratedTool struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Mode        string `yaml:"mode"`
}

type LoadedPack struct {
	Root           string
	ManifestPath   string
	Manifest       CapabilityPack
	ManifestRaw    []byte
	Capabilities   []Capability
	Policies       []Policy
	MCPBindings    []MCPBinding
	TargetProfiles []TargetProfile
}
