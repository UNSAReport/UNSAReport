package registry

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/UNSAReport/UNSAReport/internal/ports"
)

type Adapter struct {
	localDir  string
	templates []ports.TemplateInfo
}

func New(localDir string) *Adapter {
	a := &Adapter{
		localDir: localDir,
	}
	a.loadLocalTemplates()
	return a
}

func NewLocal(localDir string) *Adapter {
	return New(localDir)
}

func (a *Adapter) loadLocalTemplates() {
	a.templates = nil

	entries, err := os.ReadDir(a.localDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		manifestPath := filepath.Join(a.localDir, name, "manifest.json")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			continue
		}

		a.templates = append(a.templates, ports.TemplateInfo{
			Name:      name,
			LocalPath: filepath.Join(a.localDir, name),
			Repo:      "UNSAReport/templates",
			Ref:       "main",
		})
	}
}

func (a *Adapter) ListTemplates() ([]ports.TemplateInfo, error) {
	a.loadLocalTemplates()

	if len(a.templates) == 0 {
		return nil, fmt.Errorf("no templates found in %s", a.localDir)
	}

	return a.templates, nil
}

func (a *Adapter) GetTemplate(name string) (ports.TemplateInfo, error) {
	templates, err := a.ListTemplates()
	if err != nil {
		return ports.TemplateInfo{}, err
	}

	for _, t := range templates {
		if t.Name == name {
			return t, nil
		}
	}

	return ports.TemplateInfo{}, fmt.Errorf("template %q not found", name)
}

func (a *Adapter) TemplateExists(name string) bool {
	path := filepath.Join(a.localDir, name)
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func (a *Adapter) TemplateDir() string {
	return a.localDir
}
