package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sarak.sh/mikluko/changelog/v2/pkg/changelog"
)

// lintManifest reports, against the changelog at path, the flake beside it
// stating a version other than the one doc's newest entry names. It runs nix
// only where the check is on, a flake.nix sits there, and doc names a version.
func lintManifest(path, attr string, doc *changelog.Changelog, opts changelog.Options, log, annotations io.Writer) ([]changelog.Finding, error) {
	if severity(opts, changelog.CheckManifestVersion) == changelog.Off {
		return nil, nil
	}
	if _, ok := doc.Latest(); !ok {
		return nil, nil
	}
	m, ok := readFlake(filepath.Dir(path), attr)
	if !ok {
		return nil, nil
	}
	findings := doc.LintManifest(m, opts)
	return findings, report(path, findings, log, annotations)
}

// readFlake reads the version of the package attr names in the flake in dir,
// by evaluating it: a flake may compute the version, so the file is never read
// as text. ok is false where dir holds no flake.nix.
//
// The flake is the one git tracks, as for nix eval .#attr run in dir, and its
// lock file is left as it is.
func readFlake(dir, attr string) (m changelog.Manifest, ok bool) {
	file := filepath.Join(dir, "flake.nix")
	m = changelog.Manifest{File: file, Field: attr + ".version"}
	if _, err := os.Stat(file); errors.Is(err, fs.ErrNotExist) {
		return m, false
	} else if err != nil {
		m.Err = err
		return m, true
	}
	nix, err := exec.LookPath("nix")
	if err != nil {
		m.Err = fmt.Errorf("nix is not on PATH; put it on the runner, or switch %s off", changelog.CheckManifestVersion)
		return m, true
	}
	cmd := exec.Command(nix, "--extra-experimental-features", "nix-command flakes",
		"eval", "--raw", "--no-write-lock-file", ".#"+m.Field)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		m.Err = fmt.Errorf("nix eval .#%s: %w: %s", m.Field, err, oneLine(stderr.String()))
		return m, true
	}
	m.Version = string(out)
	return m, true
}

// oneLine collapses every run of whitespace in s, newlines included, to one
// space: a finding is one line in the log and one workflow command in an
// annotation.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
