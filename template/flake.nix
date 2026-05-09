{
  description = "Lab report template environment";

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
          fonts = with pkgs; [ lato ];
        in
        pkgs.mkShell {
          packages =
            with pkgs;
            [
              typst
              typstyle
              tinymist
              vhs
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
