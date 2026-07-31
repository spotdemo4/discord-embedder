package discord

import (
	"context"
	"discord-embedder/internal/config"
	"discord-embedder/internal/logger"
	"errors"
	"os/exec"

	"github.com/bwmarrin/discordgo"
)

const (
	embedSpeedNormal = "x1"
	embedSpeedOneAndHalf = "x1.5"
	embedSpeedDouble = "x2"
)

func New(ctx context.Context) error {
	log := logger.FromContext(ctx)
	cfg := config.FromContext(ctx)

	// Create a new Discord session using the provided bot token
	session, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		log.ErrorContext(ctx, "could not create discord session", "error", err)
		return err
	}

	commands := applicationCommands()

	// Add discord handlers
	session.AddHandler(onInteraction(ctx))
	session.AddHandler(onMessage(ctx))
	session.AddHandler(onReady(ctx, commands))
	session.AddHandler(onJoin(ctx, commands))

	// Add intents
	session.Identify.Intents = discordgo.IntentsDirectMessages | discordgo.IntentsGuildMessages

	// Start the websocket connection to Discord
	err = session.Open()
	if err != nil {
		log.ErrorContext(ctx, "could not open session", "error", err)
		return err
	}

	// Wait for context cancellation
	log.InfoContext(ctx, "discord session started")
	<-ctx.Done()

	// Close the session
	if err = session.Close(); err != nil {
		log.ErrorContext(ctx, "could not close session", "error", err)
		return err
	}

	log.InfoContext(ctx, "discord session closed")
	return nil
}

func applicationCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "embed",
			Description: "Embed a video from a URL",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "url",
					Description: "URL of the video to embed",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
				{
					Name:        "start",
					Description: "Start time of the video in 00:00 format (e.g. 01:30)",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    false,
				},
				{
					Name:        "end",
					Description: "End time of the video in 00:00 format (e.g. 02:00)",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    false,
				},
				{
					Name:        "speed",
					Description: "Playback speed of the embedded video",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    false,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: embedSpeedNormal, Value: embedSpeedNormal},
						{Name: embedSpeedOneAndHalf, Value: embedSpeedOneAndHalf},
						{Name: embedSpeedDouble, Value: embedSpeedDouble},
					},
				},
				{
					Name:        "spoiler",
					Description: "Whether to embed the video as a spoiler",
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Required:    false,
				},
			},
		},
		{
			Name:        "version",
			Description: "Show application and dependency versions",
		},
	}
}

// errMsg attempts to parse an exec.ExitError to return a more useful error message.
func errMsg(err error) string {
	if err == nil {
		return ""
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
		return string(exitErr.Stderr)
	}

	return err.Error()
}
