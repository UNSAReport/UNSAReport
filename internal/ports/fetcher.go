package ports

import "context"

type TemplateFetcher interface {
	Fetch(ctx context.Context, repo, ref, templatePath string) (map[string][]byte, error)
	FetchRaw(ctx context.Context, repo, ref, path string) ([]byte, error)
	LoadLocal(dir string) (map[string][]byte, error)
}
