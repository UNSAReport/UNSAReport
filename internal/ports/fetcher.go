package ports

import "context"

// TemplateFetcher abstracts retrieving template files from remote repositories or local directories.
type TemplateFetcher interface {
	Fetch(ctx context.Context, repo, ref, templatePath string) (map[string][]byte, error)
	FetchRaw(ctx context.Context, repo, ref, path string) ([]byte, error)
	LoadLocal(dir string) (map[string][]byte, error)
}
