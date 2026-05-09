package ports

import "context"

type TemplateFetcher interface {
	Fetch(ctx context.Context, repo, ref string) (map[string][]byte, error)
	LoadLocal(dir string) (map[string][]byte, error)
}
