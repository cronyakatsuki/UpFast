{ pkgs ? import <nixpkgs> {} }:
pkgs.mkShell {
  packages = with pkgs; [
    go
    gopls
    gofumpt
    gotools
    golangci-lint
    delve

    # Nix tooling
    alejandra
    nixd
  ];
}
