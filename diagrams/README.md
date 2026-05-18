# Actlane Diagrams

This directory contains Phase 0 architecture diagrams for the Actlane RFC.

The source files live in `diagrams/plantuml/`.
Rendered SVG files should live in `diagrams/svg/` when rendering is available.

## Phase 0 Diagrams

- `plantuml/actlane-overview.puml` - high-level product boundary.
- `plantuml/capability-development-flow.puml` - how a capability becomes generated artifacts.
- `plantuml/pack-portability-flow.puml` - how one pack can move across agent runtimes.
- `plantuml/brownfield-adoption-flow.puml` - safe adoption in an existing project.
- `plantuml/optional-runtime-enforcement.puml` - optional policy enforcement after generation.

## Guardrail

Actlane is generation-first. Runtime enforcement is optional and should be shown as a later layer, not as the core MVP.
