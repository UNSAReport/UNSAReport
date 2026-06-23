package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/UNSAReport/UNSAReport/internal/adapters/config"
	"github.com/UNSAReport/UNSAReport/internal/ports"
)

type ComponentService struct {
	Fetcher  ports.TemplateFetcher
	FS       ports.FileSystem
	Config   ports.ConfigStore
	Registry ports.ComponentRegistry
}

func NewComponentService(f ports.TemplateFetcher, fs ports.FileSystem, c ports.ConfigStore, r ports.ComponentRegistry) *ComponentService {
	return &ComponentService{
		Fetcher:  f,
		FS:       fs,
		Config:   c,
		Registry: r,
	}
}

type ComponentInstallResult struct {
	Name            string
	RangeSpec       string
	ResolvedVersion string
	Status          string
}

func (s *ComponentService) Add(ctx context.Context, name, rangeSpec string, force bool) error {
	resolvedVersion, info, cv, err := s.Registry.ResolveVersion(name, rangeSpec)
	if err != nil {
		return fmt.Errorf("resolve version: %w", err)
	}
	return s.addResolved(ctx, name, force, resolvedVersion, info, cv)
}

func (s *ComponentService) AddFromManifest(ctx context.Context, components map[string]string) ([]ComponentInstallResult, error) {
	var results []ComponentInstallResult
	visited := make(map[string]bool)

	for name, rangeSpec := range components {
		resolvedVersion, info, cv, err := s.Registry.ResolveVersion(name, rangeSpec)
		if err != nil {
			return results, fmt.Errorf("resolve %s: %w", name, err)
		}

		if err := s.resolveAndInstallDeps(ctx, name, resolvedVersion, info, cv, visited); err != nil {
			return results, fmt.Errorf("add %s: %w", name, err)
		}

		results = append(results, ComponentInstallResult{
			Name:            name,
			RangeSpec:       rangeSpec,
			ResolvedVersion: resolvedVersion.String(),
			Status:          "installed",
		})
	}

	return results, nil
}

func (s *ComponentService) resolveAndInstallDeps(ctx context.Context, name string, resolvedVersion *semver.Version, info *ports.ComponentInfo, cv *ports.ComponentVersion, visited map[string]bool) error {
	if visited[name] {
		return nil
	}
	visited[name] = true

	for _, dep := range cv.Dependencies {
		depName, depRange := parseComponentDep(dep)
		if visited[depName] {
			continue
		}

		depVersion, depInfo, depCv, err := s.Registry.ResolveVersion(depName, depRange)
		if err != nil {
			return fmt.Errorf("resolve dependency %s: %w", depName, err)
		}

		if err := s.resolveAndInstallDeps(ctx, depName, depVersion, depInfo, depCv, visited); err != nil {
			return err
		}
	}

	return s.addResolved(ctx, name, false, resolvedVersion, info, cv)
}

func (s *ComponentService) addResolved(ctx context.Context, name string, force bool, resolvedVersion *semver.Version, info *ports.ComponentInfo, cv *ports.ComponentVersion) error {
	cwd, err := s.FS.Getwd()
	if err != nil {
		return fmt.Errorf("get cwd: %w", err)
	}

	projectRoot, cfg, ok, err := s.Config.FindProjectRoot(cwd)
	if err != nil {
		return fmt.Errorf("find project root: %w", err)
	}
	if !ok {
		return fmt.Errorf("unsareport.json not found. Are you in a project directory?")
	}

	var componentNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !componentNameRegex.MatchString(name) {
		return fmt.Errorf("invalid component name %q: must contain only alphanumeric characters, underscores, or dashes", name)
	}

	localPath := filepath.Join(projectRoot, "components", name+".typ")

	if s.FS.FileExists(localPath) && !force {
		localData, err := s.FS.ReadFile(localPath)
		if err != nil {
			return fmt.Errorf("read local file: %w", err)
		}

		lf, err := s.Config.ReadLockfile(projectRoot)
		if err != nil {
			return fmt.Errorf("read lockfile: %w", err)
		}

		lockKey := "components/" + name + ".typ"
		if existingPkg, exists := lf.Packages[lockKey]; exists {
			localHash := config.ComputeIntegrity(localData)
			if localHash != existingPkg.Integrity {
				fmt.Fprintf(os.Stdout, "Warning: Local modifications detected in %s. Overwrite? (y/N): ", name+".typ")
				var answer string
				fmt.Scanln(&answer)
				if strings.ToLower(strings.TrimSpace(answer)) != "y" {
					fmt.Fprintf(os.Stdout, "Skipped: %s\n", name)
					return nil
				}
			}
		}
	}

	data, err := s.Registry.FetchComponentFile(*info, cv)
	if err != nil {
		return fmt.Errorf("fetch component: %w", err)
	}

	if len(data) == 0 {
		return fmt.Errorf("component %s returned empty data", name)
	}

	for _, dep := range cv.Dependencies {
		depName, depRange := parseComponentDep(dep)
		_, _, _, depErr := s.Registry.ResolveVersion(depName, depRange)
		if depErr != nil {
			return fmt.Errorf("dependency %s@%s is not satisfiable: %w", depName, depRange, depErr)
		}
	}

	if err := s.FS.EnsureDir(filepath.Dir(localPath)); err != nil {
		return fmt.Errorf("ensure components dir: %w", err)
	}

	if err := s.FS.WriteFileAtomic(localPath, data, 0o644); err != nil {
		return fmt.Errorf("write component: %w", err)
	}

	lf, err := s.Config.ReadLockfile(projectRoot)
	if err != nil {
		return fmt.Errorf("read lockfile: %w", err)
	}

	if lf.Packages == nil {
		lf.Packages = make(map[string]ports.LockfilePackage)
	}

	lockKey := "components/" + name + ".typ"
	lf.Packages[lockKey] = ports.LockfilePackage{
		Version:   resolvedVersion.String(),
		Resolved:  fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", ports.DefaultComponentRepo, ports.DefaultRef, cv.Path),
		Integrity: config.ComputeIntegrity(data),
	}

	if err := s.Config.WriteLockfile(projectRoot, lf); err != nil {
		return fmt.Errorf("write lockfile: %w", err)
	}

	if cfg.Components == nil {
		cfg.Components = make(map[string]ports.ComponentConfigEntry)
	}
	cfg.Components[name] = ports.ComponentConfigEntry{
		Version:     resolvedVersion.String(),
		InstalledAt: time.Now().Format(time.RFC3339),
	}

	if err := s.Config.WriteConfig(projectRoot, cfg); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

func (s *ComponentService) Remove(ctx context.Context, name string) error {
	cwd, err := s.FS.Getwd()
	if err != nil {
		return fmt.Errorf("get cwd: %w", err)
	}

	projectRoot, cfg, ok, err := s.Config.FindProjectRoot(cwd)
	if err != nil {
		return fmt.Errorf("find project root: %w", err)
	}
	if !ok {
		return fmt.Errorf("unsareport.json not found. Are you in a project directory?")
	}

	localPath := filepath.Join(projectRoot, "components", name+".typ")
	if s.FS.FileExists(localPath) {
		if err := s.FS.Remove(localPath); err != nil {
			return fmt.Errorf("delete component file: %w", err)
		}
	}

	lf, err := s.Config.ReadLockfile(projectRoot)
	if err != nil {
		return fmt.Errorf("read lockfile: %w", err)
	}

	lockKey := "components/" + name + ".typ"
	delete(lf.Packages, lockKey)

	if err := s.Config.WriteLockfile(projectRoot, lf); err != nil {
		return fmt.Errorf("write lockfile: %w", err)
	}

	delete(cfg.Components, name)

	if err := s.Config.WriteConfig(projectRoot, cfg); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

func (s *ComponentService) List(ctx context.Context) error {
	components, err := s.Registry.ListComponents()
	if err != nil {
		return fmt.Errorf("list components: %w", err)
	}

	cwd, err := s.FS.Getwd()
	if err != nil {
		return fmt.Errorf("get cwd: %w", err)
	}

	_, cfg, ok, _ := s.Config.FindProjectRoot(cwd)

	installed := make(map[string]string)
	if ok && cfg.Components != nil {
		for name, entry := range cfg.Components {
			installed[name] = entry.Version
		}
	}

	fmt.Fprintf(os.Stdout, "%-20s %-10s %-40s\n", "NAME", "INSTALLED", "DESCRIPTION")
	fmt.Fprintln(os.Stdout, strings.Repeat("-", 70))

	for _, c := range components {
		status := "no"
		if v, ok := installed[c.Name]; ok {
			status = v
		}
		fmt.Fprintf(os.Stdout, "%-20s %-10s %-40s\n", c.Name, status, c.Description)
	}

	return nil
}

func (s *ComponentService) Update(ctx context.Context, name string) error {
	cwd, err := s.FS.Getwd()
	if err != nil {
		return fmt.Errorf("get cwd: %w", err)
	}

	projectRoot, cfg, ok, err := s.Config.FindProjectRoot(cwd)
	if err != nil {
		return fmt.Errorf("find project root: %w", err)
	}
	if !ok {
		return fmt.Errorf("unsareport.json not found. Are you in a project directory?")
	}

	if name != "" {
		return s.updateSingle(ctx, projectRoot, cfg, name)
	}

	if cfg.Components == nil {
		fmt.Fprintln(os.Stdout, "No components installed.")
		return nil
	}

	for compName := range cfg.Components {
		if err := s.updateSingle(ctx, projectRoot, cfg, compName); err != nil {
			fmt.Fprintf(os.Stdout, "Warning: failed to update %s: %v\n", compName, err)
		}
	}

	return nil
}

func (s *ComponentService) updateSingle(ctx context.Context, projectRoot string, cfg ports.UnsareportConfig, name string) error {
	entry, ok := cfg.Components[name]
	if !ok {
		return fmt.Errorf("component %q not installed", name)
	}

	latestVersion, _, _, err := s.Registry.ResolveVersion(name, "latest")
	if err != nil {
		return fmt.Errorf("resolve latest version: %w", err)
	}

	if entry.Version == latestVersion.String() {
		fmt.Fprintf(os.Stdout, "%s: already up to date (%s)\n", name, entry.Version)
		return nil
	}

	fmt.Fprintf(os.Stdout, "%s: updating %s -> %s\n", name, entry.Version, latestVersion.String())

	if err := s.Add(ctx, name, "latest", true); err != nil {
		return fmt.Errorf("update component: %w", err)
	}

	return nil
}

func parseComponentDep(dep string) (name, rangeSpec string) {
	if i := strings.Index(dep, "@"); i != -1 {
		return dep[:i], dep[i+1:]
	}
	return dep, "*"
}
