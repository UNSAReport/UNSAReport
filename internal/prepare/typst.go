package prepare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type queryItem struct {
	Value *struct {
		Name  string      `json:"name"`
		Value interface{} `json:"value"`
	} `json:"value"`
}

func QueryVars(ctx context.Context, reportPath string, useRoot bool) (map[string]string, error) {
	args := []string{"query"}
	if useRoot {
		args = append(args, "--root", ".")
	}
	args = append(args, reportPath, "<var_export>")

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
		case []interface{}:
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

func Compile(ctx context.Context, reportPath, reportPDF, title string, useRoot bool) error {
	args := []string{"compile"}
	if useRoot {
		args = append(args, "--root", ".")
	}
	args = append(args, "--input", fmt.Sprintf("title=%s", title), reportPath, reportPDF)

	cmd := exec.CommandContext(ctx, "typst", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	// The CLI layer wires stdio for user visibility.
	return cmd.Run()
}
