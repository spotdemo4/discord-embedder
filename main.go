package main

import (
	"context"
	"discord-embedder/internal/handlers"
	"embed"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

//go:embed templates/home.html
var home embed.FS

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
			{
				Name:        "spoiler",
				Description: "Whether to embed the video as a spoiler",
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Required:    false,
			},
		},
	},
}

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Failed to load .env file, using environment variables")
	}

	// Get env
	DiscordToken := os.Getenv("DISCORD_TOKEN")
	if DiscordToken == "" {
		log.Fatalf("env DISCORD_TOKEN not set")
	}
	DiscordApplicationID := os.Getenv("DISCORD_APPLICATION_ID")
	if DiscordApplicationID == "" {
		log.Fatalf("env DISCORD_APPLICATION_ID not set")
	}
	Host := os.Getenv("HOST")
	if Host == "" {
		log.Fatalf("env HOST not set")
	}
	Quicksync := os.Getenv("QUICKSYNC") == "true"

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

	// Check if files directory exists
	if _, err := os.Stat("files"); os.IsNotExist(err) {
		// Create cookies directory
		if err := os.Mkdir("files", 0755); err != nil {
			log.Fatalf("could not create files directory: %s", err)
		}
	}

	// Check if download directory exists
	if _, err := os.Stat("download"); os.IsNotExist(err) {
		// Create cookies directory
		if err := os.Mkdir("download", 0755); err != nil {
			log.Fatalf("could not create download directory: %s", err)
		}
	}

	// Create a new Discord session using the provided bot token
	session, err := discordgo.New("Bot " + DiscordToken)
	if err != nil {
		log.Fatalf("could not create session: %s", err)
	}

	// Add discord handlers
	session.AddHandler(handlers.NewInteractionHandler(Host, Quicksync))
	session.AddHandler(handlers.NewMessageHandler(Host, Quicksync))
	session.AddHandler(handlers.NewReadyHandler(DiscordApplicationID, commands))
	session.AddHandler(handlers.NewJoinHandler(DiscordApplicationID, commands))

	// Add intents
	session.Identify.Intents = discordgo.IntentsDirectMessages | discordgo.IntentsGuildMessages

	// Add server handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/", handlers.NewHomeHandler(home, Host))
	mux.HandleFunc("/files/", handlers.NewFileHandler())

	// Create HTTP server
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Gracefully shutdown on SIGINT or SIGTERM
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Printf("Received signal %s", sig)
		log.Println("Exiting")

		// Close discord connection
		err = session.Close()
		if err != nil {
			log.Printf("could not close session gracefully: %s", err)
		}

		// Close webserver
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := server.Shutdown(ctx); err != nil {
			server.Close()
		}
		cancel()
	}()

	// Start the websocket connection to Discord
	err = session.Open()
	if err != nil {
		log.Fatalf("could not open session: %s", err)
	}

	// Start http server
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err.Error())
	}
}
