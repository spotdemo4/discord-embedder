package discord

import (
	"github.com/bwmarrin/discordgo"
)

func (d *Discord) joinHandler(commands []*discordgo.ApplicationCommand) interface{} {
	return func(s *discordgo.Session, e *discordgo.GuildCreate) {
		// register commands
		_, err := s.ApplicationCommandBulkOverwrite(d.DiscordApplicationID, e.Guild.ID, commands)
		if err != nil {
			d.Logger.Warn("could not register commands for guild", "guild id", e.Guild.ID, "error", err)
		}
	}
}
