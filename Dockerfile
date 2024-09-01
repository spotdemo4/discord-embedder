FROM golang:1.23

WORKDIR /usr/src/discord-embedder

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o /usr/local/bin/discord-embedder ./...

# Download ffmpeg & curl
RUN apt-get update && apt-get install -y ffmpeg curl

# Download yt-dlp
RUN curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp && chmod +x /usr/local/bin/yt-dlp

CMD ["discord-embedder"]