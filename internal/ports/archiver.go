package ports

type Archiver interface {
	ArchiveDir(zipPath, srcDir string) error
}
