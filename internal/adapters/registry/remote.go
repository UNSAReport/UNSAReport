package registry

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/UNSAReport/UNSAReport/internal/ports"
)

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

func (a *RemoteAdapter) ListTemplates() ([]ports.TemplateInfo, error) {
	ctx := context.Background()

	data, err := a.fetcher.FetchRaw(ctx, a.repo, a.ref, "manifest.json")
	if err != nil {
		return nil, fmt.Errorf("fetch root manifest: %w", err)
	}

	var root ports.RootManifest
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse root manifest: %w", err)
	}

	if len(root.Templates) == 0 {
		return nil, fmt.Errorf("no templates found in root manifest")
	}

	templates := make([]ports.TemplateInfo, len(root.Templates))
	for i, t := range root.Templates {
		templates[i] = ports.TemplateInfo{
			Name:        t.Name,
			Description: t.Description,
			Path:        t.Path,
			Repo:        a.repo,
			Ref:         a.ref,
		}
	}

	return templates, nil
}

func (a *RemoteAdapter) GetTemplate(name string) (ports.TemplateInfo, error) {
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
