package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	createTitle  string
	createFolder string
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "创建飞书文档",
	Example: `  feishu-doc create --title "测试文档"
  feishu-doc create --title "测试文档" --folder fldbcO1UuPz8VwnpR4EB2Kh0nXe`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		doc, err := client.CreateDocument(ctx, createTitle, createFolder)
		if err != nil {
			return err
		}

		fmt.Println("文档创建成功!")
		fmt.Printf("  文档ID:   %s\n", doc.DocumentID)
		fmt.Printf("  标题:     %s\n", doc.Title)
		fmt.Printf("  修订版本: %d\n", doc.RevisionID)
		if url := client.DocURL(doc.DocumentID); url != "" {
			fmt.Printf("  URL:      %s\n", url)
		}
		return nil
	},
}

func init() {
	createCmd.Flags().StringVar(&createTitle, "title", "", "文档标题 (必填)")
	createCmd.Flags().StringVar(&createFolder, "folder", "", "目标文件夹 token (可选)")
	_ = createCmd.MarkFlagRequired("title")
	rootCmd.AddCommand(createCmd)
}
