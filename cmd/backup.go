/*
Copyright © 2024 Angad Behl
*/
package cmd

import (
	"github.com/charmbracelet/log"
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
		err := backup.StartBackup(
			backup.BackupConfig{
				Usernames:   internal.Viper.GetStringSlice("usernames"),
				InOrg:       internal.Viper.GetStringSlice("in-org"),
				BackupStars: internal.Viper.GetBool("stars"),
				Token:       internal.Viper.GetString("token"),
				Output:      internal.Viper.GetString("output"),
				RunType:     internal.Viper.GetString("run-type"),
				NtfyUrl:     internal.Viper.GetString("ntfy-url"),
			},
			// Pass an empty interval as this is a one-time backup
			"",
			0,
		)
		if err != nil {
			log.Error("Backup failed", "err", err)
		}
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

	backupCmd.PersistentFlags().String("run-type", "", "`Type of backup: clone` (clone the repositories), `fetch` (fetch the repositories and write to output if it ends in .json or `repositories.json` in output), `dry-run` (fetch the repositories and print the output). Default is `clone`")
	internal.Viper.BindPFlag("run-type", backupCmd.PersistentFlags().Lookup("run-type"))
	internal.Viper.SetDefault("run-type", "clone")

	backupCmd.PersistentFlags().String("ntfy-url", "", "Ntfy URL to send a notification to after backup")
	internal.Viper.BindPFlag("ntfy-url", backupCmd.PersistentFlags().Lookup("ntfy-url"))
	internal.Viper.SetDefault("ntfy-url", "")

}
