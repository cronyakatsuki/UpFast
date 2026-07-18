{
  description = "UpFast — minimal file upload and sharing tool";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        packages.default = pkgs.callPackage ./default.nix {};

        devShells.default = pkgs.callPackage ./shell.nix {};

        checks.go-lint = pkgs.stdenvNoCC.mkDerivation {
          name = "go-lint";
          dontBuild = true;
          src = ./.;
          doCheck = true;
          nativeBuildInputs = with pkgs; [
            golangci-lint
            go
            writableTmpDirAsHomeHook
          ];
          # Disable CGO so net/http doesn't need a C compiler in the sandbox
          CGO_ENABLED = 0;
          checkPhase = "golangci-lint run";
          installPhase = "mkdir $out";
        };
      }
    );
}
