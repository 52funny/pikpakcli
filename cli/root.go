package cli

import (
	"fmt"
	"os"

	del "github.com/52funny/pikpakcli/cli/del"
	"github.com/52funny/pikpakcli/cli/download"
	"github.com/52funny/pikpakcli/cli/empty"
	"github.com/52funny/pikpakcli/cli/list"
	"github.com/52funny/pikpakcli/cli/new"
	"github.com/52funny/pikpakcli/cli/quota"
	"github.com/52funny/pikpakcli/cli/rename"
	"github.com/52funny/pikpakcli/cli/rubbish"
	"github.com/52funny/pikpakcli/cli/share"
	"github.com/52funny/pikpakcli/cli/upload"
	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/logx"
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
			fmt.Println("Init config failed")
			logx.Error(err)
			os.Exit(1)
		}
		logx.Init(debug, debugTopics)
	},
}

// Config path
var configPath string

// Debug mode
var debug bool
var debugTopics []string

// Initialize the command line interface
func init() {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "debug mode")
	rootCmd.PersistentFlags().StringSliceVar(&debugTopics, "debug-topic", nil, "enable debug topics: api,session,transfer")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "config.yml", "config file path")
	rootCmd.AddCommand(upload.UploadCmd)
	rootCmd.AddCommand(download.DownloadCmd)
	rootCmd.AddCommand(share.ShareCommand)
	rootCmd.AddCommand(new.NewCommand)
	rootCmd.AddCommand(quota.QuotaCmd)
	rootCmd.AddCommand(list.ListCmd)
	rootCmd.AddCommand(del.DeleteCmd)
	rootCmd.AddCommand(empty.EmptyCmd)
	rootCmd.AddCommand(rubbish.RubbishCmd)
	rootCmd.AddCommand(rename.RenameCmd)
	rootCmd.AddCommand(shellCmd)
}

// Execute the command line interface
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
