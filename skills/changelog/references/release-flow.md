# The changelog-based release flow

Four GitHub workflows, copied as a set from [`workflows/`](workflows/) into a repository's
`.github/workflows/`. The changelog decides a release, and the tag is what the release ends on:

```
a push to main    release.yaml reads CHANGELOG.md: the newest entry names a version no tag carries
draft             a draft release v<version> opens, pinned to the pushed commit
check             lint.yaml, test.yaml and build.yaml run in parallel
push              the image build.yaml built goes to the registry as <version>
publish           the draft is published, which creates the tag v<version> on the pinned commit
```

- A release is asked for by merging a pull request whose newest entry is `## [X.Y.Z] - YYYY-MM-DD`.
  Notes live in `CHANGELOG.md`: both `validate` steps set `fragments: ""`.
- `lint.yaml`, `test.yaml` and `build.yaml` also run on every pull request, and on every push to
  `main` through `release.yaml`. Draft, push and publish wait on a release being due.
- A run that fails leaves its draft behind, and the next due push takes it over, pinned to its own
  commit. A fix landing on a later day also re-dates the entry, or `pending-date-mismatch` holds the
  release back.
- Runs are serialized, and GitHub keeps a single waiting run: of several pushes landing during a
  run, only the newest runs. A release whose own run was cancelled that way is cut by that newest
  push, on its commit.
- `build.yaml` is one build for a pull request and for a release: ko writes the image as an OCI
  layout, uploaded as the artifact `image`, and nothing reaches the registry before the push job.
  Called with `version`, it tags the checkout `v<version>` first, so the program reads its version
  from `debug.ReadBuildInfo().Main.Version` and no `-ldflags -X` carries one. The go command stamps
  that version from Go 1.24 on.

## Adapting the copy

- `build.yaml` names the main package as `./cmd/app`, and `release.yaml` pushes to
  `ghcr.io/<owner>/<repo>`. A package that already exists there has to grant the repository write
  access in its settings; one the first push creates inherits the repository's.
- The three checks are written for a Go module with a Taskfile and golangci-lint. Another toolchain
  replaces their steps and keeps what `release.yaml` holds `build.yaml` to: the `version` input and
  the `image` artifact.
