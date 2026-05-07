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
        in
        pkgs.mkShell {
          packages = with pkgs; [
            nodejs
            bun

            cargo
            rustc
            rustfmt
            clippy
            rust-analyzer
            zig
          ];
          nativeBuildInputs = [ pkgs.pkg-config ];
          env = {
            RUST_SRC_PATH = "${pkgs.rust.packages.stable.rustPlatform.rustLibSrc}";
          };
          buildInputs = [ pkgs.bashInteractive ];
          shellHook = "";
        };
    };
}
