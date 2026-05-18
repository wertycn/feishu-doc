package cli

import (
	"context"
	"fmt"

	"github.com/wertycn/feishu-doc/core"
	"github.com/spf13/cobra"
)

var (
	copyDocID  string
	copyName   string
	copyFolder string
	copyParent string
)

var copyCmd = &cobra.Command{
	Use:   "copy",
	Short: "创建飞书文档副本",
	Long: `复制飞书文档，支持两种模式:

  1. 知识库节点复制: --id <node_token> --parent <目标父节点> [--name <新标题>]
  2. 云空间文件复制: --id <doc_id> --folder <目标文件夹> --name <新名称>`,
	Example: `  feishu-doc copy --id NxXNwSzDRiJCd0kmLYCcrKK2n6b --parent HSRtwp75sihBMBk70PfcuDQonDh --name "副本"
  feishu-doc copy --id GfOvdqk7Oooul9x3UeHcGoqLnbd --folder fldbcXXX --name "副本"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		if copyParent != "" {
			return copyAsWikiNode(ctx)
		}
		if copyFolder != "" {
			return copyAsDriveFile(ctx)
		}
		return fmt.Errorf("请指定 --parent (知识库节点复制) 或 --folder (云空间文件复制)")
	},
}

func copyAsWikiNode(ctx context.Context) error {
	node, err := client.GetNodeInfo(ctx, copyDocID)
	if err != nil {
		return fmt.Errorf("查询源节点失败: %w", err)
	}
	spaceID := core.Deref(node.SpaceId)
	fmt.Printf("源节点: %s (Space: %s)\n", core.Deref(node.Title), spaceID)

	newNode, err := client.CopyWikiNode(ctx, spaceID, copyDocID, copyParent, "", copyName)
	if err != nil {
		return err
	}

	fmt.Println("\n复制成功!")
	fmt.Printf("  标题:        %s\n", core.Deref(newNode.Title))
	fmt.Printf("  Node Token:  %s\n", core.Deref(newNode.NodeToken))
	fmt.Printf("  Obj Token:   %s\n", core.Deref(newNode.ObjToken))
	if url := client.WikiNodeURL(core.Deref(newNode.NodeToken)); url != "" {
		fmt.Printf("  URL:         %s\n", url)
	}
	return nil
}

func copyAsDriveFile(ctx context.Context) error {
	if copyName == "" {
		return fmt.Errorf("云空间复制需要指定 --name")
	}
	result, err := client.CopyFile(ctx, copyDocID, copyName, "docx", copyFolder)
	if err != nil {
		return err
	}

	fmt.Println("复制成功!")
	fmt.Printf("  名称:     %s\n", result.Name)
	fmt.Printf("  Token:    %s\n", result.Token)
	if result.URL != "" {
		fmt.Printf("  URL:      %s\n", result.URL)
	} else if url := client.DocURL(result.Token); url != "" {
		fmt.Printf("  URL:      %s\n", url)
	}
	return nil
}

func init() {
	copyCmd.Flags().StringVar(&copyDocID, "id", "", "源文档 ID 或 Node Token (必填)")
	copyCmd.Flags().StringVar(&copyName, "name", "", "副本名称/标题 (知识库复制可选，云空间复制必填)")
	copyCmd.Flags().StringVar(&copyParent, "parent", "", "目标父节点 token (知识库节点复制)")
	copyCmd.Flags().StringVar(&copyFolder, "folder", "", "目标文件夹 token (云空间文件复制)")
	_ = copyCmd.MarkFlagRequired("id")
	rootCmd.AddCommand(copyCmd)
}
