{ pkgs ? import <nixpkgs> {}, buildGoModule ? pkgs.buildGoModule
, subPackages ? [ "cmd/upfast" ], pnameSuffix ? ""
}:
buildGoModule {
  pname = "upfast${pnameSuffix}";
  version = "1.3";
  src = ./.;
  vendorHash = null;
  inherit subPackages;
}
