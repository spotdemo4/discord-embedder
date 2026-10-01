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

with [gofmt](https://pkg.go.dev/cmd/gofmt):

```sh
gofmt -w .
```

### check

```sh
nix flake check
```

with [go](https://go.dev/) and [staticcheck](https://staticcheck.dev/):

```sh
go test ./...
go vet ./...
staticcheck ./...
go fix -diff ./...
```

### build

```sh
nix build
```

with [go](https://go.dev/):

```sh
go build ./...
```

### release

with [bumper](https://trev.zip/llc/bumper):

```sh
bumper
```

releases are automatically created for [significant](https://www.conventionalcommits.org/en/v1.0.0/#summary) changes
