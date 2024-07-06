package handlers

import (
	"overseer/common"

	"github.com/bwmarrin/discordgo"
)

var readyLog = common.GetLogger("discord.handlers.ready")

func Ready(session *discordgo.Session, event *discordgo.Ready) {
	readyLog.Info("bot ready", "username", event.User.Username)
}
