package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sarak.sh/mikluko/changelog/v2/pkg/git"
	"sarak.sh/mikluko/changelog/v2/pkg/semver"
)

func TestNextPrerelease(t *testing.T) {
	v := semver.Version{Major: 1, Minor: 3}
	for _, tc := range []struct {
		name string
		tags []string
		want string
	}{
		{"no tags", nil, "1.3.0-pre.1"},
		{"another version's pre-releases", []string{"v1.2.0-pre.4", "v1.3.1-pre.2", "v1.2.0"}, "1.3.0-pre.1"},
		{"after the highest", []string{"v1.3.0-pre.1", "v1.3.0-pre.10", "v1.3.0-pre.2"}, "1.3.0-pre.11"},
		{"other spellings", []string{"v1.3.0-pre5", "v1.3.0-rc.7", "v1.3.0-pre.x", "v1.3.0-pre.1.1"}, "1.3.0-pre.1"},
		{"without the v", []string{"1.3.0-pre.3"}, "1.3.0-pre.4"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tags := make([]git.Tag, 0, len(tc.tags))
			for _, name := range tc.tags {
				tags = append(tags, git.Tag{Name: name})
			}
			if got := nextPrerelease(v, tags).String(); got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

// prereleaseRepo is a repository whose changelog was released at 1.2.3 and
// carries a fragment calling for a minor release, with the tags given.
func prereleaseRepo(t *testing.T, tags ...string) (path, frags string) {
	t.Helper()
	dir := fixture(t)
	path = filepath.Join(dir, "CHANGELOG.md")
	frags = filepath.Join(dir, ".changelog")
	if err := os.WriteFile(path, []byte(unreleased), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(frags, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(frags, "new.md"), []byte("### Added\n\n- a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitcmd(t, dir, "add", ".")
	gitcmd(t, dir, "commit", "-m", "notes")
	for _, tag := range tags {
		gitcmd(t, dir, "tag", tag)
	}
	return path, frags
}

func TestReleaseCmdPrerelease(t *testing.T) {
	path, frags := prereleaseRepo(t, "v1.2.3", "v1.3.0-pre.1", "v1.3.0-pre.2")
	var out bytes.Buffer
	if err := releaseCmd([]string{"-changelog", path, "-fragments", frags, "-prerelease"}, &out, os.Stderr, fixed(t)); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"version=1.3.0-pre.3\n", "prerelease=true\n", "### Added\n\n- a\n\n### Fixed\n\n- f\n"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("outputs lack %q:\n%s", want, out.String())
		}
	}
	if got, _ := os.ReadFile(path); string(got) != unreleased {
		t.Errorf("the changelog changed:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(frags, "new.md")); err != nil {
		t.Errorf("the fragment is gone: %v", err)
	}
}

func TestReleaseCmdPrereleaseOfAGivenVersion(t *testing.T) {
	path, frags := prereleaseRepo(t, "v2.0.0-pre.1")
	var out bytes.Buffer
	if err := releaseCmd([]string{"-changelog", path, "-fragments", frags, "-prerelease", "-bump", "major"}, &out, os.Stderr, fixed(t)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "version=2.0.0-pre.2\n") {
		t.Errorf("outputs: %s", out.String())
	}
}

func TestReleaseCmdPrereleaseRefuses(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"an open entry", []string{"-open"}},
		{"a date", []string{"-date", "2020-02-02"}},
		{"a pre-release version", []string{"-version", "1.3.0-pre.1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path, frags := prereleaseRepo(t)
			args := append([]string{"-changelog", path, "-fragments", frags, "-prerelease"}, tc.args...)
			if err := releaseCmd(args, os.Stdout, os.Stderr, fixed(t)); err == nil {
				t.Error("no error")
			}
		})
	}
}

func TestReleaseCmdPrereleaseRefusesAShallowCheckout(t *testing.T) {
	origin, _ := prereleaseRepo(t, "v1.3.0-pre.1")
	clone := filepath.Join(t.TempDir(), "clone")
	gitcmd(t, t.TempDir(), "clone", "--depth", "1", "file://"+filepath.Dir(origin), clone)
	path := filepath.Join(clone, "CHANGELOG.md")
	err := releaseCmd([]string{"-changelog", path, "-fragments", filepath.Join(clone, ".changelog"), "-prerelease"}, os.Stdout, os.Stderr, fixed(t))
	if err == nil || !strings.Contains(err.Error(), "shallow") {
		t.Errorf("got %v, want the shallow checkout refused", err)
	}
}
