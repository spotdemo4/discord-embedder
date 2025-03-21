package handlers

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

func NewJoinHandler(appid string, commands []*discordgo.ApplicationCommand) interface{} {
	return func(s *discordgo.Session, e *discordgo.GuildCreate) {
		// Register commands
		_, err := s.ApplicationCommandBulkOverwrite(appid, e.Guild.ID, commands)
		if err != nil {
			log.Printf("could not register commands for guild %s: %s", e.Guild.ID, err)
		}
	}
}
