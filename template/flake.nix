{
  description = "A flake";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
  };

  outputs =
    { nixpkgs, ... }:
    let
      system = "x86_64-linux";
    in
    {
      devShells."${system}".default =
        let
          pkgs = import nixpkgs {
            inherit system;
            config.allowUnfree = true;
          };
          fonts = with pkgs; [
            lato
          ];
        in
        pkgs.mkShell {
          LD_LIBRARY_PATH =
            with pkgs;
            lib.makeLibraryPath [
              stdenv.cc.cc
              zlib
              glib
              libxcb
              libglvnd
            ];
          packages =
            with pkgs;
            [
              # docling is marked as broken and like who can be bothered to setup uv2nix
              # just uv run --with docling docling is good enough
              uv
              typst
              tinymist
              charm-freeze
              imagemagick
              zip
              nodejs
            ]
            ++ fonts;
          buildInputs = [ pkgs.bashInteractive ];
          shellHook = ''
            unset SOURCE_DATE_EPOCH
          '';

          env = {
            FONTCONFIG_FILE = pkgs.makeFontsConf {
              fontDirectories = fonts;
            };
          };
        };
    };
}
