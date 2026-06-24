package registry

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/UNSAReport/UNSAReport/internal/ports"
)

type registryTemplateFile struct {
	Templates map[string]registryTemplateEntry `json:"templates"`
}

type registryTemplateEntry struct {
	Description string                             `json:"description"`
	DistTags    map[string]string                  `json:"dist-tags"`
	Versions    map[string]registryTemplateVersion `json:"versions"`
}

type registryTemplateVersion struct {
	Path string `json:"path"`
}

var _ ports.TemplateRegistry = (*RemoteAdapter)(nil)

// RemoteAdapter implements ports.TemplateRegistry by fetching registry.json from a remote GitHub repository.
type RemoteAdapter struct {
	fetcher ports.TemplateFetcher
}

// NewRemote creates a RemoteAdapter that uses fetcher to retrieve registry data from GitHub.
func NewRemote(fetcher ports.TemplateFetcher) *RemoteAdapter {
	return &RemoteAdapter{
		fetcher: fetcher,
	}
}

func (a *RemoteAdapter) fetchRegistry() (*registryTemplateFile, error) {
	ctx := context.Background()
	data, err := a.fetcher.FetchRaw(ctx, ports.DefaultTemplateRepo, ports.DefaultRef, "registry.json")
	if err != nil {
		return nil, fmt.Errorf("fetch registry.json: %w", err)
	}

	var reg registryTemplateFile
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parse registry.json: %w", err)
	}

	return &reg, nil
}

func (a *RemoteAdapter) convertTemplate(name string, entry registryTemplateEntry) ports.TemplateInfo {
	return ports.TemplateInfo{
		Name:        name,
		Description: entry.Description,
	}
}

// ListTemplates fetches the remote registry and returns all available templates.
func (a *RemoteAdapter) ListTemplates() ([]ports.TemplateInfo, error) {
	reg, err := a.fetchRegistry()
	if err != nil {
		return nil, err
	}

	templates := make([]ports.TemplateInfo, 0, len(reg.Templates))
	for name, entry := range reg.Templates {
		templates = append(templates, a.convertTemplate(name, entry))
	}

	return templates, nil
}

// GetTemplate fetches the remote registry and returns metadata for the named template.
func (a *RemoteAdapter) GetTemplate(name string) (ports.TemplateInfo, error) {
	reg, err := a.fetchRegistry()
	if err != nil {
		return ports.TemplateInfo{}, err
	}

	entry, ok := reg.Templates[name]
	if !ok {
		return ports.TemplateInfo{}, fmt.Errorf("template %q not found", name)
	}

	return a.convertTemplate(name, entry), nil
}

// GetTemplateVersion fetches the remote registry and resolves rangeSpec against dist-tags and versions.
func (a *RemoteAdapter) GetTemplateVersion(name string, rangeSpec string) (ports.TemplateInfo, error) {
	reg, err := a.fetchRegistry()
	if err != nil {
		return ports.TemplateInfo{}, err
	}

	entry, ok := reg.Templates[name]
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
		return ports.TemplateInfo{}, fmt.Errorf("version %q not found in registry", resolved.String())
	}

	return ports.TemplateInfo{
		Name:        name,
		Description: entry.Description,
		Version:     resolved.String(),
		Path:        vEntry.Path,
	}, nil
}
