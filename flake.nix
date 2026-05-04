{
  description = "mestt development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { nixpkgs, ... }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      forAllSystems = nixpkgs.lib.genAttrs systems;
    in
    {
      devShells = forAllSystems (
        system:
        let
          pkgs = import nixpkgs {
            inherit system;
            config.allowUnfree = true;
          };

        in
        {
          default = pkgs.mkShell {
            packages =
              with pkgs;
              [
                go
                golangci-lint
                ffmpeg
                whisper-cpp-vulkan
              ]
              ++ pkgs.lib.optionals pkgs.stdenv.isLinux [
                pkgs.wl-clipboard
                pkgs.xclip
              ];

            shellHook = ''
              export CGO_ENABLED=0
            '';
          };
        }
      );
    };
}
