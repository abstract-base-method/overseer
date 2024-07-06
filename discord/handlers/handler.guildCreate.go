package handlers

import (
	"overseer/common"
	"overseer/discord/commands"

	"github.com/bwmarrin/discordgo"
)

var guildCreateLog = common.GetLogger("discord.handlers.guildCreate")

func GuildCreate(session *discordgo.Session, event *discordgo.GuildCreate) {
	guildCreateLog.Info("guild created", "name", event.Guild.Name)

	err := commands.RegisterCommands(session, event.Guild.ID)
	if err != nil {
		guildCreateLog.Error("failed to register commands", "error", err, "guild", event.Guild.ID)
	}
}
