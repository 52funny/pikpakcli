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
		}
		q, err := p.GetQuota()
		if err != nil {
			logrus.Errorln("get cloud quota error:", err)
			return
		}
		fmt.Printf("%-20s%-20s\n", "total", "used")
		switch human {
		case true:
			fmt.Printf("%-20s%-20s\n", utils.FormatStorage(q.Limit, true), utils.FormatStorage(q.Usage, true))
		case false:
			fmt.Printf("%-20s%-20s\n", q.Limit, q.Usage)
		}
	},
}

func init() {
	QuotaCmd.Flags().BoolVarP(&human, "human", "H", false, "display human readable format")
}
