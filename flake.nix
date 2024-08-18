{
  description = "Flake for Gossip Gloomers, the Fly.io distributed systems challenge";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
  }:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = nixpkgs.legacyPackages.${system};
    in {
      packages.default = pkgs.buildGoModule {
        pname = "gossip-gloomers";
        version = "0.0.1";
        src = ./.;
        vendorHash = "sha256-v1bYSWwfIo6azkNM+DgxK5oKnZuDTe1k2wSCKk0vdos=";
      };

      devShells.default = pkgs.mkShell {
        buildInputs = with pkgs; [
          just
          go
          gotools
          jet
          maelstrom-clj
          self.packages.${system}.default
        ];
      };
    });
}
