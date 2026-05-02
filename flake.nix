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

          linuxCudaPackages =
            with pkgs.cudaPackages;
            [
              cudatoolkit
              cudnn
              libcublas
            ];

          linuxPythonPackages = with pkgs.python314Packages; [
            faster-whisper
            ctranslate2
          ];

          linuxLibraryPath = pkgs.lib.makeLibraryPath linuxCudaPackages;
        in
        {
          default = pkgs.mkShell {
            packages =
              with pkgs;
              [
                go
                golangci-lint
                ffmpeg
                python314
              ]
              ++ pkgs.lib.optionals pkgs.stdenv.isLinux linuxPythonPackages
              ++ pkgs.lib.optionals pkgs.stdenv.isLinux linuxCudaPackages
              ++ pkgs.lib.optionals pkgs.stdenv.isLinux [
                pkgs.wl-clipboard
                pkgs.xclip
              ];

            shellHook =
              ''
                export CGO_ENABLED=0
              ''
              + pkgs.lib.optionalString pkgs.stdenv.isLinux ''
                export LD_LIBRARY_PATH="${linuxLibraryPath}:$LD_LIBRARY_PATH"
              '';
          };
        }
      );
    };
}
