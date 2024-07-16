/*
Copyright © 2024 Angad Behl
*/
package cmd

import (
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/slashtechno/gobackup-github/cobra/internal"
	"github.com/spf13/cobra"
)

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		startBackup(
			internal.Viper.GetString("username"),
			internal.Viper.GetString("token"),
			internal.Viper.GetString("output"),
			internal.Viper.GetString("interval"),
		)
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)

	backupCmd.PersistentFlags().StringP("username", "u", "", "GitHub username to backup. Leave blank to backup the authenticated user")
	internal.Viper.BindPFlag("username", backupCmd.PersistentFlags().Lookup("username"))
	internal.Viper.SetDefault("username", "")

	backupCmd.PersistentFlags().StringP("token", "t", "", "GitHub token")
	internal.Viper.BindPFlag("token", backupCmd.PersistentFlags().Lookup("token"))
	internal.Viper.SetDefault("token", "")

	backupCmd.PersistentFlags().StringP("output", "o", "", "Output directory")
	internal.Viper.BindPFlag("output", backupCmd.PersistentFlags().Lookup("output"))
	internal.Viper.SetDefault("output", "backup")

	backupCmd.Flags().StringP("interval", "i", "", "Interval to check for new content")
	internal.Viper.BindPFlag("interval", backupCmd.Flags().Lookup("interval"))
}

func startBackup(
	username string,
	token string,
	output string,
	interval string,
) {

	backupConfig := BackupConfig{
		Username: username,
		Token:    token,
		Output:   output,
	}

	// https://gobyexample.com/tickers
	if interval != "" {
		log.Info("Starting backup with interval", "interval", interval)

		ticker := time.NewTicker(internal.Viper.GetDuration("interval"))
		defer ticker.Stop()
		var wg sync.WaitGroup
		wg.Add(1)

		// Run backup on start
		backup(backupConfig)
		go func() {
			for range ticker.C {
				backup(backupConfig)
			}
		}()
		wg.Wait()
	}
	log.Info("Starting backup")
	backup(backupConfig)

}

type BackupConfig struct {
	Username string
	Token    string
	Output   string
}

func backup(config BackupConfig) {
	// Do backup
	log.Debug("Backup", "config", config)
	// https://github.com/google/go-github

}
