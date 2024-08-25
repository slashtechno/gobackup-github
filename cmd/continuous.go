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

// continuousCmd represents the continuous command
var continuousCmd = &cobra.Command{
	Use:   "continuous --interval INTERVAL",
	Short: "Start a rolling backup that backs up repositories at a set interval",
	Long:  `Start a rolling backup that backs up repositories at a set interval. The output directory will be emptied each time the backup is run.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := backup.StartBackup(
			backup.BackupConfig{
				Usernames:   internal.Viper.GetStringSlice("usernames"),
				InOrg:       internal.Viper.GetStringSlice("in-org"),
				BackupStars: internal.Viper.GetBool("stars"),
				Token:       internal.Viper.GetString("token"),
				Output:      internal.Viper.GetString("output"),
				RunType:     internal.Viper.GetString("run-type"),
			},
			internal.Viper.GetString("interval"),
			internal.Viper.GetInt("max-backups"),
		)
		if err != nil {
			log.Error("Backup failed", "err", err)
		}
	},
}

func init() {
	backupCmd.AddCommand(continuousCmd)

	continuousCmd.Flags().StringP("interval", "i", "", "Interval to check for new content")
	internal.Viper.BindPFlag("interval", continuousCmd.Flags().Lookup("interval"))
	internal.Viper.SetDefault("interval", "24h")

	continuousCmd.Flags().IntP("max-backups", "n", 0, "Number of backups to keep")
	internal.Viper.BindPFlag("max-backups", continuousCmd.Flags().Lookup("max-backups"))
	internal.Viper.SetDefault("max-backups", 1)

}
