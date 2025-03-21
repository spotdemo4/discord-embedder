package handlers

import (
	"bufio"
	"discord-embedder/internal/video"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"

	"github.com/bwmarrin/discordgo"
	"github.com/gabriel-vasile/mimetype"
)

func NewMessageHandler(host string) interface{} {
	return func(s *discordgo.Session, i *discordgo.MessageCreate) {
		if i.Author.ID == s.State.User.ID {
			return
		}

		// If this is a direct message
		if i.Message.GuildID == "" {
			handleDirectMessage(s, i)
		}

		// If this is a message in #memes
		if i.ChannelID == "150459222637805570" {
			// Check if message is URL
			u, err := url.ParseRequestURI(i.Message.Content)
			if err == nil && u.Scheme != "" && u.Host != "" {
				// Handle message
				handleMessage(s, i, host)
			}
		}
	}
}

func handleDirectMessage(s *discordgo.Session, i *discordgo.MessageCreate) {
	if i.Attachments == nil {
		return
	}

	for _, attachment := range i.Attachments {
		if filepath.Ext(attachment.Filename) != ".txt" {
			continue
		}

		// Save cookie file to cookies directory
		resp, err := s.Client.Get(attachment.URL)
		if err != nil {
			log.Printf("could not get cookie file: %s", err)
			return
		}

		file, err := os.Create(filepath.Join("cookies", attachment.Filename))
		if err != nil {
			log.Printf("could not create cookie file: %s", err)
			return
		}

		if _, err := bufio.NewReader(resp.Body).WriteTo(file); err != nil {
			log.Printf("could not write cookie file: %s", err)
			return
		}

		resp.Body.Close()
		file.Close()

		// Send response message
		if _, err := s.ChannelMessageSend(i.ChannelID, "Cookie file saved!"); err != nil {
			log.Printf("could not send message: %s", err)
		}
	}

	return
}

func handleMessage(s *discordgo.Session, i *discordgo.MessageCreate, host string) {
	// Defer delete
	thinkingMessage, err := s.ChannelMessageSend(i.ChannelID, "Thinking...")
	if err != nil {
		log.Printf("Could not send message: %s", err.Error())
		return
	}
	defer s.ChannelMessageDelete(thinkingMessage.ChannelID, thinkingMessage.ID)

	video, err := video.New(i.Message.Content)
	if err != nil {
		log.Printf("Could not get video: %s", err.Error())
	}

	// Download video
	log.Printf("Downloading video: %s", video.Url)
	if err := video.Download(); err != nil {
		log.Printf("Could not download video: %s", err.Error())
		return
	}
	defer func() {
		if err := video.Delete(); err != nil {
			log.Printf("Could not delete video: %s\n", err.Error())
		}
	}()

	// Get video codec
	codec, err := video.Codec()
	if err != nil {
		log.Printf("Could not get video codec: %s", err.Error())
		return
	}

	// Get video size
	info, err := video.File.Stat()
	if err != nil {
		log.Printf("Could not get video info: %s", err.Error())
		return
	}

	var message *discordgo.Message
	// Respond directly with video if h264 and less than 10MB
	if codec == "h264" && info.Size() < 10*1000*1000 {
		// Get content type
		mtype, err := mimetype.DetectFile(video.File.Name())
		if err != nil {
			log.Println("Could not get mime type, trying with default")
		}

		// Respond with video
		log.Printf("Sending video: %s", video.File.Name())
		message, err = s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
			Files: []*discordgo.File{
				{
					Name:        video.File.Name(),
					ContentType: mtype.String(),
					Reader:      video.File,
				},
			},
		})
		if err != nil {
			log.Printf("Could not send video to discord: %s", err.Error())
			return
		}
	} else {
		log.Printf("Compressing video: %s", video.File.Name())
		if err := video.Compress(); err != nil {
			log.Printf("Could not compress video: %s", err.Error())
			return
		}

		log.Printf("Exporting video: %s", video.File.Name())
		if err := video.Export(); err != nil {
			log.Printf("Could not export video: %s", err.Error())
			return
		}

		log.Printf("Generating thumbnail: %s", video.File.Name())
		if err := video.Thumbnail(); err != nil {
			log.Printf("Could not generate thumbnail: %s", err.Error())
			return
		}

		// Respond with message
		log.Printf("Sending message: %s", video.ID)
		message, err = s.ChannelMessageSend(i.ChannelID, fmt.Sprintf("%s/%s", host, video.ID))
		if err != nil {
			log.Printf("Could not send video to discord: %s", err.Error())
			return
		}
	}

	if message == nil {
		return
	}

	// Add reaction to message
	switch i.Author.ID {

	// Trevor
	case "669341931415011378":
		s.MessageReactionAdd(i.ChannelID, message.ID, "trev:237958248446165002")

	// Logan
	case "107300949189513216":
		s.MessageReactionAdd(i.ChannelID, message.ID, "birthdayboi:395419807933267969")

	// Matthew
	case "104981063356420096":
		s.MessageReactionAdd(i.ChannelID, message.ID, "mac:762454563550134332")

	// Adum
	case "269296454797885453":
		s.MessageReactionAdd(i.ChannelID, message.ID, "PogAdam:813996648967569408")

	// Dom
	case "412306881885765633":
		s.MessageReactionAdd(i.ChannelID, message.ID, "Bruh2:828831877720178718")

	// Bobby
	case "227588032298090496":
		s.MessageReactionAdd(i.ChannelID, message.ID, "bobert:856203132169486356")

	// Shane
	case "107549180053950464":
		s.MessageReactionAdd(i.ChannelID, message.ID, "shen:692279372765200424")
	}

	// Delete original message
	s.ChannelMessageDelete(i.ChannelID, i.Message.ID)
}
