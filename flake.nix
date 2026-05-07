{
  description = "Lab Report CLI";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    { self, nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
          config.allowUnfree = true;
        };
        ldPath = with pkgs;
          lib.makeLibraryPath [
            stdenv.cc.cc
            zlib
            glib
            libxcb
            libglvnd
          ];
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "lab-report";
          version = "0.0.0";
          src = ./.;
          subPackages = [ "cmd/lab-report" ];

          # Update this hash by running: nix build
          vendorHash = "sha256-56GDdOSzIO3f2ikTOf8dauJ2fx7h+CTTeQNruZm1lr0=";

          ldflags = [
            "-s"
            "-w"
            "-X github.com/christianmz565/lab-report/internal/cli.Version=0.0.0"
          ];
        };

        apps.default = flake-utils.lib.mkApp {
          drv = self.packages.${system}.default;
        };

        devShells.default = pkgs.mkShell {
          LD_LIBRARY_PATH = ldPath;

          packages = with pkgs; [
            go
            gopls
            gotools
            golangci-lint

            typst
            tinymist
            charm-freeze
            imagemagick
          ];

          nativeBuildInputs = [ pkgs.pkg-config ];
          buildInputs = [ pkgs.bashInteractive ];

          env = {
            GOPROXY = "https://proxy.golang.org,direct";
            GOSUMDB = "sum.golang.org";
          };

          shellHook = ''
            export GOPATH="$PWD/.go"
            export GOBIN="$GOPATH/bin"

            mkdir -p "$GOBIN"
            export PATH="$GOBIN:$PATH"
          '';
        };
      }
    );
}
