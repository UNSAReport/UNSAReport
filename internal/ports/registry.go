package ports

type TemplateInfo struct {
	Name        string
	Description string
	Path        string
	Repo        string
	Ref         string
	LocalPath   string
}

type TemplateMetadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
}

type RootManifest struct {
	Templates []TemplateMetadata `json:"templates"`
}

type TemplateRegistry interface {
	ListTemplates() ([]TemplateInfo, error)
	GetTemplate(name string) (TemplateInfo, error)
}
