package discord

import (
	"github.com/bwmarrin/discordgo"
)

func (d *Discord) onReady(commands []*discordgo.ApplicationCommand) interface{} {
	return func(s *discordgo.Session, r *discordgo.Ready) {
		d.Logger.Info("Logged in", "user", r.User)

		for _, g := range r.Guilds {
			// Register commands
			_, err := s.ApplicationCommandBulkOverwrite(d.DiscordApplicationID, g.ID, commands)
			if err != nil {
				d.Logger.Warn("could not register commands for guild", "id", g.ID, "error", err)
			}
		}
	}
}
