package ports

type CaptureConfig struct {
	Columns     int               `json:"columns"`
	FreezeFlags []string          `json:"freezeFlags"`
	Prompt      string            `json:"prompt"`
	Colors      map[string]string `json:"colors"`
}

type PrepareInputConfig struct {
	SrcDir     string `json:"srcDir"`
	ReportFile string `json:"reportFile"`
}

type PrepareOutputConfig struct {
	SubmissionDir string `json:"submissionDir"`
	FileTemplate  string `json:"fileTemplate"`
	ReportWord    string `json:"reportWord"`
	CodeWord      string `json:"codeWord"`
}

type PrepareConfig struct {
	Input  PrepareInputConfig  `json:"input"`
	Output PrepareOutputConfig `json:"output"`
}

type LabReportConfig struct {
	MultiLab bool          `json:"multiLab"`
	Sessions []string      `json:"sessions"`
	Capture  CaptureConfig `json:"capture"`
	Prepare  PrepareConfig `json:"prepare"`
}

type ConfigStore interface {
	FindProjectRoot(startDir string) (projectRoot string, cfg LabReportConfig, ok bool, err error)
	ReadConfig(destDir string) (cfg LabReportConfig, ok bool, err error)
	WriteConfig(destDir string, cfg LabReportConfig) error
}
