package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	updateDocID   string
	updateContent string
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "向文档追加文本内容",
	Example: `  feishu-doc update --id doxbcGW1HN3Xg5k2M4Fh0Dk1nZb --content "Hello, World!"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		ids, err := client.AppendText(ctx, updateDocID, updateContent)
		if err != nil {
			return err
		}

		fmt.Println("内容追加成功!")
		for _, id := range ids {
			fmt.Printf("  新块ID: %s\n", id)
		}
		return nil
	},
}

func init() {
	updateCmd.Flags().StringVar(&updateDocID, "id", "", "文档 ID (必填)")
	updateCmd.Flags().StringVar(&updateContent, "content", "", "要追加的文本内容 (必填)")
	_ = updateCmd.MarkFlagRequired("id")
	_ = updateCmd.MarkFlagRequired("content")
	rootCmd.AddCommand(updateCmd)
}
