package quota

import (
	"fmt"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/api"
	"github.com/52funny/pikpakcli/internal/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var human bool

var QuotaCmd = &cobra.Command{
	Use:   "quota",
	Short: `Get the quota for the pikpak cloud`,
	Run: func(cmd *cobra.Command, args []string) {
		p := api.NewPikPakWithContext(cmd.Context(), conf.Config.Username, conf.Config.Password)
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

		transfer, err := p.GetTransferQuota()
		if err != nil {
			logrus.Errorln("get transfer quota error:", err)
			return
		}
		displayMonthlyTransferQuota(transfer.Base)
	},
}

func init() {
	QuotaCmd.Flags().BoolVarP(&human, "human", "H", false, "display human readable format")
}

func displayCloudDownload(cloudDownload api.Quota) {
	fmt.Printf("\ncloud download:\n")
	fmt.Printf("%-20s%-20s%-20s\n", "total", "used", "remaining")
	remaining, err := cloudDownload.Remaining()
	if err != nil {
		fmt.Printf("%-20s%-20s%-20s\n", formatQuotaValue(cloudDownload.Limit), formatQuotaValue(cloudDownload.Usage), "N/A")
		return
	}
	fmt.Printf("%-20s%-20s%-20s\n", formatQuotaValue(cloudDownload.Limit), formatQuotaValue(cloudDownload.Usage), formatTransferValue(remaining))
}

func displayMonthlyTransferQuota(base api.TransferQuotaBase) {
	fmt.Printf("\nmonthly transfer:\n")
	fmt.Printf("%-20s%-20s%-20s%-20s\n", "type", "total", "used", "remaining")
	displayTransferRow("cloud download", base.Offline)
	displayTransferRow("download", base.Download)
	displayTransferRow("upload", base.Upload)
}

func displayTransferRow(name string, quota api.TransferQuota) {
	fmt.Printf(
		"%-20s%-20s%-20s%-20s\n",
		name,
		formatTransferValue(quota.TotalAssets),
		formatTransferValue(quota.Assets),
		formatTransferValue(quota.Remaining()),
	)
}

func formatTransferValue(size int64) string {
	return utils.FormatStorage(fmt.Sprintf("%d", size), human)
}

func formatQuotaValue(size string) string {
	return utils.FormatStorage(size, human)
}
