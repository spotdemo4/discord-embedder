package discord

import (
	"context"
	"discord-embedder/internal/config"
	"discord-embedder/internal/logger"
	"discord-embedder/internal/version"
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

		case "version":
			nctx := slogctx.Append(ctx, "interaction_id", i.ID)
			if err := handleVersion(nctx, s, i); err != nil {
				log.ErrorContext(nctx, "could not handle version interaction", "error", err)
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
	Speed   string `json:"speed"`
	Spoiler bool   `json:"spoiler"`
}

func embedSpeedFactor(speed string) (float64, error) {
	switch speed {
	case "", embedSpeedNormal:
		return 1, nil
	case embedSpeedOneAndHalf:
		return 1.5, nil
	case embedSpeedDouble:
		return 2, nil
	default:
		return 0, errors.New("speed must be one of x1, x1.5, or x2")
	}
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
	speed, err := embedSpeedFactor(opts.Speed)
	if err != nil {
		return err
	}
	ctx = slogctx.Append(ctx, "url", opts.URL, "speed", speed)

	// Download video
	v, err := video.Download(ctx, opts.URL)
	if err != nil {
		log.ErrorContext(ctx, "could not download video", "error", err)
		return err
	}
	ctx = slogctx.Append(ctx, "video_id", v.ID)

	// Trim video if start and end times are provided
	if opts.Start != "" && opts.End != "" {
		log.InfoContext(ctx, "trimming", "start", opts.Start, "end", opts.End)

		if err = v.Trim(ctx, opts.Start, opts.End); err != nil {
			log.ErrorContext(ctx, "could not trim video", "error", err)
			return err
		}
	}

	log.InfoContext(ctx, "compressing")
	if err = v.Compress(ctx, speed); err != nil {
		log.ErrorContext(ctx, "could not compress video", "error", err)
		return err
	}

	// Surround with spoiler tags if requested
	var embed string
	if opts.Spoiler {
		embed = fmt.Sprintf("-# || [.](%s/%s) ||", cfg.Host, v.ID)
	} else {
		embed = fmt.Sprintf("-# [.](%s/%s)", cfg.Host, v.ID)
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

func handleVersion(
	ctx context.Context,
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
) error {
	log := logger.FromContext(ctx)

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		return fmt.Errorf("defer version interaction: %w", err)
	}

	ytdlpVersion, err := version.YTDLP(ctx)
	if err != nil {
		log.WarnContext(ctx, "could not get yt-dlp version", "error", err)
		ytdlpVersion = "unknown"
	}

	ffmpegVersion, err := version.FFmpeg(ctx)
	if err != nil {
		log.WarnContext(ctx, "could not get ffmpeg version", "error", err)
		ffmpegVersion = "unknown"
	}

	content := fmt.Sprintf(
		"```\ndiscord-embedder: %s\nyt-dlp: %s\nffmpeg: %s\n```",
		version.Application,
		ytdlpVersion,
		ffmpegVersion,
	)
	message, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})
	if err != nil {
		return fmt.Errorf("send version response: %w", err)
	}
	if message == nil {
		return errors.New("send version response: empty message")
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
