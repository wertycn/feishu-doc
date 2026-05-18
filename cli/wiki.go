package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/wertycn/feishu-doc/core"

	larkwiki "github.com/larksuite/oapi-sdk-go/v3/service/wiki/v2"
	"github.com/spf13/cobra"
)

var wikiCmd = &cobra.Command{
	Use:   "wiki",
	Short: "知识库文档树操作",
}

var wikiSpacesCmd = &cobra.Command{
	Use:     "spaces",
	Short:   "列出知识库列表",
	Example: `  feishu-doc wiki spaces`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		spaces, err := client.ListSpaces(ctx)
		if err != nil {
			return err
		}

		if len(spaces) == 0 {
			fmt.Println("(暂无知识库)")
			return nil
		}

		fmt.Printf("%-30s %-20s %s\n", "名称", "Space ID", "描述")
		fmt.Println(strings.Repeat("-", 80))
		for _, s := range spaces {
			name := core.Deref(s.Name)
			desc := core.Deref(s.Description)
			if len(desc) > 30 {
				desc = desc[:28] + ".."
			}
			fmt.Printf("%-28s %-20s %s\n", name, core.Deref(s.SpaceId), desc)
		}
		fmt.Printf("\n共 %d 个知识库\n", len(spaces))
		return nil
	},
}

var infoNodeToken string

var wikiInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "查看知识库节点信息",
	Example: `  feishu-doc wiki info --token JbGFweVktig6CykG0SQcJniunNb`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		n, err := client.GetNodeInfo(ctx, infoNodeToken)
		if err != nil {
			return err
		}

		fmt.Printf("节点信息:\n")
		fmt.Printf("  标题:         %s\n", core.Deref(n.Title))
		fmt.Printf("  Space ID:     %s\n", core.Deref(n.SpaceId))
		fmt.Printf("  Node Token:   %s\n", core.Deref(n.NodeToken))
		fmt.Printf("  Obj Token:    %s\n", core.Deref(n.ObjToken))
		fmt.Printf("  Obj Type:     %s\n", core.Deref(n.ObjType))
		fmt.Printf("  Parent Node:  %s\n", core.Deref(n.ParentNodeToken))
		if n.HasChild != nil {
			fmt.Printf("  有子节点:     %v\n", *n.HasChild)
		}
		if n.NodeType != nil {
			nt := "原始节点"
			if *n.NodeType == "1" {
				nt = "快捷方式"
			}
			fmt.Printf("  节点类型:     %s\n", nt)
		}
		return nil
	},
}

var (
	createNodeParent string
	createNodeSpace  string
	createNodeTitle  string
)

var wikiCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "在知识库中创建子文档",
	Long: `在指定节点下创建子文档。

如果只提供 --parent，会自动查询其所属知识库。
需要权限: wiki:wiki`,
	Example: `  feishu-doc wiki create --parent JbGFweVktig6CykG0SQcJniunNb --title "新文档"
  feishu-doc wiki create --space 7xxx --parent JbGFweVktig6CykG0SQcJniunNb --title "新文档"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		spaceID := createNodeSpace
		if spaceID == "" {
			fmt.Println("正在查询父节点所属知识库...")
			node, err := client.GetNodeInfo(ctx, createNodeParent)
			if err != nil {
				return err
			}
			spaceID = core.Deref(node.SpaceId)
			fmt.Printf("父节点: %s (Space: %s)\n", core.Deref(node.Title), spaceID)
		}

		n, err := client.CreateWikiChild(ctx, spaceID, createNodeParent, createNodeTitle)
		if err != nil {
			return err
		}

		fmt.Println("\n子文档创建成功!")
		fmt.Printf("  标题:        %s\n", core.Deref(n.Title))
		fmt.Printf("  Node Token:  %s\n", core.Deref(n.NodeToken))
		fmt.Printf("  Obj Token:   %s\n", core.Deref(n.ObjToken))
		fmt.Printf("  Obj Type:    %s\n", core.Deref(n.ObjType))
		if url := client.WikiNodeURL(core.Deref(n.NodeToken)); url != "" {
			fmt.Printf("  URL:         %s\n", url)
		}
		return nil
	},
}

var (
	treeSpaceID string
	treeNodeID  string
	treeDepth   int
)

var wikiTreeCmd = &cobra.Command{
	Use:   "tree",
	Short: "显示知识库文档树",
	Example: `  feishu-doc wiki tree --space 7xxxxxxxxxxxx
  feishu-doc wiki tree --space 7xxxxxxxxxxxx --node NodeTokenHere
  feishu-doc wiki tree --space 7xxxxxxxxxxxx --depth 2`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		if treeNodeID != "" {
			fmt.Printf("节点: %s\n", treeNodeID)
		} else {
			fmt.Printf("知识库: %s (根节点)\n", treeSpaceID)
		}
		fmt.Println(strings.Repeat("-", 60))

		nodes, err := client.ListChildren(ctx, treeSpaceID, treeNodeID)
		if err != nil {
			return err
		}
		if len(nodes) == 0 {
			fmt.Println("(空)")
			return nil
		}
		renderTree(ctx, treeSpaceID, nodes, "", 1, treeDepth)
		return nil
	},
}

func renderTree(ctx context.Context, spaceID string, nodes []*larkwiki.Node, prefix string, depth, maxDepth int) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1

		connector := "├── "
		if isLast {
			connector = "└── "
		}

		title := core.Deref(node.Title)
		objType := core.Deref(node.ObjType)
		nodeToken := core.Deref(node.NodeToken)
		hasChild := node.HasChild != nil && *node.HasChild

		shortcut := ""
		if node.NodeType != nil && *node.NodeType == "1" {
			shortcut = " [快捷方式]"
		}

		childMark := ""
		if hasChild && (maxDepth > 0 && depth >= maxDepth) {
			childMark = " ..."
		}

		fmt.Printf("%s%s%-30s [%-5s] %s%s%s\n",
			prefix, connector, title, objType, nodeToken, shortcut, childMark)

		if hasChild && (maxDepth <= 0 || depth < maxDepth) {
			childPrefix := prefix + "│   "
			if isLast {
				childPrefix = prefix + "    "
			}
			children, err := client.ListChildren(ctx, spaceID, core.Deref(node.NodeToken))
			if err != nil {
				fmt.Printf("%s    (获取子节点失败: %v)\n", prefix, err)
				continue
			}
			renderTree(ctx, spaceID, children, childPrefix, depth+1, maxDepth)
		}
	}
}

func init() {
	wikiInfoCmd.Flags().StringVar(&infoNodeToken, "token", "", "节点 token (必填)")
	_ = wikiInfoCmd.MarkFlagRequired("token")

	wikiCreateCmd.Flags().StringVar(&createNodeParent, "parent", "", "父节点 token (必填)")
	wikiCreateCmd.Flags().StringVar(&createNodeSpace, "space", "", "知识库 Space ID (可选，不填则自动查询)")
	wikiCreateCmd.Flags().StringVar(&createNodeTitle, "title", "", "文档标题 (必填)")
	_ = wikiCreateCmd.MarkFlagRequired("parent")
	_ = wikiCreateCmd.MarkFlagRequired("title")

	wikiTreeCmd.Flags().StringVar(&treeSpaceID, "space", "", "知识库 Space ID (必填)")
	wikiTreeCmd.Flags().StringVar(&treeNodeID, "node", "", "起始节点 token (默认从根节点开始)")
	wikiTreeCmd.Flags().IntVar(&treeDepth, "depth", 0, "递归深度限制 (0=不限制)")
	_ = wikiTreeCmd.MarkFlagRequired("space")

	wikiCmd.AddCommand(wikiSpacesCmd)
	wikiCmd.AddCommand(wikiInfoCmd)
	wikiCmd.AddCommand(wikiCreateCmd)
	wikiCmd.AddCommand(wikiTreeCmd)
	rootCmd.AddCommand(wikiCmd)
}
