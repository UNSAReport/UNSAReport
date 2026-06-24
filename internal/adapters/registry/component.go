package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Masterminds/semver/v3"
	"github.com/UNSAReport/UNSAReport/internal/ports"
)

type registryFile struct {
	Components map[string]registryComponentEntry `json:"components"`
}

type registryComponentEntry struct {
	Description string                          `json:"description"`
	DistTags    map[string]string               `json:"dist-tags"`
	Versions    map[string]registryVersionEntry `json:"versions"`
}

type registryVersionEntry struct {
	Path         string   `json:"path"`
	Dependencies []string `json:"dependencies"`
}

type ComponentRegistryAdapter struct {
	fetcher ports.TemplateFetcher
}

func NewComponentRegistry(fetcher ports.TemplateFetcher) *ComponentRegistryAdapter {
	return &ComponentRegistryAdapter{
		fetcher: fetcher,
	}
}

func (a *ComponentRegistryAdapter) fetchRegistry() (*registryFile, error) {
	ctx := context.Background()
	data, err := a.fetcher.FetchRaw(ctx, ports.DefaultComponentRepo, ports.DefaultRef, "registry.json")
	if err != nil {
		return nil, fmt.Errorf("fetch registry.json: %w", err)
	}

	var reg registryFile
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parse registry.json: %w", err)
	}

	return &reg, nil
}

func (a *ComponentRegistryAdapter) convertComponent(name string, entry registryComponentEntry) ports.ComponentInfo {
	distTags := make(map[string]*semver.Version)
	for tag, vStr := range entry.DistTags {
		if v, err := semver.NewVersion(vStr); err == nil {
			distTags[tag] = v
		}
	}

	versions := make(map[string]*ports.ComponentVersion)
	for vStr, vEntry := range entry.Versions {
		v, err := semver.NewVersion(vStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: component %q has invalid version %q: %v\n", name, vStr, err) //nolint:errcheck // diagnostic output
			continue
		}
		versions[vStr] = &ports.ComponentVersion{
			Version:      v,
			Path:         vEntry.Path,
			Dependencies: vEntry.Dependencies,
		}
	}

	return ports.ComponentInfo{
		Name:        name,
		Description: entry.Description,
		DistTags:    distTags,
		Versions:    versions,
	}
}

func (a *ComponentRegistryAdapter) ListComponents() ([]ports.ComponentInfo, error) {
	reg, err := a.fetchRegistry()
	if err != nil {
		return nil, err
	}

	components := make([]ports.ComponentInfo, 0, len(reg.Components))
	for name, entry := range reg.Components {
		components = append(components, a.convertComponent(name, entry))
	}

	return components, nil
}

func (a *ComponentRegistryAdapter) GetComponent(name string) (ports.ComponentInfo, error) {
	reg, err := a.fetchRegistry()
	if err != nil {
		return ports.ComponentInfo{}, err
	}

	entry, ok := reg.Components[name]
	if !ok {
		return ports.ComponentInfo{}, fmt.Errorf("component %q not found", name)
	}

	return a.convertComponent(name, entry), nil
}

func (a *ComponentRegistryAdapter) ResolveVersion(name string, rangeSpec string) (*semver.Version, *ports.ComponentInfo, *ports.ComponentVersion, error) {
	info, err := a.GetComponent(name)
	if err != nil {
		return nil, nil, nil, err
	}

	distTags := make(map[string]*semver.Version)
	for tag, v := range info.DistTags {
		distTags[tag] = v
	}

	availableVersions := make(map[string]*semver.Version)
	for vStr, cv := range info.Versions {
		availableVersions[vStr] = cv.Version
	}

	resolved, err := resolveVersionFromMap(availableVersions, distTags, rangeSpec)
	if err != nil {
		return nil, nil, nil, err
	}

	cv := info.Versions[resolved.String()]
	return resolved, &info, cv, nil
}

func (a *ComponentRegistryAdapter) FetchComponentFile(info ports.ComponentInfo, cv *ports.ComponentVersion) ([]byte, error) {
	ctx := context.Background()
	return a.fetcher.FetchRaw(ctx, ports.DefaultComponentRepo, ports.DefaultRef, cv.Path)
}
