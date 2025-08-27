# Discord-Embedder

[![check](https://img.shields.io/github/actions/workflow/status/spotdemo4/discord-embedder/check.yaml?logo=GitHub&logoColor=%23cdd6f4&label=check&labelColor=%2311111b)](https://github.com/spotdemo4/discord-embedder/actions/workflows/check.yaml)
[![flake](https://img.shields.io/github/actions/workflow/status/spotdemo4/discord-embedder/flake.yaml?logo=nixos&logoColor=%2389dceb&label=flake&labelColor=%2311111b)](https://github.com/spotdemo4/discord-embedder/actions/workflows/flake.yaml)
[![vulnerable](https://img.shields.io/github/actions/workflow/status/spotdemo4/discord-embedder/vulnerable.yaml?logo=Go&logoColor=%2389dceb&label=vulnerable&labelColor=%2311111b)](https://github.com/spotdemo4/discord-embedder/actions/workflows/vulnerable.yaml)
[![release](https://img.shields.io/github/v/release/spotdemo4/discord-embedder?logo=github&logoColor=%23cdd6f4&labelColor=%2311111b&color=%23313244)](https://github.com/spotdemo4/discord-embedder/releases/latest)

A Discord bot that embeds a video from a given URL using [yt-dlp](https://github.com/yt-dlp/yt-dlp)

## Installation

Binary executables are available in [releases](https://github.com/spotdemo4/discord-embedder/releases)

Either use environment variables or create the following _config.env_ in ~/.config/discord-embedder

```env
DISCORD_TOKEN=...
DISCORD_APPLICATION_ID=...
```

## Docker Installation

Clone the repository

```
git clone https://github.com/spotdemo4/discord-embedder
```

Create a _.env_ file inside the repository

```env
DISCORD_TOKEN=...
DISCORD_APPLICATION_ID=...
```

Start the container

```
docker-compose up -d
```

## Nix Installation

Add the repository to your flake inputs

```nix
inputs = {
    ...
    discord-embedder.url = "github:spotdemo4/discord-embedder";
};
```

Add the overlay to nixpkgs

```nix
nixpkgs = {
    ...
    overlays = [
        ...
        inputs.discord-embedder.overlays.default
    ];
};
```

Finally, add discord-embedder to your packages

```nix
environment.systemPackages = with pkgs; [
    ...
    discord-embedder
];
```

## Usage/Examples

![example image](https://i.imgur.com/53gDpwW.png)
