{
  description = "Read, check and write a Keep a Changelog 2.0.0 document";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      inherit (nixpkgs) lib;
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = f: lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});

      # This flake's own build of the command, whatever nixpkgs the caller's
      # pkgs come from: the module needs a Go the caller's may not have.
      cliFor = pkgs: self.packages.${pkgs.stdenv.buildPlatform.system}.changelog;
    in
    {
      overlays.default = final: _: { changelog = final.callPackage ./package.nix { }; };

      packages = forAllSystems (pkgs: {
        default = self.packages.${pkgs.stdenv.hostPlatform.system}.changelog;
        changelog = pkgs.callPackage ./package.nix { };
      });

      lib = {
        # version is the version the newest entry of the changelog file names,
        # read by the command at evaluation time (import from derivation). It
        # fails where the changelog names no version.
        version =
          pkgs: changelog:
          lib.fileContents (
            pkgs.runCommand "changelog-version" { nativeBuildInputs = [ (cliFor pkgs) ]; } ''
              changelog version -changelog ${changelog} >$out
            ''
          );

        # check is a derivation that builds when the changelog in src, and the
        # fragments in src's fragments directory, pass every check that needs
        # no git history; the checks that compare against tags or a base
        # branch stay with CI, on a checkout.
        check =
          pkgs:
          {
            src,
            fragments ? ".changelog",
            off ? [ ],
          }:
          pkgs.runCommand "changelog-check" { nativeBuildInputs = [ (cliFor pkgs) ]; } ''
            cd ${src}
            changelog -fragments ${lib.escapeShellArg fragments} \
              -off ${
                lib.escapeShellArg (
                  lib.concatStringsSep "," (
                    [
                      "no-git-tags"
                      "manifest-version"
                    ]
                    ++ off
                  )
                )
              }
            touch $out
          '';
      };

      checks = forAllSystems (pkgs: {
        changelog = self.lib.check pkgs {
          src = lib.fileset.toSource {
            root = ./.;
            fileset = ./CHANGELOG.md;
          };
          fragments = "";
        };
        version =
          let
            got = self.lib.version pkgs ./CHANGELOG.md;
            want = lib.fileContents ./actions/VERSION;
          in
          assert lib.assertMsg (got == want) "lib.version read ${got}, actions/VERSION is ${want}";
          pkgs.emptyFile;
      });

      formatter = forAllSystems (pkgs: pkgs.nixfmt);
    };
}
