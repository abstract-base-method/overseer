package cmd

import (
	"github.com/spf13/cobra"
	"overseer/common"
	"overseer/discord"
)

var botToken string

var discordCmd = &cobra.Command{
	Use:   "discord",
	Short: "start up a discord bot",
	Long: `This is the primary entrypoint for the discord bot.
This allows for the provisioning of a discord bot to service requests from Discord`,
	Run: func(cmd *cobra.Command, args []string) {
		log := common.GetLogger("cli.discord")
		log.Debug("starting discord command")

		var server discord.DiscordServer
		if botToken != "" {
			log.Info("configuring discord from command line")
			server = discord.NewDiscordServer(botToken)
		} else if common.GetConfiguration().Discord.BotToken != "" {
			log.Info("configuring discord from configuration")
			server = discord.NewDiscordServer(common.GetConfiguration().Discord.BotToken)
		} else {
			log.Fatal("no bot token provided")
			return
		}

		if err := server.Connect(); err != nil {
			log.Fatal("failed to start discord server", "error", err)
		} else {
			log.Info("discord server shutdown")
		}
	},
}

func init() {
	rootCmd.AddCommand(discordCmd)

	discordCmd.Flags().StringVarP(&botToken, "bot-token", "t", "", "the bot token to use for the discord bot")
}
