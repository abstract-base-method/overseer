package commands

import (
	"context"
	"overseer/common"

	"github.com/bwmarrin/discordgo"
)

var devCommandLog = common.GetLogger("discord.commands.dev")
var devCommand = &discordgo.ApplicationCommand{
	Name:        "dev",
	Description: "execute developer operations",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Name:        "commands",
			Description: "Reset commands for the guild",
			Type:        discordgo.ApplicationCommandOptionSubCommand,
		},
	},
}

func devCommandFunc(ctx context.Context, session *discordgo.Session, event *discordgo.InteractionCreate) {
	options := event.ApplicationCommandData().Options
	devCommandLog.Info("dev command executed", "user", event.User.Username, "guild", event.GuildID, "options", options)

	switch options[0].Name {
	case "commands":
		devResetCommands(ctx, session, event)
	default:
		devCommandLog.Error("unknown subcommand", "subcommand", options[0].Name)
		if err := session.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Unknown subcommand",
			},
		}); err != nil {
			devCommandLog.Error("failed to respond to invalid subcommand", "error", err)
		}
	}
}

func devResetCommands(ctx context.Context, session *discordgo.Session, event *discordgo.InteractionCreate) {
	devCommandLog.Info("resetting commands", "user", event.User.Username, "guild", event.GuildID)
	err := session.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Resetting commands",
		},
	})
	if err != nil {
		devCommandLog.Error("failed to respond to sub-command for resetting commands", "error", err)
	}

	err = DeregisterCommands(session, event.GuildID)
	if err != nil {
		devCommandLog.Error("failed to deregister commands", "error", err, "guild", event.GuildID)
	}

	err = session.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Commands deregistered",
		},
	})
	if err != nil {
		devCommandLog.Error("failed to respond to deregistering commands", "error", err)
	}

	err = RegisterCommands(session, event.GuildID)
	if err != nil {
		devCommandLog.Error("failed to register commands", "error", err, "guild", event.GuildID)
	}

	err = session.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Commands registered",
		},
	})
	if err != nil {
		devCommandLog.Error("failed to respond to registering commands", "error", err)
	}
}
