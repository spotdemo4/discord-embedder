# contributing

## requirements

- [nix](https://nixos.org/)

or, without nix:

- [go](https://go.dev/)
- [yt-dlp](https://github.com/yt-dlp/yt-dlp)
- [FFmpeg](https://ffmpeg.org/)

## getting started

```sh
nix develop
```

with [direnv](https://direnv.net/):

```sh
ln -s .envrc.project .envrc
direnv allow
```

### run

```sh
nix run
```

with [go](https://go.dev/):

```sh
go run .
```

### format

```sh
nix fmt
```

with [golangci-lint](https://golangci-lint.run/):

```sh
golangci-lint fmt
```

### check

```sh
nix flake check
```

with [go](https://go.dev/) and [golangci-lint](https://golangci-lint.run/):

```sh
go test ./...
golangci-lint run ./...
```

### build

```sh
nix build
```

with [go](https://go.dev/):

```sh
go build ./...
```

### dependencies

after changing `go.mod`, tidy it and update `gomod2nix.toml`:

```sh
nix run .#update-deps
```

with [go](https://go.dev/) and [gomod2nix](https://github.com/nix-community/gomod2nix):

```sh
go mod tidy
gomod2nix generate --with-deps
```

### release

with [bumper](https://trev.zip/llc/bumper):

```sh
bumper
```

releases are automatically created for [significant](https://www.conventionalcommits.org/en/v1.0.0/#summary) changes
