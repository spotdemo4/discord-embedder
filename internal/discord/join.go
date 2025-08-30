package discord

import (
	"context"
	"discord-embedder/internal/config"
	"discord-embedder/internal/logger"

	"github.com/bwmarrin/discordgo"
)

func onJoin(ctx context.Context, commands []*discordgo.ApplicationCommand) any {
	return func(s *discordgo.Session, e *discordgo.GuildCreate) {
		log := logger.FromContext(ctx)
		cfg := config.FromContext(ctx)

		// register commands
		_, err := s.ApplicationCommandBulkOverwrite(cfg.DiscordApplicationID, e.Guild.ID, commands)
		if err != nil {
			log.WarnContext(ctx, "could not register commands for guild", "guild id", e.Guild.ID, "error", err)
		}
	}
}
