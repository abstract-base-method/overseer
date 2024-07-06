package handlers

import (
	"overseer/common"
	"overseer/discord/commands"

	"github.com/bwmarrin/discordgo"
)

var interactionCreateLogger = common.GetLogger("discord.handlers.interaction")

func InteractionCreate(session *discordgo.Session, event *discordgo.InteractionCreate) {
	interactionCreateLogger.Debug("interaction received", "type", event.Type, "command", event.ApplicationCommandData().Name, "user", event.User.Username, "guild", event.GuildID)
	if handler, ok := commands.Handlers[event.ApplicationCommandData().Name]; ok {
		interactionCreateLogger.Debug("handler found for command", "command", event.ApplicationCommandData().Name, "user", event.User.Username, "guild", event.GuildID)
		handler(session, event)
		interactionCreateLogger.Debug("handler executed", "command", event.ApplicationCommandData().Name, "user", event.User.Username, "guild", event.GuildID)
	} else {
		interactionCreateLogger.Error("no handler found for command", "command", event.ApplicationCommandData().Name)
	}
}
