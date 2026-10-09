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
              jq
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
              go # go mod tidy
              fix-hash # vendorHash
            ];
          };

          vulnerable = pkgs.mkShell {
            packages = with pkgs; [
              go
              govulncheck # go
              flake-checker # nix
              zizmor # actions
            ];
          };
        };

        # nix build [#...]
        packages = {
          default = pkgs.mkGoModule (
            final: with pkgs.lib; {
              pname = "discord-embedder";
              version = "0.7.1";
              ldflags = [ "-X trev.zip/llc/discord-embedder/internal/version.Application=${final.version}" ];

              src = fileset.toSource {
                root = ./.;
                fileset = fileset.unions [
                  ./go.mod
                  ./go.sum
                  (fileset.fileFilter (file: file.hasExt "go") ./.)
                  ./templates
                ];
              };
              goSum = ./go.sum;
              vendorHash = "sha256-WV1r2eh+33Os0nF46Y2jyrFCBiRiUhFe7/EgFK/uZ58=";

              nativeBuildInputs = with pkgs; [
                makeWrapper
              ];

              doCheck = true;
              checkPhase = ''
                runHook preCheck
                export HOME=$(mktemp -d)
                go test ./...
                runHook postCheck
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
                homepage = "https://trev.zip/llc/discord-embedder";
                changelog = "https://trev.zip/llc/discord-embedder/releases";
                downloadPage = "https://trev.zip/llc/discord-embedder/releases/tag/v${final.version}";
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
          inherit (self.packages.${system}) default;

          go = {
            src = self.packages.${system}.default;
            packages = with pkgs; [ go-tools ];
            script = ''
              go vet ./...
              staticcheck ./...
              go fix -diff ./...
            '';
          };

          nix = {
            root = ./.;
            filter = file: file.hasExt "nix";
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

          actions-fj = {
            root = ./.forgejo/workflows;
            filter = file: file.hasExt "yaml";
            packages = with pkgs; [
              forgejo-runner
              zizmor
            ];
            script = ''
              forgejo-runner validate --workflow --path "$file"
              zizmor --offline "$file"
            '';
          };

          renovate-fj = {
            root = ./.forgejo;
            files = ./.forgejo/renovate.json;
            packages = with pkgs; [
              renovate
            ];
            script = ''
              renovate-config-validator renovate.json
            '';
          };

          scripts = {
            root = ./scripts;
            packages = with pkgs; [
              shellcheck
            ];
            script = ''
              shellcheck "$file"
            '';
          };

          config = {
            root = ./.;
            filter = file: file.hasExt "json" || file.hasExt "yaml" || file.hasExt "toml" || file.hasExt "md";
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
