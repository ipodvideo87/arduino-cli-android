package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
	aclinstall "github.com/arduino/arduino-cli/internal/acl/install"
)

type Package struct {
	Name     string                   `json:"name,omitempty"`
	Version  string                   `json:"version,omitempty"`
	Root     string                   `json:"root,omitempty"`
	Manifest aclinstall.PatchManifest `json:"manifest"`
	Ready    bool                     `json:"ready"`
	Beginner string                   `json:"beginner_summary,omitempty"`
	Detail   []string                 `json:"professional_details,omitempty"`
}

func New(root, name, version string) Package {
	return Package{
		Name:    name,
		Version: version,
		Root:    root,
		Manifest: aclinstall.PatchManifest{
			PackageName:    name,
			PackageVersion: version,
			Source:         "bootstrap package",
			Metadata:       map[string]string{"root": root},
		},
	}
}

func (p *Package) Run(ctx context.Context, executor aclinstall.StageExecutor) error {
	pipeline := aclinstall.NewAndroidInstallPatchPipeline(executor)
	if err := pipeline.Run(ctx, &p.Manifest); err != nil {
		p.Ready = false
		p.Beginner = err.Error()
		return err
	}
	p.Ready = p.Manifest.Status == diagnostics.StatusPassed || p.Manifest.Status == diagnostics.StatusWarning
	p.Beginner = p.Manifest.Summary()
	p.Detail = append([]string(nil), p.Manifest.StageSummaries()...)
	return nil
}

func (p Package) JSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

func (p Package) String() string {
	return fmt.Sprintf("%s@%s ready=%t", p.Name, p.Version, p.Ready)
}

func (p Package) BeginnerSummary() string {
	if strings.TrimSpace(p.Beginner) != "" {
		return p.Beginner
	}
	if p.Ready {
		return "bootstrap package ready"
	}
	return "bootstrap package not ready"
}
