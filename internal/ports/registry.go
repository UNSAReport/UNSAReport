package ports

const (
	DefaultTemplateRepo  = "UNSAReport/templates"
	DefaultComponentRepo = "UNSAReport/components"
	DefaultRef           = "main"
)

type TemplateInfo struct {
	Name        string
	Description string
	Version     string
	Path        string
	LocalPath   string
}

type TemplateRegistry interface {
	ListTemplates() ([]TemplateInfo, error)
	GetTemplate(name string) (TemplateInfo, error)
	GetTemplateVersion(name string, rangeSpec string) (TemplateInfo, error)
}
