package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sarak.sh/mikluko/changelog/v2/pkg/changelog"
)

const flakeChangelog = "# Changelog\n\n## [Unreleased]\n\n## [1.2.0] - 2026-03-01\n\n### Added\n\n- a thing\n"

// fakeNix puts on PATH, and alone there, a nix running script, and returns the
// file it records its arguments and working directory in. Where script is ""
// PATH holds no nix at all.
func fakeNix(t *testing.T, script string) (calls string) {
	t.Helper()
	bin := t.TempDir()
	calls = filepath.Join(t.TempDir(), "calls")
	t.Setenv("PATH", bin)
	if script == "" {
		return calls
	}
	body := "#!/bin/sh\npwd -P > " + calls + "\nprintf '%s\\n' \"$@\" >> " + calls + "\n" + script + "\n"
	if err := os.WriteFile(filepath.Join(bin, "nix"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return calls
}

// flakeRepo writes the changelog beside a flake.nix, or without one, and
// returns the changelog's path.
func flakeRepo(t *testing.T, flake bool) string {
	t.Helper()
	path := write(t, flakeChangelog)
	if flake {
		if err := os.WriteFile(filepath.Join(filepath.Dir(path), "flake.nix"), []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func manifestOutcome(t *testing.T, path, attr string, opts changelog.Options) ([]changelog.Finding, string) {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var log bytes.Buffer
	findings, err := lintManifest(path, attr, changelog.Parse(src), opts, &log, &log)
	if err != nil {
		t.Fatal(err)
	}
	return findings, log.String()
}

func TestTheFlakeIsEvaluatedInTheChangelogsDirectory(t *testing.T) {
	calls := fakeNix(t, "printf 1.2.0")
	path := flakeRepo(t, true)

	if findings, log := manifestOutcome(t, path, "baox", changelog.Options{}); len(findings) != 0 {
		t.Fatalf("findings %v, want none: the flake states the newest entry's version; log: %s", findings, log)
	}
	dir, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	want := dir + "\n--extra-experimental-features\nnix-command flakes\neval\n--raw\n--no-write-lock-file\n.#baox.version\n"
	if got := string(read(t, calls)); got != want {
		t.Errorf("nix ran as\n%s\nwant\n%s", got, want)
	}
}

func TestAFlakeBehindTheNewestEntryTurnsTheBuildRed(t *testing.T) {
	fakeNix(t, "printf 1.1.0")
	path := flakeRepo(t, true)

	findings, log := manifestOutcome(t, path, "default", changelog.Options{})
	if len(findings) != 1 || findings[0].Check != changelog.CheckManifestVersion || findings[0].Line != 5 || !red(findings, changelog.Error) {
		t.Fatalf("findings %v, want one %s at error on the newest entry's heading", findings, changelog.CheckManifestVersion)
	}
	for _, w := range []string{"1.2.0", `"1.1.0"`, "default.version"} {
		if !strings.Contains(findings[0].Msg, w) {
			t.Errorf("message %q does not name %s", findings[0].Msg, w)
		}
	}
	if !strings.Contains(log, path+":5:") {
		t.Errorf("log %q does not report against the changelog's line 5", log)
	}
}

// A check that cannot run and says nothing passes the release commit that
// forgot the bump, which is the one it exists to stop.
func TestAFlakeThatCannotBeEvaluatedIsAFinding(t *testing.T) {
	for _, tc := range []struct {
		name, script string
		want         []string
	}{
		{"no nix on PATH", "", []string{"nix is not on PATH", "default.version"}},
		{"no such package", "echo \"error: flake 'git+file:///src' does not provide attribute 'packages.x86_64-linux.default'\" >&2\nexit 1",
			[]string{"default.version", "does not provide attribute 'packages.x86_64-linux.default'"}},
		{"an error over several lines", "printf 'error:\\n       … while evaluating\\n\\n       error: attribute %s missing\\n' \"'version'\" >&2\nexit 1",
			[]string{"error: … while evaluating error: attribute 'version' missing"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fakeNix(t, tc.script)
			path := flakeRepo(t, true)

			findings, _ := manifestOutcome(t, path, "default", changelog.Options{})
			if len(findings) != 1 || findings[0].Check != changelog.CheckManifestVersion || !red(findings, changelog.Error) {
				t.Fatalf("findings %v, want one %s at error", findings, changelog.CheckManifestVersion)
			}
			for _, w := range tc.want {
				if !strings.Contains(findings[0].Msg, w) {
					t.Errorf("message %q does not carry %q", findings[0].Msg, w)
				}
			}
		})
	}
}

func TestNixDoesNotRunWhereThereIsNothingToCompare(t *testing.T) {
	off := changelog.DefaultSeverities()
	off[changelog.CheckManifestVersion] = changelog.Off
	for _, tc := range []struct {
		name  string
		flake bool
		doc   string
		opts  changelog.Options
	}{
		{name: "no flake.nix", doc: flakeChangelog},
		{name: "the check switched off", flake: true, doc: flakeChangelog, opts: changelog.Options{Severities: off}},
		{name: "no entry naming a version", flake: true, doc: "# Changelog\n\n## [Unreleased]\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := fakeNix(t, "printf 9.9.9")
			path := flakeRepo(t, tc.flake)
			if err := os.WriteFile(path, []byte(tc.doc), 0o644); err != nil {
				t.Fatal(err)
			}
			if findings, log := manifestOutcome(t, path, "default", tc.opts); len(findings) != 0 || log != "" {
				t.Fatalf("findings %v and log %q, want none", findings, log)
			}
			if _, err := os.Stat(calls); err == nil {
				t.Error("nix ran")
			}
		})
	}
}
