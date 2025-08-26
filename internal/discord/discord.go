package discord

import (
	"context"
	"discord-embedder/internal/app"

	"github.com/bwmarrin/discordgo"
)

type Discord struct {
	*app.App

	session *discordgo.Session
}

func New(ctx context.Context, a *app.App) (*Discord, error) {
	// create a new Discord session using the provided bot token
	session, err := discordgo.New("Bot " + a.DiscordToken)
	if err != nil {
		a.Logger.ErrorContext(ctx, "could not create discord session", "error", err)
		return nil, err
	}

	d := &Discord{
		a,
		session,
	}

	commands := []*discordgo.ApplicationCommand{
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
					Name:        "spoiler",
					Description: "Whether to embed the video as a spoiler",
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Required:    false,
				},
			},
		},
	}

	// add discord handlers
	session.AddHandler(d.interactionHandler(ctx))
	session.AddHandler(d.messageHandler(ctx))
	session.AddHandler(d.readyHandler(commands))
	session.AddHandler(d.joinHandler(commands))

	// add intents
	session.Identify.Intents = discordgo.IntentsDirectMessages | discordgo.IntentsGuildMessages

	// start the websocket connection to Discord
	err = session.Open()
	if err != nil {
		d.Logger.ErrorContext(ctx, "could not open session", "error", err)
		return nil, err
	}

	return d, nil
}

func (d *Discord) Close() error {
	return d.session.Close()
}
