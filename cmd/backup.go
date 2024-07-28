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
	Use:   "backup [flags]",
	Short: "Backup a GitHub user",
	Long: `Backup either the authenticated user or specified user(s). All users in specified organization(s) can also be backed up.
	Backing up the authenticated user clones private repositories as well.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		backup.StartBackup(
			backup.BackupConfig{
				Usernames:   internal.Viper.GetStringSlice("usernames"),
				InOrg:       internal.Viper.GetStringSlice("in-org"),
				BackupStars: internal.Viper.GetBool("stars"),
				Token:       internal.Viper.GetString("token"),
				Output:      internal.Viper.GetString("output"),
			},
			// Pass an empty interval as this is a one-time backup
			"",
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

	// Optionally, backup stars as well
	backupCmd.PersistentFlags().BoolP("backup-stars", "s", false, "Backup starred repositories")
	internal.Viper.BindPFlag("backup-stars", backupCmd.PersistentFlags().Lookup("stars"))
	internal.Viper.SetDefault("backup-stars", false)

}
