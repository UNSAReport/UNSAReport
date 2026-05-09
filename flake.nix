{
  description = "Lab Report CLI";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
          config.allowUnfree = true;
        };
        ldPath =
          with pkgs;
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
          version = "1.0.0";
          src = ./.;
          subPackages = [ "cmd/lab-report" ];

          vendorHash = "sha256-xU5rQY64h11wbDn8BuckO32KHpaihCaxkBQnfM0H2tQ=";

          nativeBuildInputs = [ pkgs.pkg-config ];
          buildInputs = [ pkgs.fontconfig ];

          ldflags = [
            "-s"
            "-w"
          ];

          meta = with pkgs.lib; {
            description = "A CLI tool for generating lab reports from markdown files.";
            homepage = "https://github.com/christianmz565/lab-report";
            license = licenses.mit;
            platforms = platforms.unix ++ platforms.darwin ++ platforms.windows;
          };
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

            fontconfig
            pkg-config
          ];

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
