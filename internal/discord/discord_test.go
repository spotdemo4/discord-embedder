package discord

import (
	"testing"

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
