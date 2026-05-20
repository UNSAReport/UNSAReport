package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/christianmz565/lab-report/internal/ports"
)

type CaptureService struct {
	Renderer ports.Renderer
	FS       ports.FileSystem
	Config   ports.ConfigStore
}

func NewCaptureService(r ports.Renderer, fs ports.FileSystem, c ports.ConfigStore) *CaptureService {
	return &CaptureService{
		Renderer: r,
		FS:       fs,
		Config:   c,
	}
}

type CaptureOptions struct {
	Cwd             string
	Args            []string
	FreezeFlags     []string
	SaveFreezeFlags bool
}

func (s *CaptureService) Execute(ctx context.Context, opts CaptureOptions) error {
	cwd, err := s.FS.Getwd()
	if err != nil {
		return fmt.Errorf("get cwd: %w", err)
	}

	projectRoot, cfg, ok, _ := s.Config.FindProjectRoot(cwd)
	if !ok {
		projectRoot = cwd
		cfg, _, _ = s.Config.ReadConfig(cwd)
	}

	if opts.SaveFreezeFlags {
		cfg.Capture.FreezeFlags = opts.FreezeFlags
		if err := s.Config.WriteConfig(projectRoot, cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
	}

	if len(opts.Args) < 1 {
		return fmt.Errorf("result image path is required")
	}
	resultPath := opts.Args[0]
	instructions := opts.Args[1:]

	if err := s.FS.EnsureDir(filepath.Dir(resultPath)); err != nil {
		return fmt.Errorf("ensure result directory: %w", err)
	}

	var commands []ports.CaptureCommand

	if opts.Cwd != "" {
		commands = append(commands, ports.CaptureCommand{Type: "Type", Args: fmt.Sprintf("cd %s", opts.Cwd)})
		commands = append(commands, ports.CaptureCommand{Type: "Enter"})
		commands = append(commands, ports.CaptureCommand{Type: "Type", Args: "clear"})
		commands = append(commands, ports.CaptureCommand{Type: "Enter"})
	}

	for _, instr := range instructions {
		fmt.Printf("Capturing instruction: %s\n", instr)
		if after, ok := strings.CutPrefix(instr, "w:"); ok {
			d, err := time.ParseDuration(after)
			if err != nil {
				d, err = time.ParseDuration(after + "ms")
				if err != nil {
					d, err = time.ParseDuration(after + "s")
				}
			}
			if err == nil {
				commands = append(commands, ports.CaptureCommand{Type: "Sleep", Delay: d})
				continue
			}
		}

		if after, ok := strings.CutPrefix(instr, "r:"); ok {
			commands = append(commands, ports.CaptureCommand{Type: "Raw", Args: after})
			commands = append(commands, ports.CaptureCommand{Type: "Sleep", Delay: 1 * time.Second})
			continue
		}

		if after, ok := strings.CutPrefix(instr, "c:"); ok {
			commands = append(commands, ports.CaptureCommand{Type: "Ctrl", Args: after})
			commands = append(commands, ports.CaptureCommand{Type: "Sleep", Delay: 1 * time.Second})
			continue
		}

		if after, ok := strings.CutPrefix(instr, "k:"); ok {
			commands = append(commands, ports.CaptureCommand{Type: "Key", Args: after})
			commands = append(commands, ports.CaptureCommand{Type: "Sleep", Delay: 1 * time.Second})
			continue
		}

		commands = append(commands, ports.CaptureCommand{Type: "Command", Args: instr})
		commands = append(commands, ports.CaptureCommand{Type: "Sleep", Delay: 1 * time.Second})
	}

	commands = append(commands, ports.CaptureCommand{Type: "Sleep", Delay: 1 * time.Second})

	var finalFlags []string
	if opts.SaveFreezeFlags {
		finalFlags = cfg.Capture.FreezeFlags
	} else {
		finalFlags = append(cfg.Capture.FreezeFlags, opts.FreezeFlags...)
	}

	output, err := s.Renderer.Render(ctx, resultPath, commands, finalFlags, cfg.Capture)
	if err != nil {
		return fmt.Errorf("render: %w", err)
	}

	if err := s.FS.EnsureDir("capture_logs"); err == nil {
		timestamp := time.Now().Format("02-01-2006_15-04-05")
		logPath := filepath.Join("capture_logs", timestamp+".log")
		s.FS.WriteFileAtomic(logPath, []byte(output), 0644)
	}

	return nil
}
