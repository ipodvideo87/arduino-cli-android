package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const ManifestFileName = "manifest.json"

// Manifest describes an ACL runtime package. It is intentionally extensible so
// future runtime builders can add fields without breaking older managers.
type Manifest struct {
	SchemaVersion      string                     `json:"schema_version"`
	RuntimeID          string                     `json:"runtime_id"`
	RuntimeVersion     string                     `json:"runtime_version"`
	Architecture       string                     `json:"architecture"`
	SupportedABIs      []string                   `json:"supported_abis"`
	Loader             RuntimeFile                `json:"loader"`
	Libraries          []RuntimeFile              `json:"libraries,omitempty"`
	Hashes             map[string]string          `json:"hashes,omitempty"`
	CompatibilityLevel string                     `json:"compatibility_level"`
	CreatedAt          string                     `json:"created_at"`
	Build              BuildInfo                  `json:"build"`
	Extensions         map[string]json.RawMessage `json:"extensions,omitempty"`
}

type RuntimeFile struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	SONAME   string `json:"soname,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Required bool   `json:"required,omitempty"`
}

type BuildInfo struct {
	Tool          string `json:"tool,omitempty"`
	Builder       string `json:"builder,omitempty"`
	Source        string `json:"source,omitempty"`
	SourceVersion string `json:"source_version,omitempty"`
	GitCommit     string `json:"git_commit,omitempty"`
	GoVersion     string `json:"go_version,omitempty"`
	BuiltAt       string `json:"built_at,omitempty"`
	HostOS        string `json:"host_os,omitempty"`
	HostArch      string `json:"host_arch,omitempty"`
	Notes         string `json:"notes,omitempty"`
}

func LoadManifest(path string) (Manifest, error) {
	var manifest Manifest
	data, err := os.ReadFile(path)
	if err != nil {
		return manifest, err
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifest, fmt.Errorf("decode manifest %q: %w", path, err)
	}
	return manifest, nil
}

func (m Manifest) ValidateBasic() error {
	switch {
	case strings.TrimSpace(m.SchemaVersion) == "":
		return errors.New("schema_version is required")
	case strings.TrimSpace(m.RuntimeID) == "":
		return errors.New("runtime_id is required")
	case strings.TrimSpace(m.RuntimeVersion) == "":
		return errors.New("runtime_version is required")
	case strings.TrimSpace(m.Architecture) == "":
		return errors.New("architecture is required")
	case len(m.SupportedABIs) == 0:
		return errors.New("supported_abis is required")
	case strings.TrimSpace(m.CompatibilityLevel) == "":
		return errors.New("compatibility_level is required")
	case strings.TrimSpace(m.CreatedAt) == "":
		return errors.New("created_at is required")
	case strings.TrimSpace(m.Loader.Path) == "":
		return errors.New("loader.path is required")
	}

	if err := validateRelativePath(m.Loader.Path); err != nil {
		return fmt.Errorf("loader.path: %w", err)
	}
	for i, file := range m.Libraries {
		if strings.TrimSpace(file.Path) == "" {
			return fmt.Errorf("libraries[%d].path is required", i)
		}
		if err := validateRelativePath(file.Path); err != nil {
			return fmt.Errorf("libraries[%d].path: %w", i, err)
		}
	}

	return nil
}

func (m Manifest) AllFiles() []RuntimeFile {
	files := make([]RuntimeFile, 0, 1+len(m.Libraries))
	if m.Loader.Path != "" {
		files = append(files, m.Loader)
	}
	files = append(files, m.Libraries...)
	return files
}

func validateRelativePath(rel string) error {
	if filepath.IsAbs(rel) {
		return errors.New("must be relative")
	}
	clean := filepath.Clean(rel)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return errors.New("must stay within the runtime root")
	}
	if clean != rel {
		return fmt.Errorf("must be clean relative path, got %q", rel)
	}
	return nil
}
