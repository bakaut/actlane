package pack

type BrokerBundle struct {
	Pack            string                       `json:"pack"`
	Version         string                       `json:"version"`
	Target          string                       `json:"target"`
	Capabilities    []BrokerBundleCapability     `json:"capabilities"`
	Policies        []BrokerBundlePolicy         `json:"policies"`
	MCPBindings     []BrokerBundleMCPBinding     `json:"mcpBindings"`
	Responsibility  []BrokerBundleResponsibility `json:"responsibility,omitempty"`
	RuntimeProfiles []BrokerBundleRuntimeProfile `json:"runtimeProfiles,omitempty"`
	Evidence        []BrokerBundleEvidence       `json:"evidence,omitempty"`
}

type BrokerBundleCapability struct {
	Name        string         `json:"name"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	Spec        CapabilitySpec `json:"spec"`
}

type BrokerBundlePolicy struct {
	Name        string     `json:"name"`
	Title       string     `json:"title,omitempty"`
	Description string     `json:"description,omitempty"`
	Spec        PolicySpec `json:"spec"`
}

type BrokerBundleMCPBinding struct {
	Name        string         `json:"name"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	Spec        MCPBindingSpec `json:"spec"`
}

type BrokerBundleResponsibility struct {
	Name        string         `json:"name"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	Spec        map[string]any `json:"spec"`
}

type BrokerBundleRuntimeProfile struct {
	Name        string             `json:"name"`
	Title       string             `json:"title,omitempty"`
	Description string             `json:"description,omitempty"`
	Spec        RuntimeProfileSpec `json:"spec"`
}

type BrokerBundleEvidence struct {
	Name        string               `json:"name"`
	Title       string               `json:"title,omitempty"`
	Description string               `json:"description,omitempty"`
	Spec        EvidenceContractSpec `json:"spec"`
}

func NewBrokerBundle(loaded *LoadedPack, target string) BrokerBundle {
	bundle := BrokerBundle{
		Pack:    loaded.Manifest.Metadata.Name,
		Version: loaded.Manifest.Metadata.Version,
		Target:  target,
	}
	for _, item := range loaded.Capabilities {
		bundle.Capabilities = append(bundle.Capabilities, BrokerBundleCapability{
			Name:        item.Metadata.Name,
			Title:       item.Metadata.Title,
			Description: item.Metadata.Description,
			Spec:        item.Spec,
		})
	}
	for _, item := range loaded.Policies {
		bundle.Policies = append(bundle.Policies, BrokerBundlePolicy{
			Name:        item.Metadata.Name,
			Title:       item.Metadata.Title,
			Description: item.Metadata.Description,
			Spec:        item.Spec,
		})
	}
	for _, item := range loaded.MCPBindings {
		if item.Spec.Strategy.Handler == "actlane.pack.author" {
			continue
		}
		bundle.MCPBindings = append(bundle.MCPBindings, BrokerBundleMCPBinding{
			Name:        item.Metadata.Name,
			Title:       item.Metadata.Title,
			Description: item.Metadata.Description,
			Spec:        brokerMCPBindingSpec(item.Spec),
		})
	}
	for _, item := range loaded.Contracts {
		bundle.Responsibility = append(bundle.Responsibility, BrokerBundleResponsibility{
			Name:        item.Metadata.Name,
			Title:       item.Metadata.Title,
			Description: item.Metadata.Description,
			Spec:        brokerResponsibilitySpec(item.Spec),
		})
	}
	for _, item := range loaded.RuntimeProfiles {
		bundle.RuntimeProfiles = append(bundle.RuntimeProfiles, BrokerBundleRuntimeProfile{
			Name:        item.Metadata.Name,
			Title:       item.Metadata.Title,
			Description: item.Metadata.Description,
			Spec:        item.Spec,
		})
	}
	for _, item := range loaded.Evidence {
		bundle.Evidence = append(bundle.Evidence, BrokerBundleEvidence{
			Name:        item.Metadata.Name,
			Title:       item.Metadata.Title,
			Description: item.Metadata.Description,
			Spec:        item.Spec,
		})
	}
	return bundle
}

func LoadedFromBrokerBundle(bundle BrokerBundle) *LoadedPack {
	loaded := &LoadedPack{
		Manifest: CapabilityPack{
			Document: Document{
				Kind: "CapabilityPack",
				Metadata: Metadata{
					Name:    bundle.Pack,
					Version: bundle.Version,
				},
			},
		},
	}
	for _, item := range bundle.Capabilities {
		loaded.Capabilities = append(loaded.Capabilities, Capability{
			Document: Document{Kind: "Capability", Metadata: bundleMetadata(item.Name, item.Title, item.Description)},
			Spec:     item.Spec,
		})
	}
	for _, item := range bundle.Policies {
		loaded.Policies = append(loaded.Policies, Policy{
			Document: Document{Kind: "ToolCallPolicy", Metadata: bundleMetadata(item.Name, item.Title, item.Description)},
			Spec:     item.Spec,
		})
	}
	for _, item := range bundle.MCPBindings {
		loaded.MCPBindings = append(loaded.MCPBindings, MCPBinding{
			Document: Document{Kind: "MCPBinding", Metadata: bundleMetadata(item.Name, item.Title, item.Description)},
			Spec:     item.Spec,
		})
	}
	for _, item := range bundle.Responsibility {
		loaded.Contracts = append(loaded.Contracts, ResponsibilityContract{
			Document: Document{Kind: "ResponsibilityContract", Metadata: bundleMetadata(item.Name, item.Title, item.Description)},
			Spec:     item.Spec,
		})
	}
	for _, item := range bundle.RuntimeProfiles {
		loaded.RuntimeProfiles = append(loaded.RuntimeProfiles, RuntimeProfile{
			Document: Document{Kind: "RuntimeProfile", Metadata: bundleMetadata(item.Name, item.Title, item.Description)},
			Spec:     item.Spec,
		})
	}
	for _, item := range bundle.Evidence {
		loaded.Evidence = append(loaded.Evidence, EvidenceContract{
			Document: Document{Kind: "EvidenceContract", Metadata: bundleMetadata(item.Name, item.Title, item.Description)},
			Spec:     item.Spec,
		})
	}
	return loaded
}

func bundleMetadata(name, title, description string) Metadata {
	return Metadata{Name: name, Title: title, Description: description}
}

func brokerMCPBindingSpec(spec MCPBindingSpec) MCPBindingSpec {
	if spec.Strategy.Handler != "actlane.mcp.broker" {
		return spec
	}
	out := spec
	for i := range out.Servers {
		out.Servers[i].Command = []string{"actlane"}
		out.Servers[i].Args = []string{"mcp", "serve", "--broker-bundle", "./broker/broker-bundle.json"}
		out.Servers[i].Env = nil
		out.Servers[i].URL = ""
		out.Servers[i].Headers = nil
		out.Servers[i].OAuth = nil
	}
	return out
}

func brokerResponsibilitySpec(spec map[string]any) map[string]any {
	out := map[string]any{}
	for _, key := range []string{"humanBoundary", "risk", "scopes", "checks", "evidence", "tools"} {
		if value, ok := spec[key]; ok {
			out[key] = value
		}
	}
	return out
}
