package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Masterminds/semver/v3"
	"github.com/UNSAReport/UNSAReport/internal/ports"
)

type Adapter struct {
	localDir  string
	registry  *registryTemplateFile
	templates []ports.TemplateInfo
}

func New(localDir string) *Adapter {
	a := &Adapter{
		localDir: localDir,
	}
	a.loadRegistry()
	return a
}

func NewLocal(localDir string) *Adapter {
	return New(localDir)
}

func (a *Adapter) loadRegistry() {
	a.registry = nil
	a.templates = nil

	regPath := filepath.Join(a.localDir, "registry.json")
	data, err := os.ReadFile(regPath)
	if err != nil {
		return
	}

	var reg registryTemplateFile
	if err := json.Unmarshal(data, &reg); err != nil {
		return
	}

	a.registry = &reg

	for name, entry := range reg.Templates {
		a.templates = append(a.templates, ports.TemplateInfo{
			Name:        name,
			Description: entry.Description,
			LocalPath:   filepath.Join(a.localDir, name),
		})
	}
}

func (a *Adapter) ListTemplates() ([]ports.TemplateInfo, error) {
	a.loadRegistry()

	if len(a.templates) == 0 {
		return nil, fmt.Errorf("no templates found in %s", a.localDir)
	}

	return a.templates, nil
}

func (a *Adapter) GetTemplate(name string) (ports.TemplateInfo, error) {
	templates, err := a.ListTemplates()
	if err != nil {
		return ports.TemplateInfo{}, err
	}

	for _, t := range templates {
		if t.Name == name {
			return t, nil
		}
	}

	return ports.TemplateInfo{}, fmt.Errorf("template %q not found", name)
}

func (a *Adapter) GetTemplateVersion(name string, rangeSpec string) (ports.TemplateInfo, error) {
	if a.registry == nil {
		return ports.TemplateInfo{}, fmt.Errorf("registry not loaded")
	}

	entry, ok := a.registry.Templates[name]
	if !ok {
		return ports.TemplateInfo{}, fmt.Errorf("template %q not found", name)
	}

	distTags := make(map[string]*semver.Version)
	for tag, vStr := range entry.DistTags {
		if v, err := semver.NewVersion(vStr); err == nil {
			distTags[tag] = v
		}
	}

	availableVersions := make(map[string]*semver.Version)
	for vStr := range entry.Versions {
		if v, err := semver.NewVersion(vStr); err == nil {
			availableVersions[vStr] = v
		}
	}

	resolved, err := resolveVersionFromMap(availableVersions, distTags, rangeSpec)
	if err != nil {
		return ports.TemplateInfo{}, fmt.Errorf("resolve version: %w", err)
	}

	vEntry, ok := entry.Versions[resolved.String()]
	if !ok {
		return ports.TemplateInfo{}, fmt.Errorf("version %q not found", resolved.String())
	}

	return ports.TemplateInfo{
		Name:        name,
		Description: entry.Description,
		Version:     resolved.String(),
		Path:        vEntry.Path,
		LocalPath:   filepath.Join(a.localDir, vEntry.Path),
	}, nil
}

func (a *Adapter) TemplateExists(name string) bool {
	path := filepath.Join(a.localDir, name)
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func (a *Adapter) TemplateDir() string {
	return a.localDir
}
