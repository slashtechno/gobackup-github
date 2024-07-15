/*
Copyright © 2024 Angad Behl
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var ConfigFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	// Use:   "cobra",
	Short: "A tool to backup GitHub repos, stars, and gists",
	// Long:  ``,
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

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cobra.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.PersistentFlags().StringVar(&ConfigFile, "config", "config.yaml", "config file path")
	rootCmd.MarkPersistentFlagFilename("config", "toml", "yaml", "json")
	rootCmd.PersistentFlags().StringP("log-level", "l", "asdasda", "Log level")
	viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))
}
