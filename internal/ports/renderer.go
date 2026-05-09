package ports

import "context"

type Renderer interface {
	Render(ctx context.Context, tapePath string) error
}
