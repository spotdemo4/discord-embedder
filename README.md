# discord-embedder

[![check](https://trev.zip/llc/discord-embedder/actions/workflows/check.yaml/badge.svg?branch=main&logo=forgejo&logoColor=%23bac2de&label=check&labelColor=%23313244)](https://trev.zip/llc/discord-embedder/actions?workflow=check.yaml)
[![vulnerable](https://trev.zip/llc/discord-embedder/actions/workflows/vulnerable.yaml/badge.svg?branch=main&logo=forgejo&logoColor=%23bac2de&label=vulnerable&labelColor=%23313244)](https://trev.zip/llc/discord-embedder/actions?workflow=vulnerable.yaml)
[![nixpkgs](https://nix-shield.trev.zip/?url=https://trev.zip/llc/discord-embedder/raw/branch/main/flake.lock&input=nixpkgs&logoColor=%23bac2de&labelColor=%23313244&color=%235277C3)](https://nixos.org/)
[![go](<https://img.shields.io/badge/dynamic/regex?url=https://trev.zip/llc/discord-embedder/raw/branch/main/go.mod&search=toolchain%20go(.*)&replace=%241&logo=go&logoColor=%23bac2de&label=version&labelColor=%23313244&color=%2300ADD8>)](https://go.dev/doc/devel/release)

A Discord bot / web server that downloads and generates video embeds.

## requirements

- [yt-dlp](https://github.com/yt-dlp/yt-dlp)
- [FFmpeg](https://ffmpeg.org/)

the docker image and nix package include both

## using

### docker

```yaml
# docker-compose.yaml
services:
  discord-embedder:
    container_name: discord-embedder
    image: trev.zip/llc/discord-embedder:latest
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

### nix

```sh
nix run git+https://trev.zip/llc/discord-embedder.git
```

### go

```sh
GOPROXY=https://trev.zip/api/packages/llc/go \
    go install trev.zip/llc/discord-embedder@latest
```

## configuration

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

## example

![example image](https://i.imgur.com/53gDpwW.png)

## contributing

see [CONTRIBUTING.md](CONTRIBUTING.md) for requirements and getting started
