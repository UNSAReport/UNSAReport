{
  description = "Lab report template environment";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
    lab-report.url = "github:christianmz565/lab-report";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      nixpkgs,
      lab-report,
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
              lab-report.packages.${system}.default
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
