package discord

import (
	"context"
	"discord-embedder/internal/config"
	"discord-embedder/internal/logger"
	"discord-embedder/internal/video"
	"fmt"
	"net/url"
	"slices"

	"github.com/bwmarrin/discordgo"
	slogctx "github.com/veqryn/slog-context"
)

const loadingEmoji = "<a:bongocat1:499924216490229763>"

func onMessage(ctx context.Context) any {
	return func(s *discordgo.Session, i *discordgo.MessageCreate) {
		cfg := config.FromContext(ctx)

		if i.Author.ID == s.State.User.ID {
			return
		}
		if !slices.Contains(cfg.DiscordChannelIDs, i.ChannelID) {
			return
		}
		// Check if message is URL
		u, err := url.ParseRequestURI(i.Message.Content)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return
		}

		log := logger.FromContext(ctx)
		nctx := slogctx.Append(ctx, "message_id", i.Message.ID)

		// Handle message
		var msg *discordgo.Message
		msg, err = handleMessage(nctx, s, i)
		if err != nil {
			// Respond with error message
			_, err = s.ChannelMessageSend(i.ChannelID, errMsg(err))
			if err != nil {
				log.ErrorContext(nctx, "could not send video to discord", "error", err)
			}

			return
		}

		// Add reaction to new message
		if err = s.MessageReactionAdd(i.ChannelID, msg.ID, reaction(i.Author.ID)); err != nil {
			log.ErrorContext(nctx, "could not add reaction to message", "error", err)
		}

		// Delete original message
		if err = s.ChannelMessageDelete(i.ChannelID, i.Message.ID); err != nil {
			log.ErrorContext(nctx, "could not delete original message", "error", err)
		}
	}
}

func handleMessage(
	ctx context.Context,
	s *discordgo.Session,
	i *discordgo.MessageCreate,
) (*discordgo.Message, error) {
	log := logger.FromContext(ctx)
	cfg := config.FromContext(ctx)

	// Defer delete
	thinkingMessage, err := s.ChannelMessageSend(i.ChannelID, fmt.Sprintf("%s Thinking...", loadingEmoji))
	if err != nil {
		log.ErrorContext(ctx, "could not send message", "error", err)
		return nil, err
	}
	defer func() {
		if err = s.ChannelMessageDelete(i.ChannelID, thinkingMessage.ID); err != nil {
			log.ErrorContext(ctx, "could not delete message", "error", err)
		}
	}()
	ctx = slogctx.Append(ctx, "url", i.Message.Content)

	// Download video
	video, err := video.Download(ctx, i.Message.Content)
	if err != nil {
		log.ErrorContext(ctx, "could not get video", "error", err)
		return nil, err
	}
	ctx = slogctx.Append(ctx, "video_id", video.ID)

	// Compress video
	log.InfoContext(ctx, "compressing")
	if err = video.Compress(ctx); err != nil {
		log.ErrorContext(ctx, "could not compress video", "error", err)
		return nil, err
	}

	// Respond with message
	log.InfoContext(ctx, "sending")
	videoembed := fmt.Sprintf("-# [.](%s/%s)", cfg.Host, video.ID)
	return s.ChannelMessageSend(i.ChannelID, videoembed)
}

func reaction(authorID string) string {
	switch authorID {
	// Trevor
	case "669341931415011378":
		return "trev:237958248446165002"

	// Logan
	case "107300949189513216":
		return "birthdayboi:395419807933267969"

	// Matthew
	case "104981063356420096":
		return "mac:762454563550134332"

	// Adum
	case "269296454797885453":
		return "PogAdam:813996648967569408"

	// Dom
	case "412306881885765633":
		return "Bruh2:828831877720178718"

	// Bobby
	case "227588032298090496":
		return "bobert:856203132169486356"

	// Shane
	case "107549180053950464":
		return "shen:692279372765200424"
	}

	return ""
}
