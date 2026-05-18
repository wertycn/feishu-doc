package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/debugicu/feishu-doc/core"
	"github.com/debugicu/feishu-doc/internal/config"

	"github.com/spf13/cobra"
)

var (
	appID     string
	appSecret string
	domain    string
	client    *core.Client
)

var rootCmd = &cobra.Command{
	Use:   "feishu-doc",
	Short: "飞书文档操作 CLI 工具",
	Long:  "飞书文档与 Markdown 双向转换的命令行工具，支持文档的创建、查询、导出和导入操作。",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		switch cmd.Name() {
		case "set", "show", "config":
			return nil
		}

		if appID == "" {
			appID = os.Getenv("FEISHU_APP_ID")
		}
		if appSecret == "" {
			appSecret = os.Getenv("FEISHU_APP_SECRET")
		}

		var savedCfg *config.Config
		if appID == "" || appSecret == "" {
			cfg, err := config.Load()
			if err == nil {
				savedCfg = cfg
				if appID == "" {
					appID = cfg.AppID
				}
				if appSecret == "" {
					appSecret = cfg.AppSecret
				}
			}
		} else {
			savedCfg, _ = config.Load()
		}

		if domain == "" {
			domain = os.Getenv("FEISHU_DOMAIN")
		}
		if domain == "" && savedCfg != nil && savedCfg.Domain != "" {
			domain = savedCfg.Domain
		}

		if appID == "" || appSecret == "" {
			return fmt.Errorf("凭证未配置，请通过以下任一方式提供:\n" +
				"  1. feishu-doc config set --app-id <ID> --app-secret <SECRET>\n" +
				"  2. 设置环境变量 FEISHU_APP_ID / FEISHU_APP_SECRET\n" +
				"  3. 使用 --app-id / --app-secret 参数")
		}

		client = core.NewClient(appID, appSecret, domain)

		switch cmd.Name() {
		case "login", "status", "auth":
			return nil
		}

		if savedCfg == nil {
			savedCfg, _ = config.Load()
		}
		if savedCfg != nil {
			if savedCfg.TokenValid() {
				client.UserAccessToken = savedCfg.UserAccessToken
			} else if savedCfg.RefreshValid() {
				fmt.Fprintln(os.Stderr, "User token 已过期，正在自动刷新...")
				if err := refreshAndSave(savedCfg); err != nil {
					fmt.Fprintf(os.Stderr, "自动刷新失败: %v\n请运行 feishu-doc auth login 重新授权\n", err)
				} else {
					client.UserAccessToken = savedCfg.UserAccessToken
					fmt.Fprintln(os.Stderr, "Token 刷新成功")
				}
			}
		}

		if client.UserAccessToken == "" {
			fmt.Fprintln(os.Stderr, "提示: 未检测到用户授权，将使用应用身份访问 (权限受限)")
			fmt.Fprintln(os.Stderr, "运行 feishu-doc auth login 进行用户授权以获取完整权限")
		}

		return nil
	},
}

func refreshAndSave(cfg *config.Config) error {
	ctx := context.Background()
	result, err := client.RefreshUserToken(ctx, cfg.RefreshToken)
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	cfg.UserAccessToken = result.AccessToken
	cfg.RefreshToken = result.RefreshToken
	cfg.TokenExpiresAt = now + int64(result.ExpiresIn)
	cfg.RefreshExpiresAt = now + int64(result.RefreshExpiresIn)
	return config.Save(cfg)
}

func init() {
	rootCmd.PersistentFlags().StringVar(&appID, "app-id", "", "飞书应用 App ID")
	rootCmd.PersistentFlags().StringVar(&appSecret, "app-secret", "", "飞书应用 App Secret")
	rootCmd.PersistentFlags().StringVar(&domain, "domain", "", "飞书域名 (默认: feishu.cn，海外: larksuite.com)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
