### Added

- `manifest-version` check: the `flake.nix` beside the changelog states the version the newest entry names, read with `nix eval .#<attr>.version`. Where `nix` is not on `PATH` or the flake has no such package, it is a finding rather than a pass, so a repository with a `flake.nix` puts `nix` on the runner or switches the check off.
- `flake-attr` input and `-flake-attr` flag: the flake package `manifest-version` reads, `default` unless set.
- `Manifest` and `Changelog.LintManifest` in `pkg/changelog`.
- `flake.nix`: the command as the flake's `changelog` package, and `overlays.default` adding it to nixpkgs.
