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
		}
		q, err := p.GetQuota()
		if err != nil {
			logrus.Errorln("get cloud quota error:", err)
			return
		}
		fmt.Printf("%-20s%-20s\n", "total", "used")
		switch human {
		case true:
			fmt.Printf("%-20s%-20s\n", displayStorage(q.Limit), displayStorage(q.Usage))
		case false:
			fmt.Printf("%-20s%-20s\n", q.Limit, q.Usage)
		}
	},
}

func init() {
	QuotaCmd.Flags().BoolVarP(&human, "human", "H", false, "display human readable format")
}

func displayStorage(s string) string {
	size, _ := strconv.ParseFloat(s, 64)
	cnt := 0
	for size > 1024 {
		cnt += 1
		if cnt > 5 {
			break
		}
		size /= 1024
	}
	// res := strconv.Itoa(int(size))
	res := strconv.FormatFloat(size, 'g', 2, 64)
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
