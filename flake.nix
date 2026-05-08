{
  description = "mestt development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { self, nixpkgs, ... }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      forAllSystems = nixpkgs.lib.genAttrs systems;
    in
    let
      forAllPkgs =
        system:
        import nixpkgs {
          inherit system;
          config.allowUnfree = true;
        };
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = forAllPkgs system;
          linuxGuiInputs = with pkgs; [
            libGL
            libx11
            libxcursor
            libxext
            libxfixes
            libxi
            libxinerama
            libxrandr
            libxxf86vm
          ];
          mestt = pkgs.buildGoModule {
            pname = "mestt";
            version = "0.1.0-dev";
            src = pkgs.lib.cleanSource ./.;
            vendorHash = "sha256-bTrzpDrzdP3KXEz+JK53C2fdV6Q2n37T58WUpOBHynI=";
            subPackages = [ "cmd/mestt" "cmd/mesttd" ];
          };
          mestt-gui = pkgs.buildGoModule {
            pname = "mestt-gui";
            version = "0.1.0-dev";
            src = pkgs.lib.cleanSource ./.;
            vendorHash = "sha256-bTrzpDrzdP3KXEz+JK53C2fdV6Q2n37T58WUpOBHynI=";
            subPackages = [ "cmd/mestt-gui" ];
            tags = [ "fyne" ];
            nativeBuildInputs = [ pkgs.pkg-config ];
            buildInputs = pkgs.lib.optionals pkgs.stdenv.isLinux linuxGuiInputs;
            env.CGO_ENABLED = 1;
          };
        in
        {
          inherit mestt mestt-gui;
          default = mestt;
        }
      );

      apps = forAllSystems (
        system:
        {
          mestt = {
            type = "app";
            program = "${self.packages.${system}.mestt}/bin/mestt";
          };
          mestt-gui = {
            type = "app";
            program = "${self.packages.${system}.mestt-gui}/bin/mestt-gui";
          };
          default = self.apps.${system}.mestt;
        }
      );

      devShells = forAllSystems (
        system:
        let
          pkgs = forAllPkgs system;
        in
        {
          default = pkgs.mkShell {
            packages =
              with pkgs;
              [
                go
                golangci-lint
                pkg-config
                ffmpeg
                whisper-cpp-vulkan
              ]
              ++ pkgs.lib.optionals pkgs.stdenv.isLinux [
                libGL
                libx11
                libxcursor
                libxext
                libxfixes
                libxi
                libxinerama
                libxrandr
                libxxf86vm
                pkgs.wl-clipboard
                pkgs.xclip
              ];

            shellHook = ''
              export CGO_ENABLED=1
            '';
          };
        }
      );
    };
}
