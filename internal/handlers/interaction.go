package handlers

import (
	"discord-embedder/internal/video"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/gabriel-vasile/mimetype"
)

func NewInteractionHandler(host string) interface{} {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		data := i.ApplicationCommandData()
		switch data.Name {
		case "embed":
			handleEmbed(s, i, parseOptions(data.Options), host)

		default:
			log.Printf("unknown command: %s", data.Name)
		}
	}
}

type optionMap = map[string]*discordgo.ApplicationCommandInteractionDataOption

func parseOptions(options []*discordgo.ApplicationCommandInteractionDataOption) (om optionMap) {
	om = make(optionMap)
	for _, opt := range options {
		om[opt.Name] = opt
	}
	return
}

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	log.Printf("ERROR: %s\n", message)
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &message,
	}); err != nil {
		log.Printf("Could not respond to interaction: %s", err.Error())
	}
}

func handleEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, opts optionMap, host string) {
	// Defer response
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		log.Printf("Could not respond to interaction: %s", err.Error())
	}

	video, err := video.New(opts["url"].StringValue())
	if err != nil {
		log.Printf("Could not get video: %s", err.Error())
	}

	// Download video
	log.Printf("Downloading video: %s", video.Url)
	if err := video.Download(); err != nil {
		respond(s, i, fmt.Sprintf("Could not download video: %s", err.Error()))
		return
	}
	defer func() {
		if err := video.Delete(); err != nil {
			log.Printf("Could not delete video: %s\n", err.Error())
		}
	}()

	// Trim video if start and end times are provided
	if opts["start"] != nil && opts["end"] != nil {
		log.Printf("trimming video: %s", video.File.Name())
		if err := video.Trim(opts["start"].StringValue(), opts["end"].StringValue()); err != nil {
			respond(s, i, fmt.Sprintf("Could not trim video: %s", err.Error()))
			return
		}
	}

	// Add spoiler
	if opts["spoiler"] != nil && opts["spoiler"].BoolValue() {
		log.Printf("adding spoiler: %s", video.File.Name())
		if err := video.Spoiler(); err != nil {
			respond(s, i, fmt.Sprintf("Could not add spoiler: %s", err.Error()))
			return
		}
	}

	// Get video codec
	codec, err := video.Codec()
	if err != nil {
		respond(s, i, fmt.Sprintf("Could not get codec: %s", err.Error()))
		return
	}

	// Get video size
	info, err := video.File.Stat()
	if err != nil {
		respond(s, i, fmt.Sprintf("Could not get file info: %s", err.Error()))
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
		message, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Files: []*discordgo.File{
				{
					Name:        video.File.Name(),
					ContentType: mtype.String(),
					Reader:      video.File,
				},
			},
		})
		if err != nil {
			respond(s, i, fmt.Sprintf("Could not upload to Discord: %s", err.Error()))
			return
		}
	} else {
		log.Printf("Compressing video: %s", video.File.Name())
		if err := video.Compress(); err != nil {
			respond(s, i, fmt.Sprintf("Could not compress video: %s", err.Error()))
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
		fileurl := fmt.Sprintf("%s/%s", host, video.ID)
		message, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &fileurl,
		})
		if err != nil {
			log.Printf("Could not send video to discord: %s", err.Error())
			return
		}
	}

	if message == nil {
		return
	}

	// Add reaction to message
	if err := s.MessageReactionAdd(i.ChannelID, message.ID, "👍"); err != nil {
		log.Printf("Could not add reaction to message: %s", err)
	}
}
