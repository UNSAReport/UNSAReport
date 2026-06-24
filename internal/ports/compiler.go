package ports

import "context"

type Compiler interface {
	QueryVars(ctx context.Context, reportPath string) (map[string]string, error)
	Compile(ctx context.Context, reportPath, reportPDF string, inputs map[string]string) error
}
