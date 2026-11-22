package changelog

// Manifest is the version a package manifest beside the changelog states.
// Reading it is the caller's job, as for Git; this package compares.
type Manifest struct {
	// File is the manifest as a finding names it, as in flake.nix.
	File string
	// Field is where in File the version was read, as in default.version.
	Field string
	// Version is the version read, spelled as the manifest spells it.
	Version string
	// Err is why the version could not be read, where it could not. It is a
	// finding rather than a pass: a check that skips where it cannot run lets
	// through the release commit that forgot the bump.
	Err error
}

// LintManifest reports m stating a version other than the one the newest entry
// names, or no version that could be read. It is silent where no entry names a
// version.
//
// It is not part of Lint, so the fragments-release forecast does not run it: a
// release not yet cut has no manifest bump yet, since the release commit makes
// it.
func (c *Changelog) LintManifest(m Manifest, opts Options) []Finding {
	latest, ok := c.Latest()
	if !ok {
		return nil
	}
	severities := opts.Severities
	if severities == nil {
		severities = DefaultSeverities()
	}
	f := &findings{severities: severities}
	want := latest.Semver.String()
	switch {
	case m.Err != nil:
		f.add(CheckManifestVersion, 0,
			"%s states no version at %s that can be read, so it cannot be compared with %s, the newest entry: %v",
			m.File, m.Field, want, m.Err)
	case m.Version != want:
		f.add(CheckManifestVersion, latest.Line,
			"version %s is the newest entry and %s states %q at %s; the commit cutting the release sets the two alike",
			want, m.File, m.Version, m.Field)
	}
	return f.out
}
