package discord

import (
	"os"
	"os/signal"
	"overseer/discord/handlers"
	"syscall"

	"github.com/bwmarrin/discordgo"
	charm "github.com/charmbracelet/log"
)

type defaultDiscordServer struct {
	botToken string
	log      *charm.Logger
}

func (d *defaultDiscordServer) Connect() error {
	session, err := discordgo.New("Bot " + d.botToken)
	if err != nil {
		d.log.Error("failed to create discord session", "error", err)
		return err
	}

	session.AddHandler(handlers.MessageCreate)
	session.Identify.Intents = discordgo.IntentsGuildMessages

	err = session.Open()
	if err != nil {
		d.log.Error("failed to open connection to discord", "error", err)
		return err
	}

	d.log.Info("server waiting for events")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	d.log.Info("shutting down discord server")
	return nil
}
