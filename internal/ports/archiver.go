package ports

type Archiver interface {
	ArchiveDir(zipPath, srcDir string) error
	ArchiveFiles(zipPath, baseDir string, files []string) error
}
