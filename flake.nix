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
    in
    {
      overlays.default = final: _: { changelog = final.callPackage ./package.nix { }; };

      packages = forAllSystems (pkgs: {
        default = self.packages.${pkgs.stdenv.hostPlatform.system}.changelog;
        changelog = pkgs.callPackage ./package.nix { };
      });

      formatter = forAllSystems (pkgs: pkgs.nixfmt);
    };
}
