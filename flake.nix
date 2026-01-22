{
  description = "discord embedder";

  nixConfig = {
    extra-substituters = [
      "https://cache.trev.zip/nur"
    ];
    extra-trusted-public-keys = [
      "nur:70xGHUW1+1b8FqBchldaunN//pZNVo6FKuPL4U/n844="
    ];
  };

  inputs = {
    systems.url = "github:nix-systems/default";
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
    trev = {
      url = "github:spotdemo4/nur";
      inputs.systems.follows = "systems";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    semgrep-rules = {
      url = "github:semgrep/semgrep-rules";
      flake = false;
    };
  };

  outputs =
    {
      nixpkgs,
      trev,
      semgrep-rules,
      ...
    }:
    trev.libs.mkFlake (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [
            trev.overlays.packages
            trev.overlays.libs
            trev.overlays.images
          ];
        };
        fs = pkgs.lib.fileset;
      in
      rec {
        devShells = {
          default = pkgs.mkShell {
            packages = with pkgs; [
              # go
              go
              gotools
              gopls
              revive

              # util
              air
              bumper

              # nix
              nixfmt

              # actions
              prettier
            ];
            shellHook = pkgs.shellhook.ref;
          };

          bump = pkgs.mkShell {
            packages = with pkgs; [
              bumper
            ];
          };

          release = pkgs.mkShell {
            packages = with pkgs; [
              nix-flake-release
            ];
          };

          update = pkgs.mkShell {
            packages = with pkgs; [
              renovate

              # go mod vendor
              go
            ];
          };

          vulnerable = pkgs.mkShell {
            packages = with pkgs; [
              # go
              go
              govulncheck

              # nix
              flake-checker

              # actions
              octoscan
            ];
          };
        };

        checks = pkgs.lib.mkChecks {
          go = {
            src = packages.default;
            script = ''
              go test ./...
            '';
          };

          revive = {
            src = fs.toSource {
              root = ./.;
              fileset = fs.unions [
                ./revive.toml
                (fs.fileFilter (file: file.hasExt "go") ./.)
              ];
            };
            deps = with pkgs; [
              revive
            ];
            script = ''
              revive ./...
            '';
          };

          opengrep = {
            src = fs.toSource {
              root = ./.;
              fileset = fs.fileFilter (file: file.hasExt "go") ./.;
            };
            deps = with pkgs; [
              opengrep
            ];
            script = ''
              opengrep scan \
                --quiet \
                --error \
                --use-git-ignore \
                --exclude="/vendor/" \
                --config="${semgrep-rules}/go"
            '';
          };

          actions = {
            src = fs.toSource {
              root = ./.github/workflows;
              fileset = ./.github/workflows;
            };
            deps = with pkgs; [
              action-validator
              octoscan
            ];
            script = ''
              action-validator **/*.yaml
              octoscan scan .
            '';
          };

          renovate = {
            src = fs.toSource {
              root = ./.github;
              fileset = ./.github/renovate.json;
            };
            deps = with pkgs; [
              renovate
            ];
            script = ''
              renovate-config-validator renovate.json
            '';
          };

          nix = {
            src = fs.toSource {
              root = ./.;
              fileset = fs.fileFilter (file: file.hasExt "nix") ./.;
            };
            deps = with pkgs; [
              nixfmt-tree
            ];
            script = ''
              treefmt --ci
            '';
          };

          prettier = {
            src = fs.toSource {
              root = ./.;
              fileset = fs.difference (fs.fileFilter (
                file: file.hasExt "yaml" || file.hasExt "json" || file.hasExt "md"
              ) ./.) ./vendor;
            };
            deps = with pkgs; [
              prettier
            ];
            script = ''
              prettier --check .
            '';
          };

          tombi = {
            src = fs.toSource {
              root = ./.;
              fileset = fs.fileFilter (file: file.hasExt "toml") ./.;
            };
            deps = with pkgs; [
              tombi
            ];
            script = ''
              tombi format --offline --check
              tombi lint --offline --error-on-warnings
            '';
          };
        };

        apps = pkgs.lib.mkApps {
          dev.script = "air";
          run.script = "go run .";
        };

        packages = {
          default = pkgs.buildGoModule (finalAttrs: {
            pname = "discord-embedder";
            version = "0.1.13";

            src = fs.toSource {
              root = ./.;
              fileset = fs.unions [
                ./go.mod
                ./go.sum
                ./main.go
                ./vendor
                ./internal
                ./templates
              ];
            };

            goSum = finalAttrs.src + "go.sum";
            vendorHash = null;
            env.CGO_ENABLED = 0;

            meta = {
              description = "Embed videos from various sources into Discord messages";
              mainProgram = "discord-embedder";
              homepage = "https://github.com/spotdemo4/discord-embedder";
              changelog = "https://github.com/spotdemo4/discord-embedder/releases/tag/v${finalAttrs.version}";
              license = pkgs.lib.licenses.mit;
              platforms = pkgs.lib.platforms.all;
            };
          });

          image = pkgs.dockerTools.buildLayeredImage {
            name = packages.default.pname;
            tag = packages.default.version;

            fromImage = pkgs.image.ffmpeg;
            contents = with pkgs; [
              packages.default
              yt-dlp
            ];

            created = "now";
            meta = packages.default.meta;

            config = {
              Cmd = [ "${pkgs.lib.meta.getExe packages.default}" ];
              Labels = {
                "org.opencontainers.image.source" = packages.default.meta.homepage;
                "org.opencontainers.image.version" = packages.default.version;
                "org.opencontainers.image.licenses" = packages.default.meta.license.spdxId;
                "org.opencontainers.image.title" = packages.default.pname;
                "org.opencontainers.image.description" = packages.default.meta.description;
              };
            };
          };
        };

        formatter = pkgs.nixfmt-tree;
      }
    );
}
