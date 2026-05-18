package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	larkdocx "github.com/larksuite/oapi-sdk-go/v3/service/docx/v1"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
)

const (
	ExportModeMinimal = "minimal"
)

type ExportOptions struct {
	WithImages bool
	OutputDir  string
	Mode       string
}

type ExportResult struct {
	Title    string
	Markdown string
}

func (c *Client) ExportToMarkdown(ctx context.Context, docID string, opts *ExportOptions) (*ExportResult, error) {
	doc, err := c.GetDocument(ctx, docID)
	if err != nil {
		return nil, err
	}

	blocks, err := c.FetchAllBlocks(ctx, docID)
	if err != nil {
		return nil, err
	}

	var imageURLs map[string]string
	if opts != nil && opts.WithImages && opts.OutputDir != "" {
		imageURLs = c.DownloadImages(ctx, blocks, opts.OutputDir)
	} else {
		imageURLs = c.FetchImageTmpURLs(ctx, blocks)
	}

	userNames := c.resolveUserNames(ctx, blocks)
	md := BlocksToMarkdown(blocks, docID, doc.Title, imageURLs, userNames)

	return &ExportResult{Title: doc.Title, Markdown: md}, nil
}

func (c *Client) resolveUserNames(ctx context.Context, blocks []*larkdocx.Block) map[string]string {
	seen := make(map[string]bool)
	var userIDs []string
	for _, b := range blocks {
		for _, t := range getTextFields(b) {
			if t == nil {
				continue
			}
			for _, el := range t.Elements {
				if el.MentionUser != nil && el.MentionUser.UserId != nil {
					uid := *el.MentionUser.UserId
					if !seen[uid] {
						seen[uid] = true
						userIDs = append(userIDs, uid)
					}
				}
			}
		}
	}
	if len(userIDs) == 0 {
		return nil
	}
	fmt.Fprintf(c.Logger, "  解析 %d 个 @用户...\n", len(userIDs))
	return c.BatchGetUserNames(ctx, userIDs)
}

func getTextFields(b *larkdocx.Block) []*larkdocx.Text {
	return []*larkdocx.Text{
		b.Text, b.Heading1, b.Heading2, b.Heading3,
		b.Heading4, b.Heading5, b.Heading6, b.Heading7,
		b.Heading8, b.Heading9, b.Bullet, b.Ordered,
		b.Code, b.Quote, b.Equation, b.Todo,
	}
}

type exportCtx struct {
	imageURLs map[string]string
	userNames map[string]string
	blockMap  map[string]*larkdocx.Block
	depthMap  map[string]int
}

// R10: minimal 模式不输出自动标题行，文档标题由飞书 Page 块控制
func BlocksToMarkdown(blocks []*larkdocx.Block, docID, title string, imageURLs map[string]string, userNames map[string]string) string {
	ctx := &exportCtx{
		imageURLs: imageURLs,
		userNames: userNames,
		depthMap:  buildDepthMap(blocks, docID),
		blockMap:  make(map[string]*larkdocx.Block, len(blocks)),
	}
	for _, b := range blocks {
		ctx.blockMap[Deref(b.BlockId)] = b
	}

	containerChildren := make(map[string]bool)
	for _, b := range blocks {
		if b.ParentId == nil {
			continue
		}
		parent, ok := ctx.blockMap[*b.ParentId]
		if !ok {
			continue
		}
		pt := DerefInt(parent.BlockType)
		if pt == BlockTypeTable || pt == BlockTypeTableCell ||
			pt == BlockTypeQuoteContainer || pt == BlockTypeCallout {
			containerChildren[Deref(b.BlockId)] = true
		}
	}

	var buf bytes.Buffer

	// R9: 连续空 Text 块合并为单个空行
	prevIsList := false
	prevEmpty := false
	for _, block := range blocks {
		bt := DerefInt(block.BlockType)
		if bt == BlockTypePage {
			continue
		}
		if containerChildren[Deref(block.BlockId)] {
			continue
		}
		isList := bt == BlockTypeBullet || bt == BlockTypeOrdered || bt == BlockTypeTodo
		if prevIsList && !isList {
			buf.WriteString("\n")
		}
		prevIsList = isList

		md := ctx.blockToMarkdown(block, bt)

		isEmpty := bt == BlockTypeText && md == "\n"
		if isEmpty {
			if prevEmpty {
				continue
			}
			prevEmpty = true
			continue
		}
		prevEmpty = false

		if md != "" {
			buf.WriteString(md)
		}
	}

	result := buf.String()
	result = strings.TrimRight(result, "\n")
	if result != "" {
		result += "\n"
	}
	return result
}

func (c *Client) FetchImageTmpURLs(ctx context.Context, blocks []*larkdocx.Block) map[string]string {
	tokens := collectImageTokens(blocks)
	if len(tokens) == 0 {
		return nil
	}

	result := make(map[string]string)
	for i := 0; i < len(tokens); i += 50 {
		end := i + 50
		if end > len(tokens) {
			end = len(tokens)
		}
		req := larkdrive.NewBatchGetTmpDownloadUrlMediaReqBuilder().
			FileTokens(tokens[i:end]).
			Build()

		resp, err := c.larkClient.Drive.Media.BatchGetTmpDownloadUrl(ctx, req, c.RequestOpts()...)
		if err != nil || !resp.Success() {
			continue
		}
		for _, item := range resp.Data.TmpDownloadUrls {
			if item.FileToken != nil && item.TmpDownloadUrl != nil {
				result[*item.FileToken] = *item.TmpDownloadUrl
			}
		}
	}
	return result
}

func (c *Client) DownloadImages(ctx context.Context, blocks []*larkdocx.Block, outputDir string) map[string]string {
	tokens := collectImageTokens(blocks)
	if len(tokens) == 0 {
		return nil
	}

	imgDir := filepath.Join(outputDir, "images")
	if err := os.MkdirAll(imgDir, 0755); err != nil {
		fmt.Fprintf(c.Logger, "  [warn] 创建图片目录失败: %v\n", err)
		return nil
	}

	result := make(map[string]string)
	for _, token := range tokens {
		req := larkdrive.NewDownloadMediaReqBuilder().
			FileToken(token).
			Build()

		resp, err := c.larkClient.Drive.Media.Download(ctx, req, c.RequestOpts()...)
		if err != nil || !resp.Success() {
			fmt.Fprintf(c.Logger, "  [warn] 下载图片失败: %s\n", token)
			continue
		}

		fileName := resp.FileName
		if fileName == "" {
			fileName = token + ".png"
		}
		if err := saveImageFile(filepath.Join(imgDir, fileName), resp.File); err != nil {
			fmt.Fprintf(c.Logger, "  [warn] 保存图片失败: %s: %v\n", fileName, err)
			continue
		}

		result[token] = "images/" + fileName
		fmt.Fprintf(c.Logger, "  图片已下载: %s\n", fileName)
	}
	return result
}

func saveImageFile(path string, r io.Reader) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func buildDepthMap(blocks []*larkdocx.Block, docID string) map[string]int {
	dm := map[string]int{docID: -1}
	for _, b := range blocks {
		pid := Deref(b.ParentId)
		parentDepth, ok := dm[pid]
		if !ok {
			parentDepth = -1
		}
		dm[Deref(b.BlockId)] = parentDepth + 1
	}
	return dm
}

func (ec *exportCtx) blockToMarkdown(block *larkdocx.Block, bt int) string {
	bid := Deref(block.BlockId)

	switch bt {
	case BlockTypeText:
		text := ec.renderInline(block.Text)
		if text == "" {
			return "\n"
		}
		return text + "\n\n"

	case BlockTypeHeading1, BlockTypeHeading2, BlockTypeHeading3,
		BlockTypeHeading4, BlockTypeHeading5, BlockTypeHeading6,
		BlockTypeHeading7, BlockTypeHeading8, BlockTypeHeading9:
		level := bt - BlockTypeText
		if level > 6 {
			level = 6
		}
		var t *larkdocx.Text
		switch bt {
		case BlockTypeHeading1:
			t = block.Heading1
		case BlockTypeHeading2:
			t = block.Heading2
		case BlockTypeHeading3:
			t = block.Heading3
		case BlockTypeHeading4:
			t = block.Heading4
		case BlockTypeHeading5:
			t = block.Heading5
		case BlockTypeHeading6:
			t = block.Heading6
		case BlockTypeHeading7:
			t = block.Heading7
		case BlockTypeHeading8:
			t = block.Heading8
		case BlockTypeHeading9:
			t = block.Heading9
		}
		return strings.Repeat("#", level) + " " + ec.renderInline(t) + "\n\n"

	case BlockTypeBullet:
		depth := 0
		if d, ok := ec.depthMap[bid]; ok && d > 0 {
			depth = d - 1
		}
		return strings.Repeat("  ", depth) + "- " + ec.renderInline(block.Bullet) + "\n"

	// R6: 有序列表缩进从 2 空格改为 3 空格
	case BlockTypeOrdered:
		depth := 0
		if d, ok := ec.depthMap[bid]; ok && d > 0 {
			depth = d - 1
		}
		return strings.Repeat("   ", depth) + "1. " + ec.renderInline(block.Ordered) + "\n"

	case BlockTypeCode:
		lang := ""
		content := ""
		if block.Code != nil {
			if block.Code.Style != nil && block.Code.Style.Language != nil {
				lang = reverseLangMap(*block.Code.Style.Language)
			}
			content = ec.renderPlainText(block.Code)
		}
		return "```" + lang + "\n" + content + "\n```\n\n"

	// R2: 引用块多行时每行加 > 前缀
	case BlockTypeQuote:
		text := ec.renderInline(block.Quote)
		lines := strings.Split(text, "\n")
		var buf bytes.Buffer
		for i, line := range lines {
			buf.WriteString("> " + line)
			if i < len(lines)-1 {
				buf.WriteString("\n")
			}
		}
		buf.WriteString("\n\n")
		return buf.String()

	case BlockTypeTodo:
		check := "[ ]"
		if block.Todo != nil && block.Todo.Style != nil && DerefBool(block.Todo.Style.Done) {
			check = "[x]"
		}
		depth := 0
		if d, ok := ec.depthMap[bid]; ok && d > 0 {
			depth = d - 1
		}
		return strings.Repeat("  ", depth) + "- " + check + " " + ec.renderInline(block.Todo) + "\n"

	case BlockTypeEquation:
		if block.Equation != nil {
			content := ec.renderPlainText(block.Equation)
			if content != "" {
				return "$$\n" + content + "\n$$\n\n"
			}
		}
		return ""

	case BlockTypeCallout:
		return ec.renderCallout(block)

	case BlockTypeQuoteContainer:
		return ec.renderQuoteContainer(block)

	case BlockTypeImage:
		if block.Image != nil && block.Image.Token != nil {
			imgRef := ec.imageURLs[*block.Image.Token]
			if imgRef == "" {
				imgRef = "feishu://image/" + *block.Image.Token
			}
			return "![](" + imgRef + ")\n\n"
		}
		return ""

	case BlockTypeTable:
		return ec.renderTable(block)

	case BlockTypeTableCell:
		return ""

	case BlockTypeDivider:
		return "---\n\n"

	case BlockTypeAddOns:
		if block.AddOns != nil && block.AddOns.Record != nil {
			var record struct {
				Data string `json:"data"`
			}
			if json.Unmarshal([]byte(*block.AddOns.Record), &record) == nil && record.Data != "" {
				return "```mermaid\n" + record.Data + "\n```\n\n"
			}
		}
		return ""

	default:
		typeName := blockTypePlaceholder(bt)
		return fmt.Sprintf("<!-- feishu:id=%s,type=%s -->\n\n", bid, typeName)
	}
}

// R5: Markdown 特殊字符转义补全
func escapeMd(s string) string {
	var buf strings.Builder
	buf.Grow(len(s))
	for i, r := range s {
		switch r {
		case '*':
			buf.WriteString(`\*`)
		case '_':
			buf.WriteString(`\_`)
		case '~':
			buf.WriteString(`\~`)
		case '`':
			buf.WriteString("\\`")
		case '[':
			buf.WriteString(`\[`)
		case ']':
			buf.WriteString(`\]`)
		case '|':
			buf.WriteString(`\|`)
		case '#':
			if i == 0 || (i > 0 && s[i-1] == '\n') {
				buf.WriteString(`\#`)
			} else {
				buf.WriteRune(r)
			}
		case '>':
			if i == 0 || (i > 0 && s[i-1] == '\n') {
				buf.WriteString(`\>`)
			} else {
				buf.WriteRune(r)
			}
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

func (ec *exportCtx) renderInline(t *larkdocx.Text) string {
	if t == nil {
		return ""
	}
	var buf bytes.Buffer
	for _, el := range t.Elements {
		if el.Equation != nil && el.Equation.Content != nil {
			buf.WriteString("$" + strings.TrimSpace(*el.Equation.Content) + "$")
			continue
		}
		if el.MentionUser != nil && el.MentionUser.UserId != nil {
			uid := *el.MentionUser.UserId
			name := ec.userNames[uid]
			if name != "" {
				buf.WriteString("@" + name + "<!-- uid:" + uid + " -->")
			} else {
				buf.WriteString("@" + uid)
			}
			continue
		}
		if el.MentionDoc != nil {
			title := Deref(el.MentionDoc.Title)
			url := Deref(el.MentionDoc.Url)
			if title == "" {
				title = Deref(el.MentionDoc.Token)
			}
			if url != "" {
				buf.WriteString("[" + title + "](" + url + ")")
			} else {
				buf.WriteString(title)
			}
			continue
		}
		if el.TextRun == nil {
			continue
		}
		content := Deref(el.TextRun.Content)
		if content == "" {
			continue
		}

		s := el.TextRun.TextElementStyle
		if s == nil {
			buf.WriteString(escapeMd(content))
			continue
		}

		isCode := DerefBool(s.InlineCode)
		isBold := DerefBool(s.Bold)
		isItalic := DerefBool(s.Italic)
		isStrike := DerefBool(s.Strikethrough)
		hasLink := s.Link != nil && Deref(s.Link.Url) != ""

		if isCode {
			content = "`" + content + "`"
		} else {
			content = escapeMd(content)
			if isBold && isItalic {
				content = "***" + content + "***"
			} else if isBold {
				content = "**" + content + "**"
			} else if isItalic {
				content = "*" + content + "*"
			}
			if isStrike {
				content = "~~" + content + "~~"
			}
		}
		if hasLink {
			content = "[" + content + "](" + *s.Link.Url + ")"
		}

		buf.WriteString(content)
	}
	return buf.String()
}

// R7: 代码块末尾只去除最后一个 \n，不用 TrimRight 去除所有尾部换行
func (ec *exportCtx) renderPlainText(t *larkdocx.Text) string {
	if t == nil {
		return ""
	}
	var buf bytes.Buffer
	for _, el := range t.Elements {
		if el.TextRun != nil {
			buf.WriteString(Deref(el.TextRun.Content))
		}
	}
	s := buf.String()
	if strings.HasSuffix(s, "\n") {
		s = s[:len(s)-1]
	}
	return s
}

func (ec *exportCtx) renderTable(block *larkdocx.Block) string {
	if block.Table == nil || block.Table.Property == nil {
		return ""
	}
	rows := DerefInt(block.Table.Property.RowSize)
	cols := DerefInt(block.Table.Property.ColumnSize)
	if rows == 0 || cols == 0 {
		return ""
	}
	cellIDs := block.Table.Cells

	grid := make([][]string, rows)
	for r := 0; r < rows; r++ {
		grid[r] = make([]string, cols)
		for c := 0; c < cols; c++ {
			idx := r*cols + c
			if idx >= len(cellIDs) {
				continue
			}
			cellBlock, ok := ec.blockMap[cellIDs[idx]]
			if !ok {
				continue
			}
			if cellBlock.Children != nil {
				var parts []string
				for _, childID := range cellBlock.Children {
					if child, ok := ec.blockMap[childID]; ok {
						if t := ec.cellBlockText(child); t != "" {
							parts = append(parts, t)
						}
					}
				}
				grid[r][c] = strings.Join(parts, " ")
			}
		}
	}

	var buf bytes.Buffer
	buf.WriteString("|")
	for c := 0; c < cols; c++ {
		buf.WriteString(" " + grid[0][c] + " |")
	}
	buf.WriteString("\n|")
	for c := 0; c < cols; c++ {
		buf.WriteString(" --- |")
	}
	buf.WriteString("\n")
	for r := 1; r < rows; r++ {
		buf.WriteString("|")
		for c := 0; c < cols; c++ {
			buf.WriteString(" " + grid[r][c] + " |")
		}
		buf.WriteString("\n")
	}
	buf.WriteString("\n")
	return buf.String()
}

func (ec *exportCtx) renderCallout(block *larkdocx.Block) string {
	admonition := "NOTE"
	if block.Callout != nil && block.Callout.EmojiId != nil {
		switch *block.Callout.EmojiId {
		case "warning", "⚠️":
			admonition = "WARNING"
		case "bulb", "💡":
			admonition = "TIP"
		case "exclamation", "❗":
			admonition = "IMPORTANT"
		case "red_circle", "🔴", "no_entry", "🚫":
			admonition = "CAUTION"
		}
	}

	var lines []string
	lines = append(lines, "> [!"+admonition+"]")
	for _, childID := range block.Children {
		child, ok := ec.blockMap[childID]
		if !ok {
			continue
		}
		ct := DerefInt(child.BlockType)
		text := ec.childBlockInlineText(child, ct)
		lines = append(lines, "> "+text)
	}
	return strings.Join(lines, "\n") + "\n\n"
}

func (ec *exportCtx) renderQuoteContainer(block *larkdocx.Block) string {
	var lines []string
	for _, childID := range block.Children {
		child, ok := ec.blockMap[childID]
		if !ok {
			continue
		}
		ct := DerefInt(child.BlockType)
		text := ec.childBlockInlineText(child, ct)
		if text == "" && ct == BlockTypeText {
			lines = append(lines, ">")
		} else {
			lines = append(lines, "> "+text)
		}
	}
	return strings.Join(lines, "\n") + "\n\n"
}

func (ec *exportCtx) childBlockInlineText(block *larkdocx.Block, bt int) string {
	switch bt {
	case BlockTypeText:
		return ec.renderInline(block.Text)
	case BlockTypeHeading1, BlockTypeHeading2, BlockTypeHeading3,
		BlockTypeHeading4, BlockTypeHeading5, BlockTypeHeading6,
		BlockTypeHeading7, BlockTypeHeading8, BlockTypeHeading9:
		level := bt - BlockTypeText
		if level > 6 {
			level = 6
		}
		var t *larkdocx.Text
		switch bt {
		case BlockTypeHeading1:
			t = block.Heading1
		case BlockTypeHeading2:
			t = block.Heading2
		case BlockTypeHeading3:
			t = block.Heading3
		case BlockTypeHeading4:
			t = block.Heading4
		case BlockTypeHeading5:
			t = block.Heading5
		case BlockTypeHeading6:
			t = block.Heading6
		case BlockTypeHeading7:
			t = block.Heading7
		case BlockTypeHeading8:
			t = block.Heading8
		case BlockTypeHeading9:
			t = block.Heading9
		}
		return strings.Repeat("#", level) + " " + ec.renderInline(t)
	case BlockTypeBullet:
		return "- " + ec.renderInline(block.Bullet)
	case BlockTypeOrdered:
		return "1. " + ec.renderInline(block.Ordered)
	case BlockTypeTodo:
		check := "[ ]"
		if block.Todo != nil && block.Todo.Style != nil && DerefBool(block.Todo.Style.Done) {
			check = "[x]"
		}
		return "- " + check + " " + ec.renderInline(block.Todo)
	case BlockTypeCode:
		lang := ""
		if block.Code != nil && block.Code.Style != nil && block.Code.Style.Language != nil {
			lang = reverseLangMap(*block.Code.Style.Language)
		}
		return "```" + lang + "\n" + ec.renderPlainText(block.Code) + "\n```"
	case BlockTypeDivider:
		return "---"
	case BlockTypeImage:
		if block.Image != nil && block.Image.Token != nil {
			ref := ec.imageURLs[*block.Image.Token]
			if ref == "" {
				ref = "feishu://image/" + *block.Image.Token
			}
			return "![](" + ref + ")"
		}
		return ""
	default:
		return ec.renderInline(block.Text)
	}
}

// R3: 表格单元格复用 renderInline 处理 Text 块的行内样式
func (ec *exportCtx) cellBlockText(block *larkdocx.Block) string {
	if block.BlockType == nil {
		return ""
	}
	switch *block.BlockType {
	case BlockTypeText:
		return ec.renderInline(block.Text)
	default:
		return ec.renderPlainText(block.Text)
	}
}

func displayWidth(s string) int {
	n := 0
	for _, r := range s {
		if r > 0x7F {
			n += 2
		} else {
			n++
		}
	}
	return n
}

func padRight(s string, width int) string {
	gap := width - displayWidth(s)
	if gap <= 0 {
		return s
	}
	return s + strings.Repeat(" ", gap)
}

func collectImageTokens(blocks []*larkdocx.Block) []string {
	var tokens []string
	for _, b := range blocks {
		if DerefInt(b.BlockType) == BlockTypeImage && b.Image != nil && b.Image.Token != nil {
			tokens = append(tokens, *b.Image.Token)
		}
	}
	return tokens
}

var blockTypeSlug = map[int]string{
	1: "page", 2: "text",
	3: "heading1", 4: "heading2", 5: "heading3",
	6: "heading4", 7: "heading5", 8: "heading6",
	9: "heading7", 10: "heading8", 11: "heading9",
	12: "bullet", 13: "ordered", 14: "code",
	15: "quote", 16: "equation", 17: "todo",
	18: "bitable", 19: "callout", 20: "chat_card", 21: "diagram",
	22: "divider", 23: "file", 24: "grid", 25: "grid_column",
	26: "iframe", 27: "image", 28: "isv", 29: "mindnote",
	30: "sheet", 31: "table", 32: "table_cell",
	33: "view", 34: "quote_container", 35: "task",
	36: "okr", 40: "add_ons", 41: "jira_issue", 43: "board",
	44: "agenda", 48: "link_preview",
}

func blockTypePlaceholder(bt int) string {
	if s, ok := blockTypeSlug[bt]; ok {
		return s
	}
	return fmt.Sprintf("unknown_%d", bt)
}

var reverseLang = map[int]string{
	1: "", 7: "bash", 8: "csharp", 9: "cpp", 10: "c",
	12: "css", 15: "dart", 18: "dockerfile",
	22: "go", 24: "html", 26: "http", 28: "json", 29: "java",
	30: "javascript", 32: "kotlin", 33: "latex", 36: "lua",
	38: "makefile", 39: "markdown", 40: "nginx",
	43: "php", 44: "perl", 46: "powershell", 48: "protobuf",
	49: "python", 50: "r", 52: "ruby", 53: "rust",
	55: "scss", 56: "sql", 57: "scala", 60: "shell",
	61: "swift", 63: "typescript", 66: "xml", 67: "yaml",
	68: "cmake", 69: "diff", 71: "graphql", 73: "properties",
	74: "solidity", 75: "toml",
}

func reverseLangMap(code int) string {
	if name, ok := reverseLang[code]; ok {
		return name
	}
	return ""
}
