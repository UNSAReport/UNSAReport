package registry

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/Masterminds/semver/v3"
)

func resolveVersionFromMap(
	availableVersions map[string]*semver.Version,
	distTags map[string]*semver.Version,
	rangeSpec string,
) (*semver.Version, error) {
	switch rangeSpec {
	case "latest", "":
		if latest, ok := distTags["latest"]; ok {
			return latest, nil
		}
		return nil, fmt.Errorf("no 'latest' dist-tag found")

	case "*":
		return resolveLatestFromMap(availableVersions)
	}

	constraint, err := semver.NewConstraint(rangeSpec)
	if err != nil {
		return nil, fmt.Errorf("invalid version range %q: %w", rangeSpec, err)
	}

	var candidates []*semver.Version
	for _, v := range availableVersions {
		ok, _ := constraint.Validate(v)
		if ok {
			candidates = append(candidates, v)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no version matching %q", rangeSpec)
	}

	slices.SortFunc(candidates, func(a, b *semver.Version) int {
		return cmp.Compare(a.String(), b.String())
	})

	return candidates[len(candidates)-1], nil
}

func resolveLatestFromMap(versions map[string]*semver.Version) (*semver.Version, error) {
	var all []*semver.Version
	for _, v := range versions {
		if v.Prerelease() != "" {
			continue
		}
		all = append(all, v)
	}

	if len(all) == 0 {
		return nil, fmt.Errorf("no stable versions found")
	}

	slices.SortFunc(all, func(a, b *semver.Version) int {
		return cmp.Compare(a.String(), b.String())
	})

	return all[len(all)-1], nil
}
