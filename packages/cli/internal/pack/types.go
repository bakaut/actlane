package pack

type Document struct {
	Schema     string   `yaml:"$schema"`
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
}

type Metadata struct {
	Name        string `yaml:"name"`
	Title       string `yaml:"title"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
}

type CapabilityPack struct {
	Document `yaml:",inline"`
	Spec     PackSpec `yaml:"spec"`
}

type PackSpec struct {
	Capabilities    []string       `yaml:"capabilities"`
	Policies        []string       `yaml:"policies"`
	Targets         []string       `yaml:"targets"`
	TargetProfiles  []string       `yaml:"targetProfiles"`
	AdoptionProfile map[string]any `yaml:"adoptionProfile"`
}

type Capability struct {
	Document `yaml:",inline"`
	Spec     CapabilitySpec `yaml:"spec"`
	Path     string         `yaml:"-"`
	Raw      []byte         `yaml:"-"`
}

type CapabilitySpec struct {
	WhenToUse string                       `yaml:"whenToUse"`
	Targets   []string                     `yaml:"targets"`
	Inputs    map[string]Field             `yaml:"inputs"`
	Outputs   map[string]Field             `yaml:"outputs"`
	Policies  []string                     `yaml:"policies"`
	ToolFlow  []ToolFlowStep               `yaml:"toolFlow"`
	MCP       MCPSpec                      `yaml:"mcp"`
	Profiles  map[string]CapabilityProfile `yaml:"profiles"`
	Reporting map[string]bool              `yaml:"reporting"`
	Extra     map[string]any               `yaml:",inline"`
}

type Field struct {
	Type     string `yaml:"type"`
	Items    *Field `yaml:"items"`
	Required bool   `yaml:"required"`
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
	Output     TargetProfileOutput     `yaml:"output"`
	Transforms TargetProfileTransforms `yaml:"transforms"`
	Install    map[string]any          `yaml:"install"`
}

type TargetProfileOutput struct {
	Root   string `yaml:"root"`
	Config string `yaml:"config"`
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
	Mutate           []PolicyMutate   `yaml:"mutate"`
	RequiresApproval []map[string]any `yaml:"requiresApproval"`
	Audit            PolicyAudit      `yaml:"audit"`
}

type PolicyDeny struct {
	Reason       string   `yaml:"reason"`
	Paths        []string `yaml:"paths"`
	MaxFiles     int      `yaml:"maxFiles"`
	MaxDiffBytes int      `yaml:"maxDiffBytes"`
}

type PolicyMutate struct {
	Field        string `yaml:"field"`
	EnsurePrefix string `yaml:"ensurePrefix"`
	Value        any    `yaml:"value"`
	Reason       string `yaml:"reason"`
}

type PolicyAudit struct {
	Include []string `yaml:"include"`
}

type LoadedPack struct {
	Root           string
	ManifestPath   string
	Manifest       CapabilityPack
	ManifestRaw    []byte
	Capabilities   []Capability
	Policies       []Policy
	TargetProfiles []TargetProfile
}
