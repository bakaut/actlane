package profile

import (
	"fmt"

	"github.com/actlane/actlane/packages/cli/internal/pack"
)

type TargetRenderer interface {
	Render(files map[string][]byte, loaded *pack.LoadedPack, capability pack.Capability, targetProfile pack.TargetProfile) error
}

func rendererFor(target string) (TargetRenderer, error) {
	switch target {
	case "opencode":
		return openCodeRenderer{}, nil
	default:
		return nil, fmt.Errorf("unsupported target renderer %q", target)
	}
}
