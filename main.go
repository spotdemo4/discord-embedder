package main

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
)

var commands = []*discordgo.ApplicationCommand{
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
		},
	},
}

func main() {
	// Check if yt-dlp is installed
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		log.Fatalf("yt-dlp is not installed")
	}

	// Check if ffmpeg is installed
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		log.Fatalf("ffmpeg is not installed")
	}

	// Check if ffprobe is installed
	if _, err := exec.LookPath("ffprobe"); err != nil {
		log.Fatalf("ffprobe is not installed")
	}

	// Check if cookies directory exists
	if _, err := os.Stat("cookies"); os.IsNotExist(err) {
		// Create cookies directory
		if err := os.Mkdir("cookies", 0755); err != nil {
			log.Fatalf("could not create cookies directory: %s", err)
		}
	}

	// Read in environment variables
	env := env{}
	if err := env.read(); err != nil {
		log.Fatalf("could not read environment variables: %s", err)
	}

	// Create a new Discord session using the provided bot token.
	session, err := discordgo.New("Bot " + env.DiscordToken)
	if err != nil {
		log.Fatalf("could not create session: %s", err)
	}

	// Add command handlers
	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		data := i.ApplicationCommandData()
		switch data.Name {
		case "embed":
			handleEmbed(s, i, parseOptions(data.Options))

		default:
			log.Printf("unknown command: %s", data.Name)

		}
	})

	// Add direct message handler
	session.AddHandler(func(s *discordgo.Session, i *discordgo.MessageCreate) {
		if i.Author.ID == s.State.User.ID {
			return
		}

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
	})

	// Add ready handler
	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as %s", r.User.String())

		for _, g := range r.Guilds {
			// Register commands
			_, err = session.ApplicationCommandBulkOverwrite(env.DiscordApplicationID, g.ID, commands)
			if err != nil {
				log.Printf("could not register commands for guild %s: %s", g.ID, err)
			}
		}
	})

	// Add on join guild handler
	session.AddHandler(func(s *discordgo.Session, e *discordgo.GuildCreate) {
		// Register commands
		_, err = session.ApplicationCommandBulkOverwrite(env.DiscordApplicationID, e.Guild.ID, commands)
		if err != nil {
			log.Printf("could not register commands for guild %s: %s", e.Guild.ID, err)
		}
	})

	session.Identify.Intents = discordgo.IntentsDirectMessages

	// Open the websocket connection to Discord and begin listening.
	err = session.Open()
	if err != nil {
		log.Fatalf("could not open session: %s", err)
	}

	// Wait here until CTRL-C or other term signal is received.
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	<-sigch

	err = session.Close()
	if err != nil {
		log.Printf("could not close session gracefully: %s", err)
	}
}

type env struct {
	// Discord API token
	DiscordToken         string `mapstructure:"DISCORD_TOKEN"`
	DiscordApplicationID string `mapstructure:"DISCORD_APPLICATION_ID"`
}

func (e *env) read() error {
	// Get config directory
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(userConfigDir, "discord-embedder")

	// Check if config directory exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		// Create config directory
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return errors.New("could not create config directory")
		}
	}

	// Check if env file exists
	if _, err := os.Stat(filepath.Join(configDir, "config.env")); os.IsNotExist(err) {
		// Create env file
		file, err := os.Create(filepath.Join(configDir, "config.env"))
		if err != nil {
			return errors.New("could not create config.env file")
		}
		defer file.Close()
	}

	// Set env file name and path
	viper.SetConfigName("config.env")
	viper.AddConfigPath(configDir)
	viper.SetConfigType("env")

	// Read in env file
	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	// Read in environment variables
	if err := viper.BindEnv("DISCORD_TOKEN"); err != nil {
		return err
	}
	if err := viper.BindEnv("DISCORD_APPLICATION_ID"); err != nil {
		return err
	}

	// Unmarshal env variables
	if err := viper.Unmarshal(e); err != nil {
		return err
	}

	// Check if Discord token is set
	if e.DiscordToken == "" {
		return errors.New("DISCORD_TOKEN is not set")
	}
	if e.DiscordApplicationID == "" {
		return errors.New("DISCORD_APPLICATION_ID is not set")
	}

	return nil
}

type optionMap = map[string]*discordgo.ApplicationCommandInteractionDataOption

func parseOptions(options []*discordgo.ApplicationCommandInteractionDataOption) (om optionMap) {
	om = make(optionMap)
	for _, opt := range options {
		om[opt.Name] = opt
	}
	return
}

func handleEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, opts optionMap) {
	// Defer response
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		log.Printf("could not respond to interaction: %s", err)
	}

	// Get URL
	URL, err := url.Parse(opts["url"].StringValue())
	if err != nil {
		resp := fmt.Sprintf("could not parse url: %s", err)
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &resp,
		}); err != nil {
			log.Printf("could not respond to interaction: %s", err)
		}

		return
	}

	video := &video{
		Name: hex.EncodeToString([]byte(opts["url"].StringValue())),
		Url:  URL,
	}

	// Download video
	log.Printf("downloading video: %s", video.Url)
	if err := video.download(); err != nil {
		resp := fmt.Sprintf("could not download video: %s", err)
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &resp,
		}); err != nil {
			log.Printf("could not respond to interaction: %s", err)
		}

		return
	}
	defer func() {
		if err := video.delete(); err != nil {
			log.Printf("could not delete video: %s", err)
		}
	}()

	// Trim video if start and end times are provided
	if opts["start"] != nil && opts["end"] != nil {
		log.Printf("trimming video: %s", video.File.Name())
		if err := video.trim(opts["start"].StringValue(), opts["end"].StringValue()); err != nil {
			resp := fmt.Sprintf("could not trim video: %s", err)
			if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &resp,
			}); err != nil {
				log.Printf("could not respond to interaction: %s", err)
			}

			return
		}
	}

	// Convert to H264
	codec, err := video.codec()
	if err != nil {
		log.Printf("could not get codec: %s", err)
	} else {
		if codec != "h264" {
			log.Printf("converting video: %s", video.File.Name())
			if err := video.convert(); err != nil {
				resp := fmt.Sprintf("could not convert video: %s", err)
				if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: &resp,
				}); err != nil {
					log.Printf("could not respond to interaction: %s", err)
				}

				return
			}
		}
	}

	// Compress the video if >25MB
	if info, err := video.File.Stat(); err != nil {
		log.Printf("could not get file info: %s", err)
	} else {
		if info.Size() > 25*1000*1000 {
			log.Printf("compressing video: %s", video.File.Name())
			if err := video.compress(); err != nil {
				resp := fmt.Sprintf("could not compress video: %s", err)
				if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: &resp,
				}); err != nil {
					log.Printf("could not respond to interaction: %s", err)
				}

				return
			}
		}
	}

	// Respond with video
	log.Printf("responding with video: %s", video.File.Name())
	message, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Files: []*discordgo.File{
			{
				Name:        video.File.Name(),
				ContentType: "video/mp4",
				Reader:      video.File,
			},
		},
	})
	if err != nil {
		resp := "Could not upload to Discord!"
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &resp,
		}); err != nil {
			log.Printf("could not respond to interaction: %s", err)
		}

		return
	}

	// Add reaction to message
	if err := s.MessageReactionAdd(i.ChannelID, message.ID, "👍"); err != nil {
		log.Printf("could not add reaction to message: %s", err)
	}
}

type video struct {
	Name string
	Url  *url.URL
	File *os.File
}

// download downloads the video
func (v *video) download() error {
	// Find domain of URL
	domain := strings.TrimPrefix(v.Url.Hostname(), "www.")

	// Check if cookie file exists for URL
	cookieFileName := ""
	err := filepath.Walk("cookies", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if cookie file exists for domain
		if strings.Contains(info.Name(), domain) {
			cookieFileName = info.Name()
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Download video
	var cmd *exec.Cmd
	if cookieFileName == "" {
		cmd = exec.Command("yt-dlp", "-o", fmt.Sprintf("%s.%%(ext)s", v.Name), v.Url.String())
	} else {
		log.Printf("using cookie file: %s", cookieFileName)
		cmd = exec.Command("yt-dlp", "-o", fmt.Sprintf("%s.%%(ext)s", v.Name), "--cookies", filepath.Join("cookies", cookieFileName), v.Url.String())
	}

	if err := cmd.Run(); err != nil {
		return err
	}

	// Find video file
	if err := v.find(); err != nil {
		return err
	}

	return nil
}

// find finds the video file
func (v *video) find() error {
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasPrefix(info.Name(), v.Name) {
			v.File, err = os.Open(info.Name())
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}
	if v.File == nil {
		return errors.New("could not find video file")
	}

	return nil
}

// delete deletes the video file
func (v *video) delete() error {
	if err := v.File.Close(); err != nil {
		log.Printf("could not close file: %s", err)
	}

	if err := os.Remove(v.File.Name()); err != nil {
		return err
	}

	return nil
}

// codec returns the codec of the video
func (v *video) codec() (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=codec_name", "-of", "default=noprint_wrappers=1:nokey=1", v.File.Name())

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(out), nil
}

// convert converts the video to H264
func (v *video) convert() error {
	cmd := exec.Command("ffmpeg", "-i", v.File.Name(), "-c:v", "libx264", "-c:a", "aac", "-b:a", "160k", fmt.Sprintf("%s-convert.mp4", v.Name))

	if err := cmd.Run(); err != nil {
		return err
	}

	// Delete original video
	if err := v.delete(); err != nil {
		return err
	}

	// Set new video name
	v.Name = fmt.Sprintf("%s-convert", v.Name)

	// Find new video file
	if err := v.find(); err != nil {
		return err
	}

	return nil
}

// compress compresses the video to <25MB
func (v *video) compress() error {
	// Get length of video
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", v.File.Name())

	out, err := cmd.Output()
	if err != nil {
		return err
	}

	// Get length of video without decimals
	length, err := strconv.Atoi(strings.SplitN(string(out), ".", 2)[0])
	if err != nil {
		return err
	}
	length = length + 1

	// Compresses video to <25MB
	targetSize := 25 * 1000 * 1000 * 8
	totalBitrate := targetSize / length
	audioBitrate := 128 * 1000
	videoBitrate := totalBitrate - audioBitrate

	cmd = exec.Command("ffmpeg",
		"-i", v.File.Name(),
		"-b:v", strconv.Itoa(videoBitrate),
		"-maxrate:v", strconv.Itoa(videoBitrate),
		"-bufsize:v", strconv.Itoa(targetSize/20),
		"-b:a", strconv.Itoa(audioBitrate),
		fmt.Sprintf("%s-compress.mp4", v.Name),
	)

	if err := cmd.Run(); err != nil {
		return err
	}

	// Delete original video
	if err := v.delete(); err != nil {
		return err
	}

	// Set new video name
	v.Name = fmt.Sprintf("%s-compress", v.Name)

	// Find new video file
	if err := v.find(); err != nil {
		return err
	}

	return nil
}

// Trim video to start and end time
func (v *video) trim(start string, end string) error {
	cmd := exec.Command("ffmpeg", "-ss", start, "-to", end, "-i", v.File.Name(), fmt.Sprintf("%s-trim.mp4", v.Name))

	if err := cmd.Run(); err != nil {
		return err
	}

	// Delete original video
	if err := v.delete(); err != nil {
		return err
	}

	// Set new video name
	v.Name = fmt.Sprintf("%s-trim", v.Name)

	// Find new video file
	if err := v.find(); err != nil {
		return err
	}

	return nil
}
