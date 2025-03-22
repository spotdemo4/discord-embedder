FROM debian:trixie-slim

WORKDIR /app

# Allow non-free
RUN sed -i -e's/ main/ main contrib non-free/g' /etc/apt/sources.list.d/debian.sources

# Download intel media packages
RUN apt-get update && apt-get install -y intel-media-va-driver-non-free libmfx-gen1.2 libvpl2 libvpl-tools libva-glx2 va-driver-all vainfo

# Download golang, ffmpeg, curl, python
RUN apt-get install -y golang ffmpeg curl python3

# Download yt-dlp
RUN curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux -o /usr/local/bin/yt-dlp && chmod +x /usr/local/bin/yt-dlp

# Install deps
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Build
COPY . .
RUN go build -v -o /usr/local/bin/discord-embedder .

CMD ["discord-embedder"]