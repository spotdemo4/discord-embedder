package version

import "testing"

func TestFirstLine(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    string
		wantErr bool
	}{
		{
			name:   "yt-dlp version",
			output: "2026.07.04\n",
			want:   "2026.07.04",
		},
		{
			name:   "leading whitespace and CRLF",
			output: " \r\n\t2026.07.04 \r\n",
			want:   "2026.07.04",
		},
		{
			name:    "empty output",
			output:  " \r\n\t\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := firstLine(tt.output)
			if (err != nil) != tt.wantErr {
				t.Fatalf("firstLine() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("firstLine() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseFFmpeg(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    string
		wantErr bool
	}{
		{
			name:   "standard output",
			output: "ffmpeg version 8.1.2 Copyright (c) 2000-2026 the FFmpeg developers\n",
			want:   "8.1.2",
		},
		{
			name:   "revision version",
			output: "ffmpeg version n7.1.1-42-g123abc Copyright (c) 2000-2026 the FFmpeg developers\n",
			want:   "n7.1.1-42-g123abc",
		},
		{
			name:   "additional lines",
			output: "ffmpeg version 8.1.2 Copyright (c) 2000-2026 the FFmpeg developers\nbuilt with gcc 15\n",
			want:   "8.1.2",
		},
		{
			name:   "unexpected output",
			output: "custom ffmpeg build\n",
			want:   "custom ffmpeg build",
		},
		{
			name:    "missing version token",
			output:  "ffmpeg version \n",
			wantErr: true,
		},
		{
			name:    "empty output",
			output:  "\r\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFFmpeg(tt.output)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseFFmpeg() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parseFFmpeg() = %q, want %q", got, tt.want)
			}
		})
	}
}
