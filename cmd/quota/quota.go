package quota

import (
	"fmt"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/pikpak"
	"github.com/52funny/pikpakcli/internal/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var human bool

var QuotaCmd = &cobra.Command{
	Use:   "quota",
	Short: `Get the quota for the pikpak cloud`,
	Run: func(cmd *cobra.Command, args []string) {
		p := pikpak.NewPikPak(conf.Config.Username, conf.Config.Password)
		err := p.Login()
		if err != nil {
			logrus.Errorln("Login Failed:", err)
			return
		}
		q, err := p.GetQuota()
		if err != nil {
			logrus.Errorln("get cloud quota error:", err)
			return
		}
		fmt.Println("Storage:")
		fmt.Printf("%-20s%-20s\n", "total", "used")
		if human {
			fmt.Printf("%-20s%-20s\n", utils.FormatStorage(q.Quota.Limit, true), utils.FormatStorage(q.Quota.Usage, true))
		} else {
			fmt.Printf("%-20s%-20s\n", q.Quota.Limit, q.Quota.Usage)
		}

		displayCloudDownload(q.Quotas.CloudDownload)
	},
}

func init() {
	QuotaCmd.Flags().BoolVarP(&human, "human", "H", false, "display human readable format")
}

func displayCloudDownload(cloudDownload pikpak.Quota) {
	fmt.Printf("\ncloud download:\n")
	fmt.Printf("%-20s%-20s%-20s\n", "total", "used", "remaining")
	remaining, err := cloudDownload.Remaining()
	if err != nil {
		fmt.Printf("%-20s%-20s%-20s\n", cloudDownload.Limit, cloudDownload.Usage, "N/A")
		return
	}
	fmt.Printf("%-20s%-20s%-20d\n", cloudDownload.Limit, cloudDownload.Usage, remaining)
}
