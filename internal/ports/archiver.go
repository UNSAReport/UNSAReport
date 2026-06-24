package ports

// Archiver abstracts creating zip archives from directories or file lists.
type Archiver interface {
	ArchiveDir(zipPath, srcDir string) error
	ArchiveFiles(zipPath, baseDir string, files []string) error
}
