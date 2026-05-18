package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	larkdocx "github.com/larksuite/oapi-sdk-go/v3/service/docx/v1"
	"github.com/spf13/cobra"
)

var (
	getDocID   string
	showBlocks bool
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "获取飞书文档内容",
	Example: `  feishu-doc get --id doxbcGW1HN3Xg5k2M4Fh0Dk1nZb
  feishu-doc get --id doxbcGW1HN3Xg5k2M4Fh0Dk1nZb --blocks`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		doc, err := client.GetDocument(ctx, getDocID)
		if err != nil {
			return err
		}

		fmt.Printf("文档: %s\n", doc.Title)
		fmt.Printf("文档ID: %s\n", doc.DocumentID)
		fmt.Printf("修订版本: %d\n", doc.RevisionID)
		fmt.Println(strings.Repeat("-", 50))

		if showBlocks {
			return showBlockList(ctx, getDocID)
		}
		return showRawContent(ctx, getDocID)
	},
}

func showRawContent(ctx context.Context, docID string) error {
	content, err := client.GetRawContent(ctx, docID)
	if err != nil {
		return err
	}
	fmt.Println(content)
	return nil
}

func showBlockList(ctx context.Context, docID string) error {
	blocks, err := client.FetchAllBlocks(ctx, docID)
	if err != nil {
		return err
	}
	for _, block := range blocks {
		printBlock(block)
	}
	return nil
}

var blockTypeNames = map[int]string{
	1:  "Page",
	2:  "Text",
	3:  "H1",
	4:  "H2",
	5:  "H3",
	6:  "H4",
	7:  "H5",
	8:  "H6",
	9:  "H7",
	10: "H8",
	11: "H9",
	12: "Bullet",
	13: "Ordered",
	14: "Code",
	15: "Quote",
	17: "Todo",
	19: "Callout",
	22: "Divider",
	27: "Image",
	31: "Table",
	32: "Cell",
	40: "AddOns",
}

func printBlock(block *larkdocx.Block) {
	bt := *block.BlockType
	typeName, ok := blockTypeNames[bt]
	if !ok {
		typeName = fmt.Sprintf("Type(%d)", bt)
	}

	bid := *block.BlockId

	if bt == 40 {
		if mermaid := extractMermaid(block); mermaid != "" {
			fmt.Printf("[%-8s] <%s> ```mermaid\n%s\n```\n", "Mermaid", bid, mermaid)
			return
		}
		fmt.Printf("[%-8s] <%s>\n", typeName, bid)
		return
	}

	text := extractBlockText(block)
	if text != "" {
		fmt.Printf("[%-8s] <%s> %s\n", typeName, bid, text)
	} else {
		fmt.Printf("[%-8s] <%s>\n", typeName, bid)
	}
}

func extractBlockText(block *larkdocx.Block) string {
	var elements []*larkdocx.TextElement

	switch *block.BlockType {
	case 2:
		if block.Text != nil {
			elements = block.Text.Elements
		}
	case 3:
		if block.Heading1 != nil {
			elements = block.Heading1.Elements
		}
	case 4:
		if block.Heading2 != nil {
			elements = block.Heading2.Elements
		}
	case 5:
		if block.Heading3 != nil {
			elements = block.Heading3.Elements
		}
	case 6:
		if block.Heading4 != nil {
			elements = block.Heading4.Elements
		}
	case 7:
		if block.Heading5 != nil {
			elements = block.Heading5.Elements
		}
	case 8:
		if block.Heading6 != nil {
			elements = block.Heading6.Elements
		}
	case 9:
		if block.Heading7 != nil {
			elements = block.Heading7.Elements
		}
	case 10:
		if block.Heading8 != nil {
			elements = block.Heading8.Elements
		}
	case 11:
		if block.Heading9 != nil {
			elements = block.Heading9.Elements
		}
	case 12:
		if block.Bullet != nil {
			elements = block.Bullet.Elements
		}
	case 13:
		if block.Ordered != nil {
			elements = block.Ordered.Elements
		}
	case 14:
		if block.Code != nil {
			elements = block.Code.Elements
		}
	case 15:
		if block.Quote != nil {
			elements = block.Quote.Elements
		}
	}

	return joinTextElements(elements)
}

func joinTextElements(elements []*larkdocx.TextElement) string {
	var parts []string
	for _, el := range elements {
		if el.TextRun != nil && el.TextRun.Content != nil {
			parts = append(parts, *el.TextRun.Content)
		}
	}
	return strings.Join(parts, "")
}

func extractMermaid(block *larkdocx.Block) string {
	if block.AddOns == nil || block.AddOns.Record == nil {
		return ""
	}
	var record struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal([]byte(*block.AddOns.Record), &record); err != nil {
		return ""
	}
	return record.Data
}

func init() {
	getCmd.Flags().StringVar(&getDocID, "id", "", "文档 ID (必填)")
	getCmd.Flags().BoolVar(&showBlocks, "blocks", false, "显示文档块结构 (默认显示纯文本)")
	_ = getCmd.MarkFlagRequired("id")
	rootCmd.AddCommand(getCmd)
}
