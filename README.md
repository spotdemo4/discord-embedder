
# Discord-Embedder

A Discord bot that embeds a video from a given URL using [yt-dlp](https://github.com/yt-dlp/yt-dlp)


## Installation

Binary executables are available in [releases](https://github.com/spotdemo4/discord-embedder/releases)

Either use environment variables or create the following *config.env* in ~/.config/discord-embedder
```env
DISCORD_TOKEN=...
DISCORD_APPLICATION_ID=...
```
## Docker Installation

Clone the repository
```
git clone https://github.com/spotdemo4/discord-embedder
```

Create a *.env* file inside the repository
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