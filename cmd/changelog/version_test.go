package main

import (
	"bytes"
	"os"
	"testing"
)

func TestVersionCmdPrintsTheNewestVersion(t *testing.T) {
	path := write(t, unreleased)
	var out bytes.Buffer
	if err := versionCmd([]string{"-changelog", path}, &out, os.Stderr); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "1.2.3\n" {
		t.Errorf("printed %q, want %q", got, "1.2.3\n")
	}
}

func TestVersionCmdReadsNoGit(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("CHANGELOG.md", []byte(unreleased), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := versionCmd(nil, &out, os.Stderr); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "1.2.3\n" {
		t.Errorf("printed %q, want %q", got, "1.2.3\n")
	}
}

func TestVersionCmdRefusesAChangelogNamingNoVersion(t *testing.T) {
	path := write(t, "# Changelog\n\n## [Unreleased]\n\n### Fixed\n\n- f\n")
	var out bytes.Buffer
	if err := versionCmd([]string{"-changelog", path}, &out, os.Stderr); err == nil {
		t.Fatal("no error for a changelog naming no version")
	}
	if out.Len() != 0 {
		t.Errorf("printed %q on error", out.String())
	}
}

func TestVersionCmdTakesNoArguments(t *testing.T) {
	path := write(t, unreleased)
	if err := versionCmd([]string{"-changelog", path, "extra"}, &bytes.Buffer{}, os.Stderr); err == nil {
		t.Fatal("no error for a stray argument")
	}
}
