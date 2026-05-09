package ports

type CaptureConfig struct {
	TapeConfig string `json:"tapeConfig,omitempty"`
}

type PrepareInputConfig struct {
	SrcDir     string `json:"srcDir,omitempty"`
	ReportFile string `json:"reportFile,omitempty"`
}

type PrepareOutputConfig struct {
	SubmissionDir string `json:"submissionDir,omitempty"`
	FileTemplate  string `json:"fileTemplate,omitempty"`
	ReportWord    string `json:"reportWord,omitempty"`
	CodeWord      string `json:"codeWord,omitempty"`
}

type PrepareConfig struct {
	Input  PrepareInputConfig  `json:"input"`
	Output PrepareOutputConfig `json:"output"`
}

type LabReportConfig struct {
	MultiLab bool          `json:"multiLab"`
	Capture  CaptureConfig `json:"capture"`
	Prepare  PrepareConfig `json:"prepare"`
}

type ConfigStore interface {
	FindProjectRoot(startDir string) (projectRoot string, cfg LabReportConfig, ok bool, err error)
	ReadConfig(destDir string) (cfg LabReportConfig, ok bool, err error)
	WriteConfig(destDir string, cfg LabReportConfig) error
}
