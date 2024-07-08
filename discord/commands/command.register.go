package commands

import (
	"context"
	"overseer/common"

	"github.com/bwmarrin/discordgo"
)

var registerCommandLog = common.GetLogger("discord.commands.register")
var registerCommand = &discordgo.ApplicationCommand{
	Name:        "register",
	Description: "register a new user",
}

func registerCommandFunc(ctx context.Context, session *discordgo.Session, event *discordgo.InteractionCreate) {
	err := session.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Registering user",
		},
	})
	if err != nil {
		registerCommandLog.Error("failed to respond to register command", "error", err)
	}
}
