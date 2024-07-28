/*
Copyright © 2024 Angad Behl
*/
package cmd

import (
	"github.com/slashtechno/gobackup-github/internal"
	"github.com/slashtechno/gobackup-github/pkg/backup"
	"github.com/spf13/cobra"
)

// continuousCmd represents the continuous command
var continuousCmd = &cobra.Command{
	Use:   "continuous [flags]",
	Short: "Start a rolling backup that backs up repositories at a set interval",
	Long:  `Start a rolling backup that backs up repositories at a set interval. The output directory will be emptied each time the backup is run.`,
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
			internal.Viper.GetString("interval"),
		)
	},
}

func init() {
	backupCmd.AddCommand(continuousCmd)

	continuousCmd.Flags().StringP("interval", "i", "", "Interval to check for new content")
	internal.Viper.BindPFlag("interval", continuousCmd.Flags().Lookup("interval"))
	internal.Viper.SetDefault("interval", "")

}
