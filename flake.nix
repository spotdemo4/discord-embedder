{
  description = "discord embedder";

  inputs = {
    systems.url = "systems";
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
    utils = {
      url = "github:numtide/flake-utils";
      inputs.systems.follows = "systems";
    };
    nur = {
      url = "github:nix-community/NUR";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    semgrep-rules = {
      url = "github:semgrep/semgrep-rules";
      flake = false;
    };
  };

  outputs = {
    nixpkgs,
    utils,
    nur,
    semgrep-rules,
    ...
  }:
    utils.lib.eachDefaultSystem (system: let
      pkgs = import nixpkgs {
        inherit system;
        overlays = [nur.overlays.default];
      };
    in rec {
      devShells.default = pkgs.mkShell {
        packages = with pkgs; [
          git
          pkgs.nur.repos.trev.bumper

          # Go
          go
          gotools
          gopls
          air
          golangci-lint
          govulncheck

          # Nix
          alejandra
          flake-checker

          # Actions
          action-validator
          prettier
          skopeo
          pkgs.nur.repos.trev.renovate
        ];
        shellHook = pkgs.nur.repos.trev.shellhook.ref;
      };

      checks =
        pkgs.nur.repos.trev.lib.mkChecks {
          lint = {
            src = ./.;
            deps = with pkgs; [
              go
              golangci-lint
              alejandra
              prettier
              action-validator
              pkgs.nur.repos.trev.renovate
            ];
            script = ''
              golangci-lint run ./...
              alejandra -c .
              prettier --check .
              action-validator .github/workflows/*
              renovate-config-validator
              renovate-config-validator .github/renovate-global.json
            '';
          };

          scan = {
            src = ./.;
            deps = [
              pkgs.nur.repos.trev.opengrep
            ];
            script = ''
              opengrep scan --quiet --error --config="${semgrep-rules}/go"
            '';
          };
        }
        // {
          build = packages.default.overrideAttrs {
            doCheck = true;
          };
          shell = devShells.default;
        };

      packages = with pkgs.nur.repos.trev.lib; rec {
        default = pkgs.buildGoModule (finalAttrs: {
          pname = "discord-embedder";
          version = "0.1.5";
          src = ./.;
          goSum = ./go.sum;
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

        image = pkgs.dockerTools.streamLayeredImage {
          name = "${default.pname}";
          tag = "${default.version}";
          created = "now";
          contents = with pkgs; [
            default

            # deps
            dockerTools.caCertificates
            yt-dlp
            ffmpeg
          ];
          config = {
            Cmd = [
              "${pkgs.lib.meta.getExe default}"
            ];
          };
        };

        linux-amd64 = go.moduleToPlatform default "linux" "amd64";
        linux-arm64 = go.moduleToPlatform default "linux" "arm64";
        linux-arm = go.moduleToPlatform default "linux" "arm";
        darwin-arm64 = go.moduleToPlatform default "darwin" "arm64";
        windows-amd64 = go.moduleToPlatform default "windows" "amd64";
      };

      formatter = pkgs.alejandra;
    });
}
