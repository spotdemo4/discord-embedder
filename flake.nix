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
    trev = {
      url = "github:spotdemo4/nur";
      inputs.systems.follows = "systems";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      self,
      trev,
      ...
    }:
    trev.libs.mkFlake (
      system: pkgs:
      let
        ffmpeg-qsv = pkgs.ffmpeg.override {
          ffmpegVariant = "headless";
          withVpl = true;
        };
      in
      {
        devShells = {
          default = pkgs.mkShell {
            shellHook = pkgs.shellhook.ref;
            packages = with pkgs; [
              # go
              go
              gotools
              gopls

              # deps
              ffmpeg-qsv
              yt-dlp

              # lint
              revive

              # format
              nixfmt
              tombi
              prettier

              # util
              air
              bumper
              flake-release
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
              go # go mod vendor
            ];
          };

          vulnerable = pkgs.mkShell {
            packages = with pkgs; [
              # go
              go
              govulncheck

              flake-checker # flake
              octoscan # actions
            ];
          };
        };

        apps = pkgs.mkApps {
          dev = "air";
          run = "go run .";
          vendor = "go mod tidy && go mod vendor";
        };

        checks =
          with pkgs.lib;
          pkgs.mkChecks {
            go = {
              src = self.packages.${system}.default;
              script = ''
                go test ./...
              '';
            };

            revive = {
              root = ./.;
              fileset = fileset.unions [
                ./revive.toml
                ./main.go
                ./internal
              ];
              packages = with pkgs; [
                revive
              ];
              script = ''
                revive ./...
              '';
            };

            actions = {
              root = ./.github/workflows;
              packages = with pkgs; [
                action-validator
                octoscan
              ];
              forEach = ''
                action-validator "$file"
                octoscan scan "$file"
              '';
            };

            renovate = {
              root = ./.github;
              fileset = ./.github/renovate.json;
              packages = with pkgs; [
                renovate
              ];
              script = ''
                renovate-config-validator renovate.json
              '';
            };

            nix = {
              root = ./.;
              ignore = ./vendor;
              filter = file: file.hasExt "nix";
              packages = with pkgs; [
                nixfmt
              ];
              forEach = ''
                nixfmt --check "$file"
              '';
            };

            prettier = {
              root = ./.;
              ignore = ./vendor;
              filter = file: file.hasExt "yaml" || file.hasExt "json" || file.hasExt "md";
              packages = with pkgs; [
                prettier
              ];
              forEach = ''
                prettier --check "$file"
              '';
            };

            tombi = {
              root = ./.;
              ignore = ./vendor;
              filter = file: file.hasExt "toml";
              packages = with pkgs; [
                tombi
              ];
              forEach = ''
                tombi format --offline --check "$file"
                tombi lint --offline --error-on-warnings "$file"
              '';
            };
          };

        packages.default = pkgs.buildGoModule (
          final: with pkgs.lib; {
            pname = "discord-embedder";
            version = "0.1.19";

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

            nativeBuildInputs = with pkgs; [ makeWrapper ];
            postFixup = ''
              wrapProgram $out/bin/discord-embedder \
                --prefix PATH : ${
                  pkgs.lib.makeBinPath [
                    ffmpeg-qsv
                    pkgs.yt-dlp
                  ]
                }
            '';

            meta = {
              mainProgram = "discord-embedder";
              description = "Embed videos from various sources into Discord messages";
              license = licenses.mit;
              platforms = platforms.linux;
              badPlatforms = [ systems.inspect.platformPatterns.isStatic ];
              homepage = "https://github.com/spotdemo4/discord-embedder";
              changelog = "https://github.com/spotdemo4/discord-embedder/releases/tag/v${final.version}";
            };
          }
        );

        images.default = pkgs.mkImage {
          src = self.packages.${system}.default;
          contents = with pkgs; [
            dockerTools.caCertificates
          ];
        };

        formatter = pkgs.nixfmt-tree;
        schemas = trev.schemas;
      }
    );
}
