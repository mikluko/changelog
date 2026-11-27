# The Forgejo release flow

Two Forgejo workflows and the action they log in to the registry with, copied as a tree from
[`forgejo/`](forgejo/) into a repository's `.forgejo/`, for a repository built by a Nix flake on a
runner labelled `nixos`. A release is asked for by hand, and the tag is what it ends on:

```
dispatch          Release on main, prerelease ticked or not
prepare           release: fold the fragments, write VERSION, commit and push to main
                  pre-release: compute X-pre.N from the tags, change nothing
ci                ci.yaml on that commit: nix flake check, nix build .#release
publish           VERSION written and staged, .#release built, the images pushed as <version>
tag               POST /releases at that commit, which creates the tag v<version>
```

- Notes are fragments in `.changelog/`. A pull request adds one; `ci.yaml` warns on one that
  carries none. Nothing changes `CHANGELOG.md` between releases.
- The version reaches the build through `VERSION`, which the flake reads. The release commit
  writes it; a pre-release overwrites it in the checkout and stages it, so the flake sees it with
  no commit.
- A pre-release, `vX-pre.N`, is numbered one above the `vX-pre.N` tags. It moves no file on
  `main`, so the release that follows folds the same fragments as `X`.
- A run that fails after pushing the release commit leaves a dated entry no tag names, and
  `validate` reports it due: the next dispatch builds and tags that commit without folding. A
  pre-release is refused until it is.
- A push to `main` refused because `main` moved publishes nothing; dispatch again.

## Adapting the copy

[flakybin's `flake.nix`](https://sarak.sh/mikluko/flakybin/src/branch/main/flake.nix) is a flake
that provides all of it: `release` runs each `dockerTools.streamLayeredImage` script, copies the
archive into the layout with `skopeo copy --dest-compress-format gzip`, and adds the platforms to
`index.json` with `jq`.

- The flake provides what the workflows name: `packages.<system>.default` whose `version` is read
  from `VERSION`; `packages.<system>.release`, a directory holding `image/`, an OCI layout with one
  manifest per platform, each carrying its `platform`, which `crane push --index` sends as one
  multi-arch index; the checks `nix flake check` runs; and a `devShell` carrying `crane` and `jq`.
  With `version` read from `VERSION`, `manifest-version` holds the file to the changelog.
- The image goes to the forge's own registry under the repository's name. `login-registry` trades
  the job's OIDC token for an Authorized Integration of the owner's, scoped `write:package`, whose
  audience is the variable `REGISTRY_AUDIENCE`, and leaves one `config.json` in `RUNNER_TEMP` that
  `DOCKER_CONFIG` and `HELM_REGISTRY_CONFIG` name for the job's later steps.
- The release commit is pushed with the workflow's token: a branch protection rule on `main` lets
  that token push.
- A chart or another artifact is one more file in `release`, built by the flake, and one more
  push in `publish`, so the tag still names only what exists. A chart with no dependencies
  packages offline: `helm package` in a `runCommand`, with `--version` from `VERSION`.
