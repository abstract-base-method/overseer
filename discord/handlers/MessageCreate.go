package handlers

import (
	"github.com/bwmarrin/discordgo"
)

func MessageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {
	if message.Author.ID == session.State.User.ID {
		return
	}

	if message.Content == "ping" {
		session.ChannelMessageSend(message.ChannelID, "Pong!")
	}
}
