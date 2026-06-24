package main

import (
	"log/slog"
	"os"

	"github.com/UNSAReport/UNSAReport/internal/cmd"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cmd.Execute()
}
