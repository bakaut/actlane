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
	WhenToUse string           `yaml:"whenToUse"`
	Targets   []string         `yaml:"targets"`
	Inputs    map[string]Field `yaml:"inputs"`
	Outputs   map[string]Field `yaml:"outputs"`
	Policies  []string         `yaml:"policies"`
	ToolFlow  []ToolFlowStep   `yaml:"toolFlow"`
	Reporting map[string]bool  `yaml:"reporting"`
	Extra     map[string]any   `yaml:",inline"`
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
	Root         string
	ManifestPath string
	Manifest     CapabilityPack
	ManifestRaw  []byte
	Capabilities []Capability
	Policies     []Policy
}
