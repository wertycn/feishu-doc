package cli

import (
	"fmt"

	"github.com/wertycn/feishu-doc/internal/config"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "管理飞书应用凭证配置",
}

var (
	cfgAppID     string
	cfgAppSecret string
	cfgDomain    string
)

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "保存 App ID 和 App Secret (AES-256-GCM 加密存储)",
	Example: `  feishu-doc config set --app-id cli_a1b2c3 --app-secret xxxxxxxxxxxx`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := &config.Config{
			AppID:     cfgAppID,
			AppSecret: cfgAppSecret,
			Domain:    cfgDomain,
		}
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("保存失败: %w", err)
		}
		fmt.Printf("配置已加密保存到 %s\n", config.Dir())
		fmt.Printf("  App ID:     %s\n", config.Mask(cfg.AppID))
		fmt.Printf("  App Secret: %s\n", config.Mask(cfg.AppSecret))
		if cfg.Domain != "" {
			fmt.Printf("  Domain:     %s\n", cfg.Domain)
		}
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "显示当前保存的配置",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		fmt.Printf("配置文件: %s\n", config.Dir())
		fmt.Printf("  App ID:     %s\n", config.Mask(cfg.AppID))
		fmt.Printf("  App Secret: %s\n", config.Mask(cfg.AppSecret))
		if cfg.Domain != "" {
			fmt.Printf("  Domain:     %s\n", cfg.Domain)
		}
		return nil
	},
}

func init() {
	configSetCmd.Flags().StringVar(&cfgAppID, "app-id", "", "飞书应用 App ID (必填)")
	configSetCmd.Flags().StringVar(&cfgAppSecret, "app-secret", "", "飞书应用 App Secret (必填)")
	configSetCmd.Flags().StringVar(&cfgDomain, "domain", "", "飞书域名 (默认: feishu.cn，海外: larksuite.com)")
	_ = configSetCmd.MarkFlagRequired("app-id")
	_ = configSetCmd.MarkFlagRequired("app-secret")

	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
}
