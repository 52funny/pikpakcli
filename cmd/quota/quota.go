package quota

import (
	"fmt"
	"strconv"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/pikpak"
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
			return // 加上这行，否则会继续执行
		}
		q, err := p.GetQuota()
		if err != nil {
			logrus.Errorln("get cloud quota error:", err)
			return
		}
		fmt.Println("Storage:")
		fmt.Printf("%-20s%-20s\n", "total", "used")
		if human {
			fmt.Printf("%-20s%-20s\n", displayStorage(q.Quota.Limit), displayStorage(q.Quota.Usage))
		} else {
			fmt.Printf("%-20s%-20s\n", q.Quota.Limit, q.Quota.Usage)
		}

		displayCloudDownload(q.Quotas.CloudDownload)
	},
}

func init() {
	QuotaCmd.Flags().BoolVarP(&human, "human", "H", false, "display human readable format")
}
func displayStorage(s string) string {
	size, _ := strconv.ParseFloat(s, 64)
	cnt := 0
	for size >= 1024 {
		cnt += 1
		if cnt > 5 {
			break
		}
		size /= 1024
	}

	var res string
	// 如果是整数则不显示小数点
	if size == float64(int64(size)) {
		res = strconv.FormatFloat(size, 'f', 0, 64)
	} else {
		res = strconv.FormatFloat(size, 'f', 2, 64)
	}

	switch cnt {
	case 0:
		res += "B"
	case 1:
		res += "KB"
	case 2:
		res += "MB"
	case 3:
		res += "GB"
	case 4:
		res += "TB"
	case 5:
		res += "PB"
	}
	return res
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
