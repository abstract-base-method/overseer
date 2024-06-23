package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var configFileLocation string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "overseer",
	Short: "AI Discord Adventures",
	Long: `overseer is a DnD mud.
It turns discord servers into places where people can 
participate in interactive LLM dungeon mastered stories 
in a variety of settings
`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&configFileLocation, "config", "", "config file (default is $HOME/.overseer.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
}
