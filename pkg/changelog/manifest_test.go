package changelog

import (
	"errors"
	"strings"
	"testing"
)

const manifestDoc = "# Changelog\n\n## [Unreleased]\n\n## [1.3.0-rc.1] - 2026-03-02\n\n### Added\n\n- a thing\n\n## [1.2.0] - 2026-03-01\n\n### Added\n\n- a thing\n"

func TestLintManifest(t *testing.T) {
	flake := func(version string, err error) Manifest {
		return Manifest{File: "flake.nix", Field: "default.version", Version: version, Err: err}
	}
	for _, tc := range []struct {
		name string
		doc  string
		m    Manifest
		off  bool
		line int
		want []string
	}{
		{name: "the version the newest entry names", doc: manifestDoc, m: flake("1.3.0-rc.1", nil)},
		{name: "a version behind it", doc: manifestDoc, m: flake("1.2.0", nil), line: 5, want: []string{"1.3.0-rc.1", `"1.2.0"`, "flake.nix", "default.version"}},
		{name: "a version spelled as a tag", doc: manifestDoc, m: flake("v1.3.0-rc.1", nil), line: 5, want: []string{`"v1.3.0-rc.1"`}},
		{name: "a version that cannot be read", doc: manifestDoc, m: flake("", errors.New("nix is not on PATH")), want: []string{"flake.nix", "default.version", "1.3.0-rc.1", "nix is not on PATH"}},
		{name: "no entry naming a version", doc: "# Changelog\n\n## [Unreleased]\n", m: flake("", errors.New("nix is not on PATH"))},
		{name: "switched off", doc: manifestDoc, m: flake("1.2.0", nil), off: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sev := DefaultSeverities()
			if tc.off {
				sev[CheckManifestVersion] = Off
			}
			wantManifestFinding(t, Parse([]byte(tc.doc)).LintManifest(tc.m, Options{Severities: sev}), tc.line, tc.want)
		})
	}
}

// wantManifestFinding holds got to one manifest-version finding at error on
// line, its message naming each of want, or to none where want is nil.
func wantManifestFinding(t *testing.T, got []Finding, line int, want []string) {
	t.Helper()
	if want == nil {
		if len(got) != 0 {
			t.Fatalf("findings %v, want none", got)
		}
		return
	}
	if len(got) != 1 || got[0].Check != CheckManifestVersion || got[0].Severity != Error || got[0].Line != line {
		t.Fatalf("findings %v, want one %s at error on line %d", got, CheckManifestVersion, line)
	}
	for _, w := range want {
		if !strings.Contains(got[0].Msg, w) {
			t.Errorf("message %q does not name %s", got[0].Msg, w)
		}
	}
}

// The forecast of a pending release lints with Lint, so a check Lint ran would
// demand of every pull request carrying a fragment a manifest bump that only the
// release commit makes.
func TestLintRaisesNoManifestFinding(t *testing.T) {
	for _, f := range Parse([]byte(manifestDoc)).Lint(Options{}) {
		if f.Check == CheckManifestVersion {
			t.Fatalf("Lint raised %v", f)
		}
	}
}
