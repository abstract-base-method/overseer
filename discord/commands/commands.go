package commands

import (
	"context"
	"overseer/common"

	"github.com/bwmarrin/discordgo"
)

type OverseerCommand struct {
	Command *discordgo.ApplicationCommand
	Handler func(ctx context.Context, session *discordgo.Session, event *discordgo.InteractionCreate)
}

var Commands map[string]*OverseerCommand

func init() {
	Commands = map[string]*OverseerCommand{
		devCommand.Name: {
			Command: devCommand,
			Handler: devCommandFunc,
		},
		registerCommand.Name: {
			Command: registerCommand,
			Handler: registerCommandFunc,
		},
	}
}

var commandsLogger = common.GetLogger("discord.commands")

func RegisterCommands(session *discordgo.Session, guildId string) error {
	commandsLogger.Debug("registering commands", "guild", guildId, "count", len(Commands))
	for _, command := range Commands {
		commandsLogger.Debug("registering command", "command", command.Command.Name, "guild", guildId)
		cmd, err := session.ApplicationCommandCreate(session.State.User.ID, guildId, command.Command)
		if err != nil {
			commandsLogger.Error("failed to register command", "command", command.Command.Name, "error", err, "guild", guildId)
			return err
		}
		commandsLogger.Debug("registered command", "command", cmd.Name, "id", cmd.ID, "guild", guildId)
	}
	return nil
}

func DeregisterCommands(session *discordgo.Session, guildId string) error {
	commandsLogger.Debug("deregistering commands", "guild", guildId)
	registeredCommands, err := session.ApplicationCommands(session.State.User.ID, guildId)
	if err != nil {
		commandsLogger.Error("failed to get registered commands", "error", err, "guild", guildId)
		return err
	}

	commandsLogger.Debug("found registered commands", "count", len(registeredCommands), "guild", guildId)
	for _, command := range registeredCommands {
		commandsLogger.Debug("deregistering command", "command", command.Name, "id", command.ID, "guild", guildId)
		err := session.ApplicationCommandDelete(session.State.User.ID, guildId, command.ID)
		if err != nil {
			commandsLogger.Error("failed to deregister command", "command", command.Name, "error", err)
			return err
		}
		commandsLogger.Debug("deregistered command", "command", command.Name, "id", command.ID, "guild", guildId)
	}

	return nil
}
