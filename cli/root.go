package cli

import (
	"os"

	del "github.com/52funny/pikpakcli/cli/del"
	"github.com/52funny/pikpakcli/cli/download"
	"github.com/52funny/pikpakcli/cli/list"
	"github.com/52funny/pikpakcli/cli/new"
	"github.com/52funny/pikpakcli/cli/quota"
	"github.com/52funny/pikpakcli/cli/rename"
	"github.com/52funny/pikpakcli/cli/share"
	"github.com/52funny/pikpakcli/cli/upload"
	"github.com/52funny/pikpakcli/conf"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pikpakcli",
	Short: "Pikpakcli is a command line interface for Pikpak",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		err := conf.InitConfig(configPath)
		if err != nil {
			logrus.Errorln(err)
			os.Exit(1)
		}
		if debug {
			logrus.SetLevel(logrus.DebugLevel)
		}
	},
}

// Config path
var configPath string

// Debug mode
var debug bool

// Initialize the command line interface
func init() {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "debug mode")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "config.yml", "config file path")
	rootCmd.AddCommand(upload.UploadCmd)
	rootCmd.AddCommand(download.DownloadCmd)
	rootCmd.AddCommand(share.ShareCommand)
	rootCmd.AddCommand(new.NewCommand)
	rootCmd.AddCommand(quota.QuotaCmd)
	rootCmd.AddCommand(list.ListCmd)
	rootCmd.AddCommand(del.DeleteCmd)
	rootCmd.AddCommand(rename.RenameCmd)
	rootCmd.AddCommand(shellCmd)
}

// Execute the command line interface
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Errorln(err)
		os.Exit(1)
	}
}
