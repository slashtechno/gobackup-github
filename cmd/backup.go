/*
Copyright © 2024 Angad Behl
*/
package cmd

import (
	"github.com/slashtechno/gobackup-github/internal"
	"github.com/slashtechno/gobackup-github/pkg/backup"
	"github.com/spf13/cobra"
)

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup a GitHub user",
	Long: `Backup either the authenticated user or a specified GitHub user.
	Backing up the authenticated user clones private repositories as well.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		backup.StartBackup(
			internal.Viper.GetStringSlice("usernames"),
			internal.Viper.GetStringSlice("in-org"),
			internal.Viper.GetString("token"),
			internal.Viper.GetString("output"),
			internal.Viper.GetString("interval"),
		)
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)

	// backupCmd.PersistentFlags().StringP("username", "u", "", "GitHub username to backup. Leave blank to backup the authenticated user")
	// internal.Viper.BindPFlag("username", backupCmd.PersistentFlags().Lookup("username"))
	// internal.Viper.SetDefault("username", "")

	backupCmd.PersistentFlags().StringSliceP("username", "u", []string{}, "GitHub username to backup. Leave blank to backup the authenticated user")
	internal.Viper.BindPFlag("usernames", backupCmd.PersistentFlags().Lookup("username"))
	internal.Viper.SetDefault("usernames",
		[]string{},
	)

	// Allow for the users in an organization to be fetched and backed up
	backupCmd.PersistentFlags().StringSlice("in-org", []string{}, "Get users from an organization")
	internal.Viper.BindPFlag("in-org", backupCmd.PersistentFlags().Lookup("in-org"))
	internal.Viper.SetDefault("in-org", []string{})

	backupCmd.PersistentFlags().StringP("token", "t", "", "GitHub token")
	internal.Viper.BindPFlag("token", backupCmd.PersistentFlags().Lookup("token"))
	internal.Viper.SetDefault("token", "")

	backupCmd.PersistentFlags().StringP("output", "o", "", "Output directory")
	internal.Viper.BindPFlag("output", backupCmd.PersistentFlags().Lookup("output"))
	internal.Viper.SetDefault("output", "backup")

	backupCmd.Flags().StringP("interval", "i", "", "Interval to check for new content")
	internal.Viper.BindPFlag("interval", backupCmd.Flags().Lookup("interval"))
}
