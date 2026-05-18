package cli

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/debugicu/feishu-doc/internal/config"

	"github.com/spf13/cobra"
)

const callbackPort = 9876

var defaultScopes = []string{
	"docx:document",
	"wiki:wiki",
	"docs:document.media:upload",
	"docs:document.media:download",
}

var allScopes = []string{
	"docx:document",
	"wiki:wiki",
	"docs:document.media:upload",
	"docs:document.media:download",
	"contact:contact.base:readonly",
}

var (
	authScopes string
	authAll    bool
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "飞书用户授权管理",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "通过浏览器登录飞书账号授权",
	Long: `启动本地 HTTP 服务，打开浏览器完成飞书 OAuth2 授权。

前置条件 (在飞书开放平台的应用配置中):
  1. 安全设置 > 重定向 URL 添加: http://127.0.0.1:9876/callback
  2. 权限管理 > 开启所需的 API 权限 (如 docx:document 等)
  3. 如果是企业自建应用，需要发布应用版本`,
	RunE: func(cmd *cobra.Command, args []string) error {
		codeCh := make(chan string, 1)
		errCh := make(chan error, 1)

		mux := http.NewServeMux()
		mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")
			if code == "" {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprint(w, "授权失败: 未收到授权码")
				errCh <- fmt.Errorf("callback 中未收到 code 参数")
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, `<html><body style="text-align:center;padding-top:80px;font-family:sans-serif">
				<h2>授权成功!</h2><p>请返回终端，此页面可以关闭。</p></body></html>`)
			codeCh <- code
		})

		listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", callbackPort))
		if err != nil {
			return fmt.Errorf("无法启动本地服务 (端口 %d 被占用): %w", callbackPort, err)
		}
		server := &http.Server{Handler: mux}
		go server.Serve(listener)
		defer server.Close()

		redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", callbackPort)
		var scopes string
		if authAll {
			scopes = strings.Join(allScopes, " ")
		} else if authScopes != "" {
			scopes = authScopes
		} else {
			scopes = strings.Join(defaultScopes, " ")
		}
		authDomain := "feishu.cn"
		if domain != "" {
			authDomain = domain
		}
		authURL := fmt.Sprintf(
			"https://open.%s/open-apis/authen/v1/authorize?app_id=%%s&redirect_uri=%%s&scope=%%s",
			authDomain,
		)
		authURL = fmt.Sprintf(authURL, appID, url.QueryEscape(redirectURI), url.QueryEscape(scopes))

		fmt.Println("正在打开浏览器进行飞书授权...")
		fmt.Printf("\n如浏览器未自动打开，请手动访问:\n%s\n\n", authURL)
		openBrowser(authURL)

		fmt.Println("等待授权回调...")

		select {
		case code := <-codeCh:
			return exchangeAndSave(code)
		case err := <-errCh:
			return err
		case <-time.After(3 * time.Minute):
			return fmt.Errorf("授权超时 (3分钟)")
		}
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看当前授权状态",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if cfg.UserAccessToken == "" {
			fmt.Println("状态: 未授权")
			fmt.Println("运行 feishu-doc auth login 进行授权")
			return nil
		}
		now := time.Now()
		tokenExpire := time.Unix(cfg.TokenExpiresAt, 0)
		refreshExpire := time.Unix(cfg.RefreshExpiresAt, 0)

		if cfg.TokenValid() {
			fmt.Println("状态: 已授权 (Token 有效)")
			fmt.Printf("  Token 过期时间:   %s (剩余 %s)\n", tokenExpire.Format("2006-01-02 15:04:05"), tokenExpire.Sub(now).Truncate(time.Second))
		} else if cfg.RefreshValid() {
			fmt.Println("状态: Token 已过期 (可自动刷新)")
		} else {
			fmt.Println("状态: Token 和 RefreshToken 均已过期，请重新登录")
			fmt.Println("运行 feishu-doc auth login 进行授权")
			return nil
		}
		fmt.Printf("  Refresh 过期时间: %s (剩余 %s)\n", refreshExpire.Format("2006-01-02 15:04:05"), refreshExpire.Sub(now).Truncate(time.Second))
		fmt.Printf("  User Token:       %s\n", config.Mask(cfg.UserAccessToken))
		return nil
	},
}

func exchangeAndSave(code string) error {
	ctx := context.Background()
	result, err := client.ExchangeToken(ctx, code)
	if err != nil {
		return err
	}

	now := nowUnix()
	cfg, _ := config.Load()
	if cfg == nil {
		cfg = &config.Config{AppID: appID, AppSecret: appSecret}
	}
	cfg.UserAccessToken = result.AccessToken
	cfg.RefreshToken = result.RefreshToken
	cfg.TokenExpiresAt = now + int64(result.ExpiresIn)
	cfg.RefreshExpiresAt = now + int64(result.RefreshExpiresIn)

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("保存 token 失败: %w", err)
	}

	fmt.Println("\n授权成功!")
	fmt.Printf("  Token 有效期至:   %s\n", time.Unix(cfg.TokenExpiresAt, 0).Format("2006-01-02 15:04:05"))
	fmt.Printf("  Refresh 有效期至: %s\n", time.Unix(cfg.RefreshExpiresAt, 0).Format("2006-01-02 15:04:05"))
	return nil
}

func nowUnix() int64 {
	return time.Now().Unix()
}

func openBrowser(rawURL string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	case "darwin":
		cmd = exec.Command("open", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	_ = cmd.Start()
}

func init() {
	authLoginCmd.Flags().BoolVar(&authAll, "all", false, "授予所有权限 (含通讯录、云空间等可选权限)")
	authLoginCmd.Flags().StringVar(&authScopes, "scopes", "", "自定义授权范围 (空格分隔)")
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
	rootCmd.AddCommand(authCmd)
}
