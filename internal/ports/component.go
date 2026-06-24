package ports

import "github.com/Masterminds/semver/v3"

type ComponentVersion struct {
	Version      *semver.Version `json:"-"`
	Path         string          `json:"path"`
	Dependencies []string        `json:"dependencies"`
}

type ComponentInfo struct {
	Name        string                       `json:"-"`
	Description string                       `json:"description"`
	DistTags    map[string]*semver.Version   `json:"dist-tags"`
	Versions    map[string]*ComponentVersion `json:"versions"`
}

type ComponentRegistry interface {
	ListComponents() ([]ComponentInfo, error)
	GetComponent(name string) (ComponentInfo, error)
	ResolveVersion(name string, rangeSpec string) (*semver.Version, *ComponentInfo, *ComponentVersion, error)
	FetchComponentFile(info ComponentInfo, cv *ComponentVersion) ([]byte, error)
}

type ComponentConfigEntry struct {
	Version     string `json:"version"`
	InstalledAt string `json:"installed_at"`
}

type LockfilePackage struct {
	Version   string `json:"version"`
	Resolved  string `json:"resolved"`
	Integrity string `json:"integrity"`
}

type LockfileTemplateFile struct {
	Integrity string `json:"integrity"`
}

type LockfileTemplate struct {
	Name    string                          `json:"name"`
	Version string                          `json:"version"`
	Files   map[string]LockfileTemplateFile `json:"files"`
}

type Lockfile struct {
	LockfileVersion string                     `json:"lockfile_version"`
	RemoteRegistry  string                     `json:"remote_registry,omitempty"`
	Template        *LockfileTemplate          `json:"template,omitempty"`
	Packages        map[string]LockfilePackage `json:"packages"`
}
