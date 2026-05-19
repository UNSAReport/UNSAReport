package typst

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/christianmz565/lab-report/internal/dependencies"
)

type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

type queryItem struct {
	Value *struct {
		Name  string `json:"name"`
		Value any    `json:"value"`
	} `json:"value"`
}

func (a *Adapter) QueryVars(ctx context.Context, reportPath string) (map[string]string, error) {
	if err := dependencies.Check(dependencies.Typst); err != nil {
		return nil, err
	}

	args := []string{"query", "--root", ".", reportPath, "<var_export>"}

	cmd := exec.CommandContext(ctx, "typst", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("typst query failed: %s", msg)
	}

	var items []queryItem
	if err := json.Unmarshal(stdout.Bytes(), &items); err != nil {
		return nil, fmt.Errorf("failed to parse typst query output: %w", err)
	}

	vars := make(map[string]string)
	for _, it := range items {
		if it.Value == nil || it.Value.Name == "" {
			continue
		}

		switch v := it.Value.Value.(type) {
		case []any:
			parts := make([]string, 0, len(v))
			for _, p := range v {
				parts = append(parts, fmt.Sprint(p))
			}
			vars[it.Value.Name] = strings.Join(parts, "-")
		default:
			vars[it.Value.Name] = fmt.Sprint(v)
		}
	}
	return vars, nil
}

func (a *Adapter) Compile(ctx context.Context, reportPath, reportPDF, title string) error {
	if err := dependencies.Check(dependencies.Typst); err != nil {
		return err
	}

	args := []string{"compile", "--root", ".", "--input", fmt.Sprintf("title=%s", title), reportPath, reportPDF}

	cmd := exec.CommandContext(ctx, "typst", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("typst compile failed: %w", err)
	}
	return nil
}
