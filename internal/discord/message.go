package discord

import (
	"context"
	"discord-embedder/internal/video"
	"fmt"
	"net/url"

	"github.com/bwmarrin/discordgo"
)

func (d *Discord) messageHandler(ctx context.Context) interface{} {
	return func(s *discordgo.Session, i *discordgo.MessageCreate) {
		if i.Author.ID == s.State.User.ID {
			return
		}

		// If this is a direct message
		if i.Message.GuildID == "" {
			// Check if message is URL
			u, err := url.ParseRequestURI(i.Message.Content)
			if err == nil && u.Scheme != "" && u.Host != "" {
				// Handle message
				if err = d.handleMessage(ctx, s, i); err != nil {
					d.Logger.ErrorContext(ctx, "could not handle direct message", "error", err)
				}
			}
		}

		// If this is a message in #memes
		if i.ChannelID == "150459222637805570" {
			// Check if message is URL
			u, err := url.ParseRequestURI(i.Message.Content)
			if err == nil && u.Scheme != "" && u.Host != "" {
				// Handle message
				if err = d.handleMessage(ctx, s, i); err != nil {
					d.Logger.ErrorContext(ctx, "could not handle channel message", "error", err)
				}
			}
		}
	}
}

func (d *Discord) handleMessage(ctx context.Context, s *discordgo.Session, i *discordgo.MessageCreate) error {
	// defer delete
	thinkingMessage, err := s.ChannelMessageSend(i.ChannelID, "Thinking...")
	if err != nil {
		d.Logger.ErrorContext(ctx, "could not send message", "error", err)
		return err
	}
	defer func() {
		if err = s.ChannelMessageDelete(i.ChannelID, thinkingMessage.ID); err != nil {
			d.Logger.ErrorContext(ctx, "could not delete message", "error", err)
		}
	}()

	// download video
	video, err := video.Download(ctx, i.Message.Content, d.FilesDir)
	if err != nil {
		d.Logger.ErrorContext(ctx, "could not get video", "error", err)
		return err
	}

	d.Logger.InfoContext(ctx, "compressing video", "file", video.AbsolutePath)
	if err = video.Compress(ctx, d.Quicksync); err != nil {
		d.Logger.ErrorContext(ctx, "could not compress video", "error", err)
		return err
	}

	d.Logger.InfoContext(ctx, "generating thumbnail", "file", video.AbsolutePath)
	if err = video.Thumbnail(ctx); err != nil {
		d.Logger.ErrorContext(ctx, "could not generate thumbnail", "error", err)
		return err
	}

	// respond with message
	d.Logger.InfoContext(ctx, "sending url", "id", video.ID)
	videoembed := fmt.Sprintf("-# [.](%s/%s)", d.Host, video.ID)
	message, err := s.ChannelMessageSend(i.ChannelID, videoembed)
	if err != nil || message == nil {
		d.Logger.ErrorContext(ctx, "could not send video to discord", "error", err)
		return err
	}

	// add reaction to message
	switch i.Author.ID {
	// Trevor
	case "669341931415011378":
		err = s.MessageReactionAdd(i.ChannelID, message.ID, "trev:237958248446165002")

	// Logan
	case "107300949189513216":
		err = s.MessageReactionAdd(i.ChannelID, message.ID, "birthdayboi:395419807933267969")

	// Matthew
	case "104981063356420096":
		err = s.MessageReactionAdd(i.ChannelID, message.ID, "mac:762454563550134332")

	// Adum
	case "269296454797885453":
		err = s.MessageReactionAdd(i.ChannelID, message.ID, "PogAdam:813996648967569408")

	// Dom
	case "412306881885765633":
		err = s.MessageReactionAdd(i.ChannelID, message.ID, "Bruh2:828831877720178718")

	// Bobby
	case "227588032298090496":
		err = s.MessageReactionAdd(i.ChannelID, message.ID, "bobert:856203132169486356")

	// Shane
	case "107549180053950464":
		err = s.MessageReactionAdd(i.ChannelID, message.ID, "shen:692279372765200424")
	}
	if err != nil {
		d.Logger.ErrorContext(ctx, "could not add reaction to message", "error", err)
		return err
	}

	// Delete original message
	return s.ChannelMessageDelete(i.ChannelID, i.Message.ID)
}
