package discord

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"trev.zip/llc/discord-embedder/internal/video"

	"github.com/bwmarrin/discordgo"
)

func TestApplicationCommandsEmbedSpeed(t *testing.T) {
	var embedCommand *discordgo.ApplicationCommand
	for _, command := range applicationCommands() {
		if command.Name == "embed" {
			embedCommand = command
			break
		}
	}
	if embedCommand == nil {
		t.Fatal("embed command not found")
	}

	var speedOption *discordgo.ApplicationCommandOption
	for _, option := range embedCommand.Options {
		if option.Name == "speed" {
			speedOption = option
			break
		}
	}
	if speedOption == nil {
		t.Fatal("speed option not found")
	}
	if speedOption.Type != discordgo.ApplicationCommandOptionString {
		t.Errorf("speed option type = %v, want string", speedOption.Type)
	}
	if speedOption.Required {
		t.Error("speed option is required, want optional")
	}

	want := []string{embedSpeedNormal, embedSpeedOneAndHalf, embedSpeedDouble}
	if len(speedOption.Choices) != len(want) {
		t.Fatalf("speed choices = %d, want %d", len(speedOption.Choices), len(want))
	}
	for index, choice := range speedOption.Choices {
		if choice.Name != want[index] {
			t.Errorf("choice %d name = %q, want %q", index, choice.Name, want[index])
		}
		if choice.Value != want[index] {
			t.Errorf("choice %d value = %v, want %q", index, choice.Value, want[index])
		}
	}
}

func TestEmbedSpeedFactor(t *testing.T) {
	tests := []struct {
		name    string
		speed   string
		want    float64
		wantErr bool
	}{
		{name: "omitted", want: 1},
		{name: "normal", speed: embedSpeedNormal, want: 1},
		{name: "one and a half", speed: embedSpeedOneAndHalf, want: 1.5},
		{name: "double", speed: embedSpeedDouble, want: 2},
		{name: "invalid", speed: "x3", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := embedSpeedFactor(tt.speed)
			if tt.wantErr {
				if err == nil {
					t.Fatal("embedSpeedFactor() expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("embedSpeedFactor() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("embedSpeedFactor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplicationCommandsCookies(t *testing.T) {
	var cookiesCommand *discordgo.ApplicationCommand
	for _, command := range applicationCommands() {
		switch command.Name {
		case "cookies":
			cookiesCommand = command
		case "embed":
			for _, option := range command.Options {
				if option.Type == discordgo.ApplicationCommandOptionAttachment {
					t.Errorf("embed option %q is an attachment, cookies belong in /cookies", option.Name)
				}
			}
		}
	}
	if cookiesCommand == nil {
		t.Fatal("cookies command not found")
	}

	want := map[string]discordgo.ApplicationCommandOptionType{
		"file": discordgo.ApplicationCommandOptionAttachment,
	}
	if len(cookiesCommand.Options) != len(want) {
		t.Fatalf("cookies options = %d, want %d", len(cookiesCommand.Options), len(want))
	}
	for _, option := range cookiesCommand.Options {
		wantType, ok := want[option.Name]
		if !ok {
			t.Errorf("unexpected cookies option %q", option.Name)
			continue
		}
		if option.Type != wantType {
			t.Errorf("cookies option %q type = %v, want %v", option.Name, option.Type, wantType)
		}
		if !option.Required {
			t.Errorf("cookies option %q is optional, want required", option.Name)
		}
	}
}

func TestErrMsgLoginRequired(t *testing.T) {
	cause := fmt.Errorf("yt-dlp failed: %w", video.ErrLoginRequired)

	tests := []struct {
		name        string
		usedCookies bool
		want        string
	}{
		{name: "without cookies", want: "tiktok.com requires a login"},
		{name: "with saved cookies", usedCookies: true, want: "saved cookies for tiktok.com did not work"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fmt.Errorf("download: %w", &loginRequiredError{err: cause, site: "tiktok.com", usedCookies: tt.usedCookies})

			got := errMsg(err)
			if !strings.Contains(got, tt.want) {
				t.Errorf("errMsg() = %q, want it to contain %q", got, tt.want)
			}
			if !strings.Contains(got, "`/cookies`") {
				t.Errorf("errMsg() = %q, want it to mention the cookies command", got)
			}
			if !errors.Is(err, video.ErrLoginRequired) {
				t.Error("loginRequiredError does not unwrap to video.ErrLoginRequired")
			}
		})
	}
}
