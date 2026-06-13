{
  description = "Lab report template environment";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
    unsarep.url = "github:UNSAReport/UNSAReport";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      nixpkgs,
      unsarep,
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
        fonts = with pkgs; [ lato ];
      in
      {
        devShells.default = pkgs.mkShell {
          packages =
            with pkgs;
            [
              typst
              typstyle
              tinymist
              charm-freeze
              imagemagick
            ]
            ++ fonts
            ++ [
              unsarep.packages.${system}.default
            ];

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
      }
    );
}
