package discord

import (
	"context"
	"discord-embedder/internal/config"
	"discord-embedder/internal/logger"
	"discord-embedder/internal/video"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
	slogctx "github.com/veqryn/slog-context"
)

func onInteraction(ctx context.Context) any {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		log := logger.FromContext(ctx)
		data := i.ApplicationCommandData()

		switch data.Name {
		case "embed":
			nctx := slogctx.Append(ctx, "interaction_id", i.ID)

			var opts embed
			err := parseOptions(data.Options, &opts)
			if err != nil {
				log.ErrorContext(nctx, "could not parse options", "error", err)
				return
			}

			err = handleEmbed(nctx, s, i, opts)
			if err != nil {
				// Respond with error message
				errMsg := errMsg(err)
				_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: &errMsg,
				})
				if err != nil {
					log.ErrorContext(nctx, "could not respond to interaction", "error", err)
				}

				return
			}

		default:
			log.InfoContext(ctx, "unknown command", "cmd", data.Name)
		}
	}
}

type embed struct {
	URL     string `json:"url"`
	Start   string `json:"start"`
	End     string `json:"end"`
	Spoiler bool   `json:"spoiler"`
}

func handleEmbed(
	ctx context.Context,
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	opts embed,
) error {
	log := logger.FromContext(ctx)
	cfg := config.FromContext(ctx)

	// Defer response
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		log.WarnContext(ctx, "could not defer interaction", "error", err)
	}

	// Validate opts
	if opts.URL == "" {
		return errors.New("url is required")
	}
	if (opts.Start != "" && opts.End == "") || (opts.Start == "" && opts.End != "") {
		return errors.New("both start and end must be provided")
	}
	ctx = slogctx.Append(ctx, "url", opts.URL)

	// Download video
	video, err := video.Download(ctx, opts.URL)
	if err != nil {
		log.ErrorContext(ctx, "could not download video", "error", err)
		return err
	}
	ctx = slogctx.Append(ctx, "video_id", video.ID)

	// Trim video if start and end times are provided
	if opts.Start != "" && opts.End != "" {
		log.InfoContext(ctx, "trimming", "start", opts.Start, "end", opts.End)

		if err = video.Trim(ctx, opts.Start, opts.End); err != nil {
			log.ErrorContext(ctx, "could not trim video", "error", err)
			return err
		}
	}

	// Compress video
	log.InfoContext(ctx, "compressing")
	if err = video.Compress(ctx); err != nil {
		log.ErrorContext(ctx, "could not compress video", "error", err)
		return err
	}

	// Surround with spoiler tags if requested
	var embed string
	if opts.Spoiler {
		embed = fmt.Sprintf("-# || [.](%s/%s) ||", cfg.Host, video.ID)
	} else {
		embed = fmt.Sprintf("-# [.](%s/%s)", cfg.Host, video.ID)
	}

	// Respond with message
	log.InfoContext(ctx, "sending")
	message, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &embed,
	})
	if err != nil || message == nil {
		log.ErrorContext(ctx, "could not send video to discord", "error", err)
		return err
	}

	return nil
}

func parseOptions(options []*discordgo.ApplicationCommandInteractionDataOption, command any) error {
	om := map[string]any{}
	for _, opt := range options {
		om[opt.Name] = opt.Value
	}

	jsonData, err := json.Marshal(om)
	if err != nil {
		return err
	}

	err = json.Unmarshal(jsonData, command)
	if err != nil {
		return err
	}

	return nil
}
