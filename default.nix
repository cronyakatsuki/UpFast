{ pkgs ? import <nixpkgs> {}, buildGoModule ? pkgs.buildGoModule }:
buildGoModule {
  pname = "upfast";
  version = "1.3";
  src = ./.;
  # No external dependencies — null means no vendor hash needed
  vendorHash = null;
  subPackages = [ "cmd/upfast" ];
}
