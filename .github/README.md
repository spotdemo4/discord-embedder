# Discord-Embedder

[![check](https://img.shields.io/github/actions/workflow/status/spotdemo4/discord-embedder/check.yaml?logo=GitHub&logoColor=%23cdd6f4&label=check&labelColor=%2311111b)](https://github.com/spotdemo4/discord-embedder/actions/workflows/check.yaml)
[![flake](https://img.shields.io/github/actions/workflow/status/spotdemo4/discord-embedder/flake.yaml?logo=nixos&logoColor=%2389dceb&label=flake&labelColor=%2311111b)](https://github.com/spotdemo4/discord-embedder/actions/workflows/flake.yaml)
[![vulnerable](https://img.shields.io/github/actions/workflow/status/spotdemo4/discord-embedder/vulnerable.yaml?logo=Go&logoColor=%2389dceb&label=vulnerable&labelColor=%2311111b)](https://github.com/spotdemo4/discord-embedder/actions/workflows/vulnerable.yaml)
[![release](https://img.shields.io/github/v/release/spotdemo4/discord-embedder?logo=github&logoColor=%23cdd6f4&labelColor=%2311111b&color=%23313244)](https://github.com/spotdemo4/discord-embedder/releases/latest)

A Discord bot / web server that downloads and embeds videos

## Installation

Binary executables are available in [releases](https://github.com/spotdemo4/discord-embedder/releases)

### Docker Installation

```yaml
# docker-compose.yaml
services:
  discord-embedder:
    container_name: discord-embedder
    image: ghcr.io/spotdemo4/discord-embedder:0.1.2 # replace version with latest
    environment:
      - DISCORD_TOKEN=...
      - DISCORD_APPLICATION_ID=...
      - FILES_DIR=/files
      ...
    volumes:
      - ./files:/files
    ports:
      - 8080:8080
    restart: unless-stopped
```

### Nix Installation

Add the repository to your flake inputs

```nix
inputs = {
    ...
    discord-embedder = {
      url = "github:spotdemo4/discord-embedder";
      inputs.nixpkgs.follows = "nixpkgs";
    };
};
```

And then add discord-embedder to your packages

```nix
environment.systemPackages = with pkgs; [
    ...
    discord-embedder.packages."${system}".default
];
```

## Configuration

All configuration is done through environment variables or a .env file:

```shell
DISCORD_TOKEN=replaceme # Discord bot token
DISCORD_APPLICATION_ID=replaceme # Discord application ID
DISCORD_CHANNEL_IDS=000000000000,000000000000 # Comma seperated list of Discord channel IDs that will automatically embed URLs
FILES_DIR=/tmp/files # Path to store downloaded files
TMP_DIR=/tmp/temp # Path to store temporary files
HOST=https://embed.example.com # URL where the web server will be reachable
PORT=8080 # Port for the web server to listen on (default: 8080)
QUICKSYNC=false # Toggle Intel QSV for compressing downloads
```

You can also set login information to bypass content filters:

```shell
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
