package ports

type TemplateInfo struct {
	Name        string
	Description string
	Repo        string
	Ref         string
	LocalPath   string
}

type TemplateRegistry interface {
	ListTemplates() ([]TemplateInfo, error)
	GetTemplate(name string) (TemplateInfo, error)
}
