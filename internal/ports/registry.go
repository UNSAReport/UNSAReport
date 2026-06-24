package ports

const (
	// DefaultTemplateRepo is the GitHub repository for report templates.
	DefaultTemplateRepo = "UNSAReport/templates"
	// DefaultComponentRepo is the GitHub repository for reusable components.
	DefaultComponentRepo = "UNSAReport/components"
	// DefaultRef is the default branch or tag used when fetching from remote repositories.
	DefaultRef = "main"
)

// TemplateInfo describes an available template including its version and location.
type TemplateInfo struct {
	Name        string
	Description string
	Version     string
	Path        string
	LocalPath   string
}

// TemplateRegistry abstracts a registry for discovering and resolving template versions.
type TemplateRegistry interface {
	ListTemplates() ([]TemplateInfo, error)
	GetTemplate(name string) (TemplateInfo, error)
	GetTemplateVersion(name string, rangeSpec string) (TemplateInfo, error)
}
