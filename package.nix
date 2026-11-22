{
  lib,
  buildGoModule,
}:

buildGoModule {
  pname = "changelog";
  # The release the actions download, which TestVersionMatchesTheChangelog
  # holds to the newest CHANGELOG.md entry.
  version = lib.fileContents ./actions/VERSION;

  src = lib.fileset.toSource {
    root = ./.;
    fileset = lib.fileset.unions [
      ./go.mod
      ./go.sum
      ./cmd/changelog
      ./internal
      ./pkg
    ];
  };

  vendorHash = "sha256-MHbXUluymzqQf68FMbOlMWUWdS8lNAZ5x51+qfcseXA=";

  # The go command writes telemetry counters under $HOME, and nix refuses to
  # start an unsandboxed build, as CI's are, once /homeless-shelter exists.
  preBuild = "export HOME=$TMPDIR";
  overrideModAttrs = _: { preBuild = "export HOME=$TMPDIR"; };

  subPackages = [ "cmd/changelog" ];
  # The suite reads the whole repository and its git history, which this
  # source leaves out; CI runs it on the checkout.
  doCheck = false;

  env.CGO_ENABLED = 0;
  ldflags = [ "-s" ];

  meta = {
    description = "Read, check and write a Keep a Changelog 2.0.0 document";
    homepage = "https://sarak.sh/mikluko/changelog";
    license = lib.licenses.mit;
    mainProgram = "changelog";
  };
}
