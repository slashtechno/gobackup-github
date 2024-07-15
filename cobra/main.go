/*
Copyright © 2024 Angad Behl
*/
package main

import (
	"io/fs"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
	"github.com/slashtechno/gobackup-github/cobra/cmd"
	"github.com/slashtechno/gobackup-github/cobra/internal"
	"github.com/spf13/cobra"
)

func init() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Warn("Failed to load .env file", "error", err)
	}
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	internal.Viper.SetEnvPrefix("GOBACKUP_GITHUB")
	// https://github.com/spf13/viper?tab=readme-ov-file#working-with-environment-variables
	internal.Viper.AutomaticEnv()
	internal.Viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	// cmd.ConfigFile should always be set since it has a default
	if cmd.ConfigFile != "" {
		internal.Viper.SetConfigFile(cmd.ConfigFile)
	} else {
		log.Warn("Default config file flag value not retrievable")
		internal.Viper.SetConfigFile("config.yaml")
	}

	if err := internal.Viper.ReadInConfig(); err == nil {
		log.Debug("Configuration file loaded", "file", internal.Viper.ConfigFileUsed())
	} else {
		// Generate a default .env file null values
		if _, ok := err.(*fs.PathError); ok {
			log.Debug("Configuration file not found, creating a new one", "file", cmd.ConfigFile)

			internal.Viper.SetDefault("log-level", "info")
			internal.Viper.SetDefault("github-token", "")

			if err := internal.Viper.WriteConfigAs(cmd.ConfigFile); err != nil {
				log.Fatal("Failed to write configuration file:", err)
			}
		} else {
			log.Fatal("Failed to read configurationfile:", "error", err)
		}
	}
}

func main() {
	// TODO: This is executed before InitConfig since OnInitialize is called when Execute is called. Thus, this doesn't correctly set the log level
	switch strings.ToLower(internal.Viper.GetString("log-level")) {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
		log.Info("Invalid log level passed, using InfoLevel", "passed", internal.Viper.GetString("log_level"))
	}

	cmd.Execute()
}
