package commands

import (
	"overseer/common"

	"github.com/bwmarrin/discordgo"
)

var Commands = []*discordgo.ApplicationCommand{
	devCommand,
}

var Handlers = map[string]func(session *discordgo.Session, event *discordgo.InteractionCreate){
	devCommand.Name: devCommandFunc,
}

var commandsLogger = common.GetLogger("discord.commands")

func RegisterCommands(session *discordgo.Session, guildId string) error {
	commandsLogger.Debug("registering commands", "guild", guildId, "count", len(Commands))
	for _, command := range Commands {
		commandsLogger.Debug("registering command", "command", command.Name, "guild", guildId)
		cmd, err := session.ApplicationCommandCreate(session.State.User.ID, guildId, command)
		if err != nil {
			commandsLogger.Error("failed to register command", "command", command.Name, "error", err, "guild", guildId)
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
