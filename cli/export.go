package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/debugicu/feishu-doc/core"

	"github.com/spf13/cobra"
)

var (
	exportDocID    string
	exportFile     string
	exportWithImgs bool
	exportMode     string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "将飞书文档导出为 Markdown 文件",
	Example: `  feishu-doc export --id GfOvdqk7Oooul9x3UeHcGoqLnbd --file output.md
  feishu-doc export --id GfOvdqk7Oooul9x3UeHcGoqLnbd   # 输出到终端`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		mode := exportMode
		if mode == "" {
			mode = core.ExportModeMinimal
		}

		opts := &core.ExportOptions{
			WithImages: exportWithImgs,
			Mode:       mode,
		}
		if exportFile != "" {
			opts.OutputDir = filepath.Dir(exportFile)
		}

		result, err := client.ExportToMarkdown(ctx, exportDocID, opts)
		if err != nil {
			return err
		}

		if exportFile != "" {
			if err := os.WriteFile(exportFile, []byte(result.Markdown), 0644); err != nil {
				return fmt.Errorf("写入文件失败: %w", err)
			}
			fmt.Printf("已导出到 %s (%d 字节)\n", exportFile, len(result.Markdown))
			if url := client.DocURL(exportDocID); url != "" {
				fmt.Printf("源文档: %s\n", url)
			}
		} else {
			fmt.Print(result.Markdown)
		}
		return nil
	},
}

func init() {
	exportCmd.Flags().StringVar(&exportDocID, "id", "", "文档 ID (必填)")
	exportCmd.Flags().StringVar(&exportFile, "file", "", "输出文件路径 (不指定则输出到终端)")
	exportCmd.Flags().BoolVar(&exportWithImgs, "with-images", false, "下载图片到本地 images/ 目录 (默认使用24h临时链接)")
	exportCmd.Flags().StringVar(&exportMode, "mode", "minimal", "导出模式: minimal(纯净，无 front matter)")
	_ = exportCmd.MarkFlagRequired("id")
	rootCmd.AddCommand(exportCmd)
}
