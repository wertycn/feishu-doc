package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/debugicu/feishu-doc/core"

	"github.com/spf13/cobra"
)

var (
	importFile      string
	importDocID     string
	importParent    string
	importTitle     string
	importOverwrite bool
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "将本地 Markdown 文件写入飞书文档",
	Long: `解析 Markdown 文件并转换为飞书文档块，支持两种模式:

  1. 追加到已有文档: --file <path> --id <doc_id>
  2. 全量覆盖: --file <path> --id <doc_id> --overwrite
  3. 在知识库节点下新建子文档: --file <path> --parent <node_token> [--title <标题>]`,
	Example: `  feishu-doc import --file readme.md --id FbredV7x2oJz3LxtK02cyfk1nld
  feishu-doc import --file readme.md --id FbredV7x2oJz3LxtK02cyfk1nld --overwrite
  feishu-doc import --file spec.md --parent JbGFweVktig6CykG0SQcJniunNb --title "技术方案"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		source, err := os.ReadFile(importFile)
		if err != nil {
			return fmt.Errorf("读取文件失败: %w", err)
		}

		ctx := context.Background()
		docID := importDocID

		if importParent != "" {
			if importOverwrite {
				return fmt.Errorf("--overwrite 不能与 --parent 同时使用")
			}
			node, err := client.GetNodeInfo(ctx, importParent)
			if err != nil {
				return err
			}
			spaceID := core.Deref(node.SpaceId)
			fmt.Printf("父节点: %s (Space: %s)\n", core.Deref(node.Title), spaceID)

			title := importTitle
			if title == "" {
				title = "Imported Document"
			}

			created, err := client.CreateWikiChild(ctx, spaceID, importParent, title)
			if err != nil {
				return err
			}
			docID = core.Deref(created.ObjToken)
			fmt.Printf("子文档已创建: %s (ObjToken: %s)\n", title, docID)
		}

		if docID == "" {
			return fmt.Errorf("请指定 --id (已有文档) 或 --parent (知识库节点)")
		}

		if importOverwrite {
			fmt.Println("全量覆盖模式: 清空文档后重建所有块")
			if err := client.OverwriteMarkdown(ctx, docID, source, importFile); err != nil {
				return err
			}
		} else {
			if err := client.ImportMarkdown(ctx, docID, source, importFile); err != nil {
				return err
			}
		}

		fmt.Printf("\n导入成功! 文档ID: %s\n", docID)
		if url := client.DocURL(docID); url != "" {
			fmt.Printf("URL: %s\n", url)
		}
		return nil
	},
}

func init() {
	importCmd.Flags().StringVar(&importFile, "file", "", "Markdown 文件路径 (必填)")
	importCmd.Flags().StringVar(&importDocID, "id", "", "目标文档 ID (追加/覆盖模式)")
	importCmd.Flags().StringVar(&importParent, "parent", "", "父节点 token (知识库创建模式)")
	importCmd.Flags().StringVar(&importTitle, "title", "", "新文档标题 (配合 --parent 使用)")
	importCmd.Flags().BoolVar(&importOverwrite, "overwrite", false, "全量覆盖模式: 先删除所有现有块再创建新块")
	_ = importCmd.MarkFlagRequired("file")
	rootCmd.AddCommand(importCmd)
}
