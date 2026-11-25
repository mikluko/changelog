package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"sarak.sh/mikluko/changelog/v2/pkg/changelog"
	"sarak.sh/mikluko/changelog/v2/pkg/git"
	"sarak.sh/mikluko/changelog/v2/pkg/semver"
)

// releasePrerelease prints the outputs release would, for the next release
// pre-release of the version the release would take, and writes nothing.
//
// The tags are read from the repository holding the changelog, and a shallow
// checkout is refused: it may lack the tags the number is counted from.
func releasePrerelease(path, fragments string, given *semver.Version, level semver.Level, stdout io.Writer) error {
	tags, err := prereleaseTags(path)
	if err != nil {
		return err
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, frags, err := readFragments(fragments)
	if err != nil {
		return err
	}
	if level != 0 {
		v, err := changelog.Parse(src).Bump(level)
		if err != nil {
			return err
		}
		given = &v
	}
	out, v, err := changelog.ReleaseFragments(src, frags, given, "")
	if err != nil {
		return err
	}
	if v.Prerelease() {
		return fmt.Errorf("-prerelease numbers the pre-release itself, so the version it is one of is final, not %s", v)
	}
	entry, _ := changelog.Parse(out).Find(v.String())
	return emit(releaseOutputs(nextPrerelease(v, tags), entry.Body), stdout)
}

// prereleaseTags is the tags of the repository holding the changelog at path.
func prereleaseTags(path string) ([]git.Tag, error) {
	repo, err := git.Open(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	shallow, err := repo.Shallow()
	if err != nil {
		return nil, err
	}
	if shallow {
		return nil, errors.New("-prerelease counts the tags, and a shallow checkout may lack them")
	}
	return repo.Tags()
}

// nextPrerelease is v-pre.N, where N is one above the highest N among the tags
// naming v-pre.N, or 1 where none does. A tag spelling the pre-release any other
// way, as in v-pre1 or v-rc.1, is not counted.
func nextPrerelease(v semver.Version, tags []git.Tag) semver.Version {
	var highest uint64
	for _, t := range tags {
		w, ok := t.Semver()
		if !ok || w.Major != v.Major || w.Minor != v.Minor || w.Patch != v.Patch || len(w.Pre) != 2 || w.Pre[0] != "pre" {
			continue
		}
		if n, err := strconv.ParseUint(w.Pre[1], 10, 64); err == nil && n > highest {
			highest = n
		}
	}
	return semver.Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch, Pre: []string{"pre", strconv.FormatUint(highest+1, 10)}}
}
