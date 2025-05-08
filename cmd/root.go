package cmd

import (
	"os"

	"github.com/52funny/pikpakcli/cmd/download"
	"github.com/52funny/pikpakcli/cmd/list"
	"github.com/52funny/pikpakcli/cmd/new"
	"github.com/52funny/pikpakcli/cmd/quota"
	"github.com/52funny/pikpakcli/cmd/share"
	"github.com/52funny/pikpakcli/cmd/upload"

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
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		err := conf.InitConfig(configPath)
		if err != nil {
			return err
		}
		if debug {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
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
	rootCmd.AddCommand(quota.QuotaCmd)
	rootCmd.AddCommand(list.ListCmd)
}

// Execute the command line interface
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Errorln(err)
		os.Exit(1)
	}
}
