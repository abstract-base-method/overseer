package handlers

import (
	"overseer/common"

	"github.com/bwmarrin/discordgo"
)

var messageCreateLog = common.GetLogger("discord.handlers.messageCreate")

func MessageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {
	if message.Author.ID == session.State.User.ID {
		return
	}

	if message.Content == "ping" {
		messageCreateLog.Info("ping command received", "author", message.Author.Username)
		session.ChannelMessageSend(message.ChannelID, "Pong!")
	}
}
