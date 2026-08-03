package save

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/api"
	"github.com/spf13/cobra"
)

var SaveCommand = &cobra.Command{
	Use:     "save",
	Aliases: []string{"s"},
	Short:   `Save shared file links to your pikpak (mypikpak.com/s/xxx)`,
	Long: `Save PikPak share links (mypikpak.com/s/<id>) to your own drive.

Usage:
  pikpakcli save <share_url> [pass_code]

If the share requires a passcode, provide it as the second argument.
Files are restored into the "Pack From Shared" folder.`,
	Example: `  pikpakcli save https://mypikpak.com/s/<share_id>
  pikpakcli save https://mypikpak.com/s/<share_id> dd3e`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shareURL := strings.TrimSpace(args[0])
		passCode := ""
		if len(args) > 1 {
			passCode = strings.TrimSpace(args[1])
		}

		shareID := parseShareID(shareURL)
		if shareID == "" {
			return fmt.Errorf("invalid share link, expected: https://mypikpak.com/s/<share_id>")
		}

		p := api.NewPikPakWithContext(cmd.Context(), conf.Config.Username, conf.Config.Password)
		if err := p.Login(); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		// 1) 获取分享信息 + pass_code_token + 顶层文件列表
		fmt.Printf("Fetching share %s ...\n", shareID)
		token, title, fileIDs, err := p.ShareInfo(shareID, passCode)
		if err != nil {
			return fmt.Errorf("access share: %w", err)
		}
		if title != "" {
			fmt.Printf("Share title: %s (%d files)\n", title, len(fileIDs))
		}

		// 2) 转存
		status, err := p.ShareRestore(shareID, token, fileIDs)
		if err != nil {
			return fmt.Errorf("restore: %w", err)
		}
		if status == "RESTORE_START" || status == "RESTORE_SUCCESS" {
			fmt.Println("Saved! Check your 'Pack From Shared' folder.")
		} else {
			fmt.Println("Restore status:", status)
		}
		return nil
	},
}

var reShareID = regexp.MustCompile(`(?:mypikpak\.com|pan\.mypikpak\.com)/s/([0-9A-Za-z_-]+)`)

func parseShareID(url string) string {
	if m := reShareID.FindStringSubmatch(url); len(m) > 1 {
		return m[1]
	}
	// 直接传 share id（裸 id 或带后缀 token 的 URL）
	u := strings.TrimSpace(url)
	u = strings.TrimSuffix(u, "/")
	if strings.Contains(u, "/") {
		// 形如 mypikpak.com/s/<id>/<token>
		parts := strings.Split(u, "/")
		for i, part := range parts {
			if part == "s" && i+1 < len(parts) {
				return parts[i+1]
			}
		}
		return ""
	}
	return u
}
