# discord-embedder

[![check](https://img.shields.io/github/actions/workflow/status/spotdemo4/discord-embedder/check.yaml?branch=main&logo=github&logoColor=%23bac2de&label=check&labelColor=%23313244)](https://github.com/spotdemo4/discord-embedder/actions/workflows/check.yaml/)
[![vulnerable](https://img.shields.io/github/actions/workflow/status/spotdemo4/discord-embedder/vulnerable.yaml?branch=main&logo=github&logoColor=%23bac2de&label=vulnerable&labelColor=%23313244)](https://github.com/spotdemo4/discord-embedder/actions/workflows/vulnerable.yaml)
[![nix](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fraw.githubusercontent.com%2Fspotdemo4%2Fdiscord-embedder%2Frefs%2Fheads%2Fmain%2Fflake.lock&query=%24.nodes.nixpkgs.original.ref&logo=nixos&logoColor=%23bac2de&label=channel&labelColor=%23313244&color=%234d6fb7)](https://nixos.org/)
[![go](https://img.shields.io/github/go-mod/go-version/spotdemo4/discord-embedder?logo=go&logoColor=%23bac2de&label=version&labelColor=%23313244&color=%2300ADD8)](https://go.dev/doc/devel/release)
[![flakehub](https://img.shields.io/endpoint?url=https://flakehub.com/f/spotdemo4/discord-embedder/badge&labelColor=%23313244)](https://flakehub.com/flake/spotdemo4/discord-embedder)

A Discord bot / web server that downloads and generates video embeds.

## Requirements

- [yt-dlp](https://github.com/yt-dlp/yt-dlp)
- [FFmpeg](https://ffmpeg.org/)

## Installation

Container images are published to [GitHub Container Registry](https://github.com/spotdemo4/discord-embedder/pkgs/container/discord-embedder), and the repository exposes a Nix package.

### Docker

```yaml
# docker-compose.yaml
services:
  discord-embedder:
    container_name: discord-embedder
    image: ghcr.io/spotdemo4/discord-embedder:0.3.0
    environment:
      - DISCORD_TOKEN=...
      - DISCORD_APPLICATION_ID=...
      - HOST=https://embed.example.com
      - FILES_DIR=/files
    volumes:
      - ./files:/files
    ports:
      - 8080:8080
    restart: unless-stopped
```

### Nix

Add the repository to your flake inputs:

```nix
inputs = {
  # ...
  discord-embedder = {
    url = "github:spotdemo4/discord-embedder";
    inputs.nixpkgs.follows = "nixpkgs";
  };
};
```

Then add discord-embedder to your packages:

```nix
environment.systemPackages = with pkgs; [
  # ...
  discord-embedder.packages."${system}".default
];
```

## Configuration

All configuration is done through environment variables or a `.env` file.

```dotenv
DISCORD_TOKEN=replaceme # Discord bot token
DISCORD_APPLICATION_ID=replaceme # Discord application ID
DISCORD_CHANNEL_IDS=000000000000,000000000000 # Comma-separated list of Discord channel IDs that automatically embed URLs
FILES_DIR=/tmp/files # Path to store downloaded files
TMP_DIR=/tmp/temp # Path to store temporary files
HOST=https://embed.example.com # URL where the web server will be reachable
PORT=8080 # Port for the web server to listen on (default: 8080)
QUICKSYNC=false # Toggle Intel QSV for transcoding downloaded videos (default: false)
LOG_LEVEL=info # Log level: debug, info, warn, or error (default: info)
```

You can also set login information to bypass content filters:

```dotenv
REDDIT_USERNAME=...
REDDIT_PASSWORD=...

TIKTOK_USERNAME=...
TIKTOK_PASSWORD=...

INSTAGRAM_USERNAME=...
INSTAGRAM_PASSWORD=...

X_USERNAME=...
X_PASSWORD=...
```

## Usage/Examples

![example image](https://i.imgur.com/53gDpwW.png)
