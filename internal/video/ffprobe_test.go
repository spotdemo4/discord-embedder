package video

import "testing"

func TestParseAudioProbe(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{name: "aac", output: "aac\n", want: true},
		{name: "opus with carriage return", output: "opus\r\n", want: true},
		{name: "surrounding whitespace", output: " \tflac \n", want: true},
		{name: "empty", output: "", want: false},
		{name: "whitespace only", output: " \r\n\t", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseAudioProbe(tt.output); got != tt.want {
				t.Errorf("parseAudioProbe() = %v, want %v", got, tt.want)
			}
		})
	}
}
