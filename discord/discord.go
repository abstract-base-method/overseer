package discord

import (
	"overseer/common"
)

func NewDiscordServer(botToken string) DiscordServer {
	return &defaultDiscordServer{
		botToken: botToken,
		log:      common.GetLogger("discord.server"),
	}
}
