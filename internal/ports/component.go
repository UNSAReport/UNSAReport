package ports

import "github.com/Masterminds/semver/v3"

// ComponentVersion represents a specific version of a component with its file path and dependency list.
type ComponentVersion struct {
	Version      *semver.Version `json:"-"`
	Path         string          `json:"path"`
	Dependencies []string        `json:"dependencies"`
}

// ComponentInfo describes a component including its available versions, dist-tags, and metadata.
type ComponentInfo struct {
	Name        string                       `json:"-"`
	Description string                       `json:"description"`
	DistTags    map[string]*semver.Version   `json:"dist-tags"`
	Versions    map[string]*ComponentVersion `json:"versions"`
}

// ComponentRegistry abstracts a registry for discovering and resolving component versions.
type ComponentRegistry interface {
	ListComponents() ([]ComponentInfo, error)
	GetComponent(name string) (ComponentInfo, error)
	ResolveVersion(name string, rangeSpec string) (*semver.Version, *ComponentInfo, *ComponentVersion, error)
	FetchComponentFile(info ComponentInfo, cv *ComponentVersion) ([]byte, error)
}

// ComponentConfigEntry records the installed version and installation timestamp of a component.
type ComponentConfigEntry struct {
	Version     string `json:"version"`
	InstalledAt string `json:"installed_at"`
}

// LockfilePackage represents a locked dependency entry with its resolved version and integrity hash.
type LockfilePackage struct {
	Version   string `json:"version"`
	Resolved  string `json:"resolved"`
	Integrity string `json:"integrity"`
}

// LockfileTemplateFile records the integrity hash for a single template file in the lockfile.
type LockfileTemplateFile struct {
	Integrity string `json:"integrity"`
}

// LockfileTemplate records the locked template name, version, and file integrity hashes.
type LockfileTemplate struct {
	Name    string                          `json:"name"`
	Version string                          `json:"version"`
	Files   map[string]LockfileTemplateFile `json:"files"`
}

// Lockfile is the top-level lockfile structure that pins template and component versions for reproducible builds.
type Lockfile struct {
	LockfileVersion string                     `json:"lockfile_version"`
	RemoteRegistry  string                     `json:"remote_registry,omitempty"`
	Template        *LockfileTemplate          `json:"template,omitempty"`
	Packages        map[string]LockfilePackage `json:"packages"`
}
