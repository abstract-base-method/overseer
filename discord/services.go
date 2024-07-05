package discord

// DiscordServer is the top level interface to operate a discord bot within overseer
type DiscordServer interface {
	// Connect will attempt a connection to discord and handle interactions
	// with guilds. This function will block until the connection is closed.
	Connect() error
}
