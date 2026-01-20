package discord

import (
	"context"
	"discord-embedder/internal/config"
	"discord-embedder/internal/logger"

	"github.com/bwmarrin/discordgo"
)

func onReady(ctx context.Context, commands []*discordgo.ApplicationCommand) any {
	return func(s *discordgo.Session, r *discordgo.Ready) {
		log := logger.FromContext(ctx)
		cfg := config.FromContext(ctx)

		log.InfoContext(ctx, "Logged in", "user", r.User)

		for _, g := range r.Guilds {
			// Register commands
			_, err := s.ApplicationCommandBulkOverwrite(cfg.DiscordApplicationID, g.ID, commands)
			if err != nil {
				log.WarnContext(ctx, "could not register commands for guild", "id", g.ID, "error", err)
			}
		}
	}
}
