package cmd

import (
	"os"

	"github.com/52funny/pikpakcli/cmd/download"
	"github.com/52funny/pikpakcli/cmd/embed"
	"github.com/52funny/pikpakcli/cmd/list"
	"github.com/52funny/pikpakcli/cmd/new"
	"github.com/52funny/pikpakcli/cmd/quota"
	"github.com/52funny/pikpakcli/cmd/share"
	"github.com/52funny/pikpakcli/cmd/upload"
	del "github.com/52funny/pikpakcli/cmd/del"
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
	// debug
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "debug mode")
	// config
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "config.yml", "config file path")
	rootCmd.AddCommand(upload.UploadCmd)
	rootCmd.AddCommand(download.DownloadCmd)
	rootCmd.AddCommand(share.ShareCommand)
	rootCmd.AddCommand(new.NewCommand)
	rootCmd.AddCommand(embed.EmbedCmd)
	rootCmd.AddCommand(quota.QuotaCmd)
	rootCmd.AddCommand(list.ListCmd)
	rootCmd.AddCommand(del.DeleteCmd)
}

// Execute the command line interface
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Errorln(err)
		os.Exit(1)
	}
}
