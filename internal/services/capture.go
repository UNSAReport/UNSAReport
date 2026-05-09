package services

import (
	"context"
	"fmt"
	"path/filepath"

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

func (s *CaptureService) Execute(ctx context.Context, tapeFile string) error {
	cwd, err := s.FS.Getwd()
	if err != nil {
		return fmt.Errorf("get cwd: %w", err)
	}

	projectRoot, _, ok, err := s.Config.FindProjectRoot(cwd)
	if !ok {
		projectRoot = cwd
	}

	if err := s.FS.Chdir(projectRoot); err != nil {
		return fmt.Errorf("chdir to project root: %w", err)
	}

	configTapePath := "config.tape"
	if !s.FS.FileExists(configTapePath) {
		configTapeContent := "Set Width 1000\nSet TypingSpeed 0.1\n"
		if err := s.FS.WriteFileAtomic(configTapePath, []byte(configTapeContent), 0644); err != nil {
			return fmt.Errorf("write config.tape: %w", err)
		}

		templateTapePath := "template.tape"
		if !s.FS.FileExists(templateTapePath) {
			templateContent := "Source config.tape\n\nType \"echo 'Hello from VHS!'\"\nEnter\nSleep 1s\n\nScreenshot output.png\n"
			if err := s.FS.WriteFileAtomic(templateTapePath, []byte(templateContent), 0644); err != nil {
				return fmt.Errorf("write template.tape: %w", err)
			}
		}

		fmt.Println("config.tape was not found.")
		fmt.Println("Created config.tape and template.tape with defaults in the project root.")
		fmt.Println("Please review them and run the command again.")
		return nil
	}

	tapePathAbs := filepath.Join(cwd, tapeFile)
	if !filepath.IsAbs(tapePathAbs) {
		var err error
		tapePathAbs, err = filepath.Abs(tapePathAbs)
		if err != nil {
			return fmt.Errorf("abs path: %w", err)
		}
	}

	if err := s.Renderer.Render(ctx, tapePathAbs); err != nil {
		return fmt.Errorf("render tape: %w", err)
	}

	return nil
}
