package video

import "testing"

func TestSanitizeOriginalURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		want    string
		wantErr bool
	}{
		{
			name:   "YouTube watch keeps only video ID",
			rawURL: "https://user:secret@www.youtube.com/watch?si=tracking&v=abc123&t=30#fragment",
			want:   "https://www.youtube.com/watch?v=abc123",
		},
		{
			name:   "YouTube watch uses first nonempty video ID",
			rawURL: "https://youtube.com/watch?v=&v=abc123&v=other",
			want:   "https://youtube.com/watch?v=abc123",
		},
		{
			name:    "YouTube watch requires video ID",
			rawURL:  "https://youtube.com/watch?feature=share",
			wantErr: true,
		},
		{
			name:   "YouTube short link removes query",
			rawURL: "https://youtu.be/abc123?si=tracking&t=30",
			want:   "https://youtu.be/abc123",
		},
		{
			name:   "YouTube path URL removes query",
			rawURL: "https://m.youtube.com/shorts/abc123?feature=share",
			want:   "https://m.youtube.com/shorts/abc123",
		},
		{
			name:   "TikTok removes query and fragment",
			rawURL: "https://www.tiktok.com/@user/video/123?_r=1&_t=tracking#player",
			want:   "https://www.tiktok.com/@user/video/123",
		},
		{
			name:   "TikTok short link removes query",
			rawURL: "https://vm.tiktok.com/ZMabc/?share_app_id=1233",
			want:   "https://vm.tiktok.com/ZMabc/",
		},
		{
			name:   "X removes query",
			rawURL: "https://x.com/user/status/123?s=20&t=tracking",
			want:   "https://x.com/user/status/123",
		},
		{
			name:   "Twitter removes query",
			rawURL: "https://mobile.twitter.com/user/status/123?ref_src=twsrc%5Etfw",
			want:   "https://mobile.twitter.com/user/status/123",
		},
		{
			name:   "unknown host removes query credentials",
			rawURL: "https://user:secret@example.com/video?token=a%2Fb&token=c#fragment",
			want:   "https://example.com/video",
		},
		{
			name:   "hostname lookalike is not treated as YouTube",
			rawURL: "https://youtube.com.example.org/watch?v=abc123&token=required",
			want:   "https://youtube.com.example.org/watch",
		},
		{
			name:    "unsupported scheme",
			rawURL:  "ftp://example.com/video",
			wantErr: true,
		},
		{
			name:    "relative URL",
			rawURL:  "/video?id=123",
			wantErr: true,
		},
		{
			name:    "missing host",
			rawURL:  "https:///video",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sanitizeOriginalURL(tt.rawURL)
			if (err != nil) != tt.wantErr {
				t.Fatalf("sanitizeOriginalURL() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("sanitizeOriginalURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
