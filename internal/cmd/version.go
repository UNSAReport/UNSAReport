package cmd

import "github.com/UNSAReport/UNSAReport/internal/ports"

// Version is set at build time via ldflags.
// Usage: go build -ldflags "-X github.com/UNSAReport/UNSAReport/internal/cmd.Version={{.Version}}"
var Version = ports.Version
