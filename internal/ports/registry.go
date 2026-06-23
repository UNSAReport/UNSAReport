package ports

import "github.com/Masterminds/semver/v3"

type TemplateVersion struct {
	Version *semver.Version `json:"-"`
	Path    string          `json:"path"`
}

type TemplateInfo struct {
	Name        string
	Description string
	Version     string
	Path        string
	Repo        string
	Ref         string
	LocalPath   string
}

type TemplateRegistry interface {
	ListTemplates() ([]TemplateInfo, error)
	GetTemplate(name string) (TemplateInfo, error)
	GetTemplateVersion(name string, rangeSpec string) (TemplateInfo, error)
}
