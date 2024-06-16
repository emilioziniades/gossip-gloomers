{
  description = "A very basic flake";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
  };

  outputs = {
    self,
    nixpkgs,
  }: let
    forAllSystems = fn:
      nixpkgs.lib.genAttrs
      ["x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin"]
      (system: fn system nixpkgs.legacyPackages.${system});
  in {
    packages = forAllSystems (system: pkgs: {
      maelstrom = pkgs.stdenv.mkDerivation rec {
        pname = "maelstrom";
        version = "0.2.3";

        src = pkgs.fetchurl {
          url = "https://github.com/jepsen-io/maelstrom/releases/download/v${version}/maelstrom.tar.bz2";
          hash = "sha256-ISS2qma139Jz9eDxLJvULkqDZeu1vyx9ot4uO0LIVho=";
        };

        installPhase = ''
          runHook preInstall

          mkdir -p $out/bin
          mv maelstrom $out/bin/maelstrom-unwrapped

          mv lib $out/bin

          makeWrapper $out/bin/maelstrom-unwrapped $out/bin/maelstrom --set PATH ${pkgs.lib.makeBinPath buildInputs}

          runHook postInstall
        '';

        nativeBuildInputs = with pkgs; [
          makeWrapper
        ];

        buildInputs = with pkgs; [
          git
          coreutils
          jdk
          gnuplot
          graphviz
        ];
      };

      default = pkgs.buildGoModule {
        pname = "fly-io-dist-sys";
        version = "0.0.1";
        src = ./.;
        vendorHash = "sha256-ISS2qma139Jz9eDxLJvULkqDZeu1vyx9ot4uO0LIVho";
      };
    });
  };
}
