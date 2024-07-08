package handlers

import (
	"context"
	"overseer/common"
	"overseer/discord/commands"

	"github.com/bwmarrin/discordgo"
)

var interactionCreateLogger = common.GetLogger("discord.handlers.interaction")

func InteractionCreate(session *discordgo.Session, event *discordgo.InteractionCreate) {
	interactionCreateLogger.Debug("interaction received", "type", event.Type, "command", event.ApplicationCommandData().Name, "user", event.User.Username, "guild", event.GuildID)
	if handler, ok := commands.Commands[event.ApplicationCommandData().Name]; ok {
		interactionCreateLogger.Debug("handler found for command", "command", event.ApplicationCommandData().Name, "user", event.User.Username, "guild", event.GuildID)
		// todo: enrich context with dependencies
		handler.Handler(context.TODO(), session, event)
		interactionCreateLogger.Debug("handler executed", "command", event.ApplicationCommandData().Name, "user", event.User.Username, "guild", event.GuildID)
	} else {
		interactionCreateLogger.Error("no handler found for command", "command", event.ApplicationCommandData().Name)
	}
}
