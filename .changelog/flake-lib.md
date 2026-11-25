### Added

- `version` subcommand: prints the version the newest entry names, reading the changelog alone, with no git history and no checks.
- `lib.version` in `flake.nix`: a project's version as the newest entry of its changelog names it, for a flake to state without a release commit setting it.
- `lib.check` in `flake.nix`: a derivation that builds where a changelog and its fragments pass every check that needs no git history.
