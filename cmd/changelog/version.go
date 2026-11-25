package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"sarak.sh/mikluko/changelog/v2/pkg/changelog"
)

// versionCmd prints the version the newest entry names. It reads the
// changelog alone, with no git history and no checks, so a build that sees
// only the file, as a Nix derivation does, can take its version from it.
func versionCmd(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(stderr)
	path := fs.String("changelog", "CHANGELOG.md", "path to the changelog")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("version takes no arguments, got %q", fs.Args())
	}
	src, err := os.ReadFile(*path)
	if err != nil {
		return err
	}
	latest, ok := changelog.Parse(src).Latest()
	if !ok {
		return fmt.Errorf("%s names no version", *path)
	}
	_, err = fmt.Fprintln(stdout, latest.Semver.String())
	return err
}
