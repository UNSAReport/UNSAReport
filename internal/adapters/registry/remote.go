package registry

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/UNSAReport/UNSAReport/internal/ports"
)

type registryTemplateFile struct {
	Version    string                             `json:"version"`
	Templates  map[string]registryTemplateEntry   `json:"templates"`
}

type registryTemplateEntry struct {
	Description string                            `json:"description"`
	Repo        string                            `json:"repo"`
	Ref         string                            `json:"ref"`
	DistTags    map[string]string                 `json:"dist-tags"`
	Versions    map[string]registryTemplateVersion `json:"versions"`
}

type registryTemplateVersion struct {
	Path string `json:"path"`
}

type RemoteAdapter struct {
	repo    string
	ref     string
	fetcher ports.TemplateFetcher
}

func NewRemote(repo, ref string, fetcher ports.TemplateFetcher) *RemoteAdapter {
	return &RemoteAdapter{
		repo:    repo,
		ref:     ref,
		fetcher: fetcher,
	}
}

func (a *RemoteAdapter) fetchRegistry() (*registryTemplateFile, error) {
	ctx := context.Background()
	data, err := a.fetcher.FetchRaw(ctx, a.repo, a.ref, "registry.json")
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
		Repo:        entry.Repo,
		Ref:         entry.Ref,
	}
}

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
		Repo:        entry.Repo,
		Ref:         entry.Ref,
	}, nil
}
