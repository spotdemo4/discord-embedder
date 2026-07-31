{
  description = "discord embedder";

  nixConfig = {
    extra-substituters = [
      "https://nix.trev.zip"
    ];
    extra-trusted-public-keys = [
      "trev:I39N/EsnHkvfmsbx8RUW+ia5dOzojTQNCTzKYij1chU="
    ];
  };

  inputs = {
    systems.url = "github:spotdemo4/systems";
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
    trevpkgs = {
      url = "github:spotdemo4/trevpkgs";
      inputs.systems.follows = "systems";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      self,
      trevpkgs,
      ...
    }:
    trevpkgs.libs.mkFlake (
      system: pkgs:
      let
        ffmpeg-qsv = pkgs.ffmpeg.override {
          ffmpegVariant = "headless";
          withVpl = true;
        };
      in
      {
        # nix develop [#...]
        devShells = {
          default = pkgs.mkShell {
            shellHook = pkgs.shellhook.ref;
            packages = with pkgs; [
              # go
              go
              gopls
              gotools
              go-tools

              # deps
              ffmpeg-qsv
              yt-dlp

              vscode-json-languageserver # json
              yaml-language-server # yaml
              tombi # toml
              oxfmt # format

              # nix
              nixd
              nil
              nixfmt

              # util
              treefmt
              bumper
              fix-hash
            ];
          };

          bump = pkgs.mkShell {
            packages = with pkgs; [
              bumper
            ];
          };

          release = pkgs.mkShell {
            packages = with pkgs; [
              flake-release
            ];
          };

          update = pkgs.mkShell {
            packages = with pkgs; [
              renovate
              go # go mod tidy && go mod vendor
              fix-hash
            ];
          };

          vulnerable = pkgs.mkShell {
            packages = with pkgs; [
              # go
              go
              govulncheck

              flake-checker # nix
              zizmor # actions
            ];
          };
        };

        # nix build [#...]
        packages = {
          default = pkgs.buildGoModule (
            final: with pkgs.lib; {
              pname = "discord-embedder";
              version = "0.3.0";
              ldflags = [ "-X discord-embedder/internal/version.Application=${final.version}" ];

              src = fileset.toSource {
                root = ./.;
                fileset = fileset.unions [
                  ./go.mod
                  ./go.sum
                  ./main.go
                  ./internal
                  ./templates
                  ./vendor
                ];
              };
              goSum = ./go.sum;
              vendorHash = null;

              nativeBuildInputs = with pkgs; [
                makeWrapper
              ];
              nativeCheckInputs = with pkgs; [
                go-tools
              ];
              checkPhase = ''
                export HOME=$(mktemp -d)
                go test ./...
                go vet ./...
                staticcheck ./...
                go fix -diff ./...
              '';
              postFixup = ''
                wrapProgram $out/bin/discord-embedder \
                  --prefix PATH : ${
                    pkgs.lib.makeBinPath [
                      ffmpeg-qsv
                      pkgs.yt-dlp
                    ]
                  } \
                  --prefix LD_LIBRARY_PATH : ${
                    pkgs.lib.makeLibraryPath [
                      pkgs.intel-media-driver
                      pkgs.vpl-gpu-rt
                    ]
                  } \
                  --set LIBVA_DRIVERS_PATH ${pkgs.intel-media-driver}/lib/dri \
                  --set LIBVA_DRIVER_NAME iHD
              '';

              meta = {
                mainProgram = "discord-embedder";
                description = "Embed videos from various sources into Discord messages";
                license = licenses.mit;
                platforms = platforms.unix;
                badPlatforms = [ systems.inspect.platformPatterns.isStatic ];
                homepage = "https://github.com/spotdemo4/discord-embedder";
                changelog = "https://github.com/spotdemo4/discord-embedder/releases/tag/v${final.version}";
              };
            }
          );
        };

        # nix build #images.[...]
        images = {
          default = pkgs.mkImage {
            src = self.packages.${system}.default;
            contents = with pkgs; [
              dockerTools.caCertificates
            ];
          };
        };

        # nix fmt
        formatter = pkgs.treefmt.withConfig {
          configFile = ./treefmt.toml;
          runtimeInputs = with pkgs; [
            go
            nixfmt
            oxfmt
          ];
        };

        # nix flake check
        checks = pkgs.mkChecks {
          go = self.packages.${system}.default.overrideAttrs {
            dontBuild = true;
            installPhase = ''
              touch $out
            '';
            postFixup = "";
          };

          nix = {
            root = ./.;
            filter = file: file.hasExt "nix";
            ignore = ./vendor;
            packages = with pkgs; [
              nixfmt
            ];
            script = ''
              nixfmt --check "$file"
            '';
          };

          actions-gh = {
            root = ./.github/workflows;
            filter = file: file.hasExt "yaml";
            packages = with pkgs; [
              action-validator
              zizmor
            ];
            script = ''
              action-validator "$file"
              zizmor --offline "$file"
            '';
          };

          renovate-gh = {
            root = ./.github;
            files = ./.github/renovate.json;
            packages = with pkgs; [
              renovate
            ];
            script = ''
              renovate-config-validator renovate.json
            '';
          };

          config = {
            root = ./.;
            filter = file: file.hasExt "json" || file.hasExt "yaml" || file.hasExt "toml" || file.hasExt "md";
            ignore = ./vendor;
            packages = with pkgs; [
              oxfmt
            ];
            script = ''
              oxfmt --check
            '';
          };
        };
      }
    );
}
