package discord

import (
	"context"
	"discord-embedder/internal/video"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func (d *Discord) interactionHandler(ctx context.Context) interface{} {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		data := i.ApplicationCommandData()
		switch data.Name {
		case "embed":
			d.handleEmbed(ctx, s, i, parseOptions(data.Options))

		default:
			d.Logger.InfoContext(ctx, "unknown command", "cmd", data.Name)
		}
	}
}

func (d *Discord) handleEmbed(
	ctx context.Context,
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	opts optionMap,
) {
	// defer response
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		d.Logger.WarnContext(ctx, "could not defer interaction", "error", err)
	}

	// download video
	video, err := video.Download(ctx, opts["url"].StringValue(), d.FilesDir)
	if err != nil {
		d.respond(s, i, fmt.Sprintf("could not download video: %s", err.Error()))
		return
	}

	// trim video if start and end times are provided
	if opts["start"] != nil && opts["end"] != nil {
		d.Logger.InfoContext(
			ctx,
			"trimming video",
			"start",
			opts["start"].StringValue(),
			"end",
			opts["end"].StringValue(),
		)

		if err = video.Trim(ctx, opts["start"].StringValue(), opts["end"].StringValue()); err != nil {
			d.respond(s, i, fmt.Sprintf("could not trim video: %s", err.Error()))
			return
		}
	}

	d.Logger.InfoContext(ctx, "compressing video", "file", video.AbsolutePath)
	if err = video.Compress(ctx, d.Quicksync); err != nil {
		d.respond(s, i, fmt.Sprintf("could not compress video: %s", err.Error()))
		return
	}

	d.Logger.InfoContext(ctx, "generating thumbnail", "file", video.AbsolutePath)
	if err = video.Thumbnail(ctx); err != nil {
		d.respond(s, i, fmt.Sprintf("could not generate thumbnail: %s", err.Error()))
		return
	}

	// surround with spoiler tags if requested
	videoembed := ""
	if opts["spoiler"] != nil && opts["spoiler"].BoolValue() {
		videoembed = fmt.Sprintf("-# || [.](%s/%s) ||", d.Host, video.ID)
	} else {
		videoembed = fmt.Sprintf("-# [.](%s/%s)", d.Host, video.ID)
	}

	// respond with message
	d.Logger.InfoContext(ctx, "sending video", "id", video.ID)
	message, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &videoembed,
	})
	if err != nil || message == nil {
		d.Logger.ErrorContext(ctx, "could not send video to discord", "error", err)
		return
	}

	// add reaction to message
	if err = s.MessageReactionAdd(i.ChannelID, message.ID, "👍"); err != nil {
		d.Logger.ErrorContext(ctx, "could not add reaction to message", "error", err)
		return
	}
}

func (d *Discord) respond(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	d.Logger.Warn("responding with error", "error", message)

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &message,
	}); err != nil {
		d.Logger.Error("could not respond to interaction", "error", err)
	}
}

type optionMap = map[string]*discordgo.ApplicationCommandInteractionDataOption

func parseOptions(options []*discordgo.ApplicationCommandInteractionDataOption) optionMap {
	om := make(optionMap)
	for _, opt := range options {
		om[opt.Name] = opt
	}
	return om
}
