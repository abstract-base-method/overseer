package cmd

import (
	"fmt"
	"net"
	"overseer/common"
	"overseer/server"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/reflection"
)

var serverPort int
var enableReflection bool

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "start up a dnd server",
	Long: `This is the primary entrypoint for the server.
This allows for the provisioning of a gRPC server and discord
connection to service requests from Discord`,
	Run: func(cmd *cobra.Command, args []string) {
		log := common.GetLogger("cli.server")
		log.Debug("starting server command", "port", serverPort)

		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", serverPort))
		if err != nil {
			log.Fatal("failed to allocate port", "error", err)
		}

		server, err := server.NewServer()
		if err != nil {
			log.Fatal("failed to create server", "error", err)
		}

		if enableReflection {
			log.Warn("enabling reflection")
			reflection.Register(server)
		}

		log.Info("starting server", "port", serverPort)
		if err := server.Serve(listener); err != nil {
			log.Fatal("failed to start server", "error", err)
		} else {
			log.Info("server shutdown")
		}
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serverCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serverCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	serverCmd.Flags().IntVar(&serverPort, "port", 4242, "port to listen on")
	serverCmd.Flags().BoolVarP(&enableReflection, "reflection", "r", false, "enable reflection")
}
