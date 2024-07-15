/*
Copyright © 2024 Angad Behl
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		fmt.Println("backup called")
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// backupCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// backupCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	// backupCmd.Flags().GetBool("toggle")

	backupCmd.PersistentFlags().StringP("username", "u", "", "GitHub username")
	viper.BindPFlag("username", backupCmd.PersistentFlags().Lookup("username"))
	backupCmd.PersistentFlags().StringP("token", "t", "", "GitHub token")
	viper.BindPFlag("token", backupCmd.PersistentFlags().Lookup("token"))

}
