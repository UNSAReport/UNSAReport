package ports

type SubmissionConfig struct {
	Template   string `json:"template,omitempty"`
	ReportWord string `json:"reportWord,omitempty"`
	CodeWord   string `json:"codeWord,omitempty"`
}

type LabReportConfig struct {
	MultiLab   bool             `json:"multiLab"`
	Submission SubmissionConfig `json:"submission"`
}

type ConfigStore interface {
	FindProjectRoot(startDir string) (projectRoot string, cfg LabReportConfig, ok bool, err error)
	ReadConfig(destDir string) (cfg LabReportConfig, ok bool, err error)
	WriteConfig(destDir string, cfg LabReportConfig) error
}
