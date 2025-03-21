package handlers

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

func NewReadyHandler(appid string, commands []*discordgo.ApplicationCommand) interface{} {
	return func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as %s", r.User.String())

		for _, g := range r.Guilds {
			// Register commands
			_, err := s.ApplicationCommandBulkOverwrite(appid, g.ID, commands)
			if err != nil {
				log.Printf("could not register commands for guild %s: %s", g.ID, err)
			}
		}
	}
}
