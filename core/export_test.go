package core

import (
	"strings"
	"testing"

	larkdocx "github.com/larksuite/oapi-sdk-go/v3/service/docx/v1"
)

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
func boolPtr(b bool) *bool    { return &b }

func makeTextBlock(id, content string) *larkdocx.Block {
	return &larkdocx.Block{
		BlockId:   strPtr(id),
		BlockType: intPtr(BlockTypeText),
		ParentId:  strPtr("doc1"),
		Text: &larkdocx.Text{
			Elements: []*larkdocx.TextElement{
				{TextRun: &larkdocx.TextRun{Content: strPtr(content)}},
			},
		},
	}
}

func makePageBlock(id string) *larkdocx.Block {
	return &larkdocx.Block{
		BlockId:   strPtr(id),
		BlockType: intPtr(BlockTypePage),
	}
}

func TestBlocksToMarkdown_TextBlock(t *testing.T) {
	blocks := []*larkdocx.Block{
		makePageBlock("doc1"),
		makeTextBlock("b1", "Hello world"),
	}
	result := BlocksToMarkdown(blocks, "doc1", "Test", nil, nil)
	if !strings.Contains(result, "Hello world") {
		t.Errorf("expected 'Hello world' in output, got:\n%s", result)
	}
}

func TestBlocksToMarkdown_NoAutoTitle(t *testing.T) {
	blocks := []*larkdocx.Block{
		makePageBlock("doc1"),
		makeTextBlock("b1", "content"),
	}
	result := BlocksToMarkdown(blocks, "doc1", "Test", nil, nil)
	if strings.HasPrefix(result, "# Test\n") {
		t.Errorf("minimal 模式不应输出自动标题行, got:\n%s", result)
	}
}

func TestBlocksToMarkdown_Headings(t *testing.T) {
	blocks := []*larkdocx.Block{
		makePageBlock("doc1"),
		{
			BlockId: strPtr("h1"), BlockType: intPtr(BlockTypeHeading1), ParentId: strPtr("doc1"),
			Heading1: &larkdocx.Text{Elements: []*larkdocx.TextElement{
				{TextRun: &larkdocx.TextRun{Content: strPtr("Title")}},
			}},
		},
		{
			BlockId: strPtr("h2"), BlockType: intPtr(BlockTypeHeading2), ParentId: strPtr("doc1"),
			Heading2: &larkdocx.Text{Elements: []*larkdocx.TextElement{
				{TextRun: &larkdocx.TextRun{Content: strPtr("Subtitle")}},
			}},
		},
		{
			BlockId: strPtr("h3"), BlockType: intPtr(BlockTypeHeading3), ParentId: strPtr("doc1"),
			Heading3: &larkdocx.Text{Elements: []*larkdocx.TextElement{
				{TextRun: &larkdocx.TextRun{Content: strPtr("Section")}},
			}},
		},
	}
	result := BlocksToMarkdown(blocks, "doc1", "Doc", nil, nil)
	if !strings.Contains(result, "# Title\n") {
		t.Error("expected H1")
	}
	if !strings.Contains(result, "## Subtitle\n") {
		t.Error("expected H2")
	}
	if !strings.Contains(result, "### Section\n") {
		t.Error("expected H3")
	}
}

func TestBlocksToMarkdown_BulletList(t *testing.T) {
	blocks := []*larkdocx.Block{
		makePageBlock("doc1"),
		{
			BlockId: strPtr("b1"), BlockType: intPtr(BlockTypeBullet), ParentId: strPtr("doc1"),
			Bullet: &larkdocx.Text{Elements: []*larkdocx.TextElement{
				{TextRun: &larkdocx.TextRun{Content: strPtr("item1")}},
			}},
		},
		{
			BlockId: strPtr("b2"), BlockType: intPtr(BlockTypeBullet), ParentId: strPtr("doc1"),
			Bullet: &larkdocx.Text{Elements: []*larkdocx.TextElement{
				{TextRun: &larkdocx.TextRun{Content: strPtr("item2")}},
			}},
		},
	}
	result := BlocksToMarkdown(blocks, "doc1", "Doc", nil, nil)
	if !strings.Contains(result, "- item1\n") {
		t.Error("expected bullet item1")
	}
	if !strings.Contains(result, "- item2\n") {
		t.Error("expected bullet item2")
	}
}

func TestBlocksToMarkdown_OrderedList(t *testing.T) {
	blocks := []*larkdocx.Block{
		makePageBlock("doc1"),
		{
			BlockId: strPtr("o1"), BlockType: intPtr(BlockTypeOrdered), ParentId: strPtr("doc1"),
			Ordered: &larkdocx.Text{Elements: []*larkdocx.TextElement{
				{TextRun: &larkdocx.TextRun{Content: strPtr("first")}},
			}},
		},
	}
	result := BlocksToMarkdown(blocks, "doc1", "Doc", nil, nil)
	if !strings.Contains(result, "1. first\n") {
		t.Errorf("expected ordered list, got:\n%s", result)
	}
}

func TestBlocksToMarkdown_OrderedListIndent(t *testing.T) {
	blocks := []*larkdocx.Block{
		makePageBlock("doc1"),
		{
			BlockId: strPtr("o1"), BlockType: intPtr(BlockTypeOrdered), ParentId: strPtr("doc1"),
			Ordered: &larkdocx.Text{Elements: []*larkdocx.TextElement{
				{TextRun: &larkdocx.TextRun{Content: strPtr("parent")}},
			}},
			Children: []string{"o2"},
		},
		{
			BlockId: strPtr("o2"), BlockType: intPtr(BlockTypeOrdered), ParentId: strPtr("o1"),
			Ordered: &larkdocx.Text{Elements: []*larkdocx.TextElement{
				{TextRun: &larkdocx.TextRun{Content: strPtr("child")}},
			}},
		},
	}
	result := BlocksToMarkdown(blocks, "doc1", "Doc", nil, nil)
	if !strings.Contains(result, "   1. child\n") {
		t.Errorf("expected 3-space indented ordered child, got:\n%s", result)
	}
}

func TestBlocksToMarkdown_CodeBlock(t *testing.T) {
	langGo := 22
	blocks := []*larkdocx.Block{
		makePageBlock("doc1"),
		{
			BlockId: strPtr("c1"), BlockType: intPtr(BlockTypeCode), ParentId: strPtr("doc1"),
			Code: &larkdocx.Text{
				Style: &larkdocx.TextStyle{Language: &langGo},
				Elements: []*larkdocx.TextElement{
					{TextRun: &larkdocx.TextRun{Content: strPtr("fmt.Println()")}},
				},
			},
		},
	}
	result := BlocksToMarkdown(blocks, "doc1", "Doc", nil, nil)
	if !strings.Contains(result, "```go\n") {
		t.Error("expected go code fence")
	}
	if !strings.Contains(result, "fmt.Println()") {
		t.Error("expected code content")
	}
}

func TestBlocksToMarkdown_CodeBlockTrailingNewline(t *testing.T) {
	blocks := []*larkdocx.Block{
		makePageBlock("doc1"),
		{
			BlockId: strPtr("c1"), BlockType: intPtr(BlockTypeCode), ParentId: strPtr("doc1"),
			Code: &larkdocx.Text{
				Style: &larkdocx.TextStyle{Language: intPtr(1)},
				Elements: []*larkdocx.TextElement{
					{TextRun: &larkdocx.TextRun{Content: strPtr("line1\n\nline3\n")}},
				},
			},
		},
	}
	result := BlocksToMarkdown(blocks, "doc1", "Doc", nil, nil)
	if !strings.Contains(result, "line1\n\nline3\n```") {
		t.Errorf("expected preserved empty line in code block, got:\n%s", result)
	}
}

func TestBlocksToMarkdown_Quote(t *testing.T) {
	blocks := []*larkdocx.Block{
		makePageBlock("doc1"),
		{
			BlockId: strPtr("q1"), BlockType: intPtr(BlockTypeQuote), ParentId: strPtr("doc1"),
			Quote: &larkdocx.Text{Elements: []*larkdocx.TextElement{
				{TextRun: &larkdocx.TextRun{Content: strPtr("quoted")}},
			}},
		},
	}
	result := BlocksToMarkdown(blocks, "doc1", "Doc", nil, nil)
	if !strings.Contains(result, "> quoted\n") {
		t.Errorf("expected blockquote, got:\n%s", result)
	}
}

func TestBlocksToMarkdown_QuoteMultiline(t *testing.T) {
	blocks := []*larkdocx.Block{
		makePageBlock("doc1"),
		{
			BlockId: strPtr("q1"), BlockType: intPtr(BlockTypeQuote), ParentId: strPtr("doc1"),
			Quote: &larkdocx.Text{Elements: []*larkdocx.TextElement{
				{TextRun: &larkdocx.TextRun{Content: strPtr("line1\nline2")}},
			}},
		},
	}
	result := BlocksToMarkdown(blocks, "doc1", "Doc", nil, nil)
	if !strings.Contains(result, "> line1\n> line2\n") {
		t.Errorf("expected multiline quote with > prefix on each line, got:\n%s", result)
	}
}

func TestBlocksToMarkdown_Divider(t *testing.T) {
	blocks := []*larkdocx.Block{
		makePageBlock("doc1"),
		{BlockId: strPtr("d1"), BlockType: intPtr(BlockTypeDivider), ParentId: strPtr("doc1")},
	}
	result := BlocksToMarkdown(blocks, "doc1", "Doc", nil, nil)
	if !strings.Contains(result, "---\n") {
		t.Error("expected divider")
	}
}

func TestBlocksToMarkdown_Image(t *testing.T) {
	token := "img_token_123"
	blocks := []*larkdocx.Block{
		makePageBlock("doc1"),
		{
			BlockId: strPtr("i1"), BlockType: intPtr(BlockTypeImage), ParentId: strPtr("doc1"),
			Image: &larkdocx.Image{Token: strPtr(token)},
		},
	}

	t.Run("with URL map", func(t *testing.T) {
		urls := map[string]string{token: "https://example.com/img.png"}
		result := BlocksToMarkdown(blocks, "doc1", "Doc", urls, nil)
		if !strings.Contains(result, "![](https://example.com/img.png)") {
			t.Errorf("expected image with URL, got:\n%s", result)
		}
	})

	t.Run("without URL map", func(t *testing.T) {
		result := BlocksToMarkdown(blocks, "doc1", "Doc", nil, nil)
		if !strings.Contains(result, "feishu://image/"+token) {
			t.Errorf("expected fallback feishu image ref, got:\n%s", result)
		}
	})
}

func TestBlocksToMarkdown_Todo(t *testing.T) {
	blocks := []*larkdocx.Block{
		makePageBlock("doc1"),
		{
			BlockId: strPtr("t1"), BlockType: intPtr(BlockTypeTodo), ParentId: strPtr("doc1"),
			Todo: &larkdocx.Text{
				Style:    &larkdocx.TextStyle{Done: boolPtr(true)},
				Elements: []*larkdocx.TextElement{{TextRun: &larkdocx.TextRun{Content: strPtr("done task")}}},
			},
		},
		{
			BlockId: strPtr("t2"), BlockType: intPtr(BlockTypeTodo), ParentId: strPtr("doc1"),
			Todo: &larkdocx.Text{
				Style:    &larkdocx.TextStyle{Done: boolPtr(false)},
				Elements: []*larkdocx.TextElement{{TextRun: &larkdocx.TextRun{Content: strPtr("open task")}}},
			},
		},
	}
	result := BlocksToMarkdown(blocks, "doc1", "Doc", nil, nil)
	if !strings.Contains(result, "[x] done task") {
		t.Error("expected checked todo")
	}
	if !strings.Contains(result, "[ ] open task") {
		t.Error("expected unchecked todo")
	}
}

func TestBlocksToMarkdown_InlineStyles(t *testing.T) {
	blocks := []*larkdocx.Block{
		makePageBlock("doc1"),
		{
			BlockId: strPtr("s1"), BlockType: intPtr(BlockTypeText), ParentId: strPtr("doc1"),
			Text: &larkdocx.Text{Elements: []*larkdocx.TextElement{
				{TextRun: &larkdocx.TextRun{
					Content:          strPtr("bold"),
					TextElementStyle: &larkdocx.TextElementStyle{Bold: boolPtr(true)},
				}},
				{TextRun: &larkdocx.TextRun{
					Content:          strPtr("italic"),
					TextElementStyle: &larkdocx.TextElementStyle{Italic: boolPtr(true)},
				}},
				{TextRun: &larkdocx.TextRun{
					Content:          strPtr("code"),
					TextElementStyle: &larkdocx.TextElementStyle{InlineCode: boolPtr(true)},
				}},
				{TextRun: &larkdocx.TextRun{
					Content:          strPtr("strike"),
					TextElementStyle: &larkdocx.TextElementStyle{Strikethrough: boolPtr(true)},
				}},
				{TextRun: &larkdocx.TextRun{
					Content: strPtr("link"),
					TextElementStyle: &larkdocx.TextElementStyle{
						Link: &larkdocx.Link{Url: strPtr("https://example.com")},
					},
				}},
			}},
		},
	}
	result := BlocksToMarkdown(blocks, "doc1", "Doc", nil, nil)
	if !strings.Contains(result, "**bold**") {
		t.Error("expected bold")
	}
	if !strings.Contains(result, "*italic*") {
		t.Error("expected italic")
	}
	if !strings.Contains(result, "`code`") {
		t.Error("expected inline code")
	}
	if !strings.Contains(result, "~~strike~~") {
		t.Error("expected strikethrough")
	}
	if !strings.Contains(result, "[link](https://example.com)") {
		t.Error("expected link")
	}
}

func TestBlocksToMarkdown_Table(t *testing.T) {
	rows := 2
	cols := 2
	blocks := []*larkdocx.Block{
		makePageBlock("doc1"),
		{
			BlockId: strPtr("tbl"), BlockType: intPtr(BlockTypeTable), ParentId: strPtr("doc1"),
			Table: &larkdocx.Table{
				Property: &larkdocx.TableProperty{RowSize: &rows, ColumnSize: &cols},
				Cells:    []string{"c00", "c01", "c10", "c11"},
			},
			Children: []string{"c00", "c01", "c10", "c11"},
		},
		{
			BlockId: strPtr("c00"), BlockType: intPtr(BlockTypeTableCell), ParentId: strPtr("tbl"),
			Children: []string{"c00t"},
		},
		{
			BlockId: strPtr("c01"), BlockType: intPtr(BlockTypeTableCell), ParentId: strPtr("tbl"),
			Children: []string{"c01t"},
		},
		{
			BlockId: strPtr("c10"), BlockType: intPtr(BlockTypeTableCell), ParentId: strPtr("tbl"),
			Children: []string{"c10t"},
		},
		{
			BlockId: strPtr("c11"), BlockType: intPtr(BlockTypeTableCell), ParentId: strPtr("tbl"),
			Children: []string{"c11t"},
		},
		makeTextBlock("c00t", "A"),
		makeTextBlock("c01t", "B"),
		makeTextBlock("c10t", "1"),
		makeTextBlock("c11t", "2"),
	}
	blocks[6].ParentId = strPtr("c00")
	blocks[7].ParentId = strPtr("c01")
	blocks[8].ParentId = strPtr("c10")
	blocks[9].ParentId = strPtr("c11")

	result := BlocksToMarkdown(blocks, "doc1", "Doc", nil, nil)
	if !strings.Contains(result, "| A") {
		t.Errorf("expected table header A, got:\n%s", result)
	}
	if !strings.Contains(result, "| B") {
		t.Errorf("expected table header B, got:\n%s", result)
	}
	if !strings.Contains(result, "---") {
		t.Error("expected table separator")
	}
}

func TestBlocksToMarkdown_TableWithStyles(t *testing.T) {
	rows := 1
	cols := 1
	blocks := []*larkdocx.Block{
		makePageBlock("doc1"),
		{
			BlockId: strPtr("tbl"), BlockType: intPtr(BlockTypeTable), ParentId: strPtr("doc1"),
			Table: &larkdocx.Table{
				Property: &larkdocx.TableProperty{RowSize: &rows, ColumnSize: &cols},
				Cells:    []string{"c00"},
			},
			Children: []string{"c00"},
		},
		{
			BlockId: strPtr("c00"), BlockType: intPtr(BlockTypeTableCell), ParentId: strPtr("tbl"),
			Children: []string{"c00t"},
		},
		{
			BlockId: strPtr("c00t"), BlockType: intPtr(BlockTypeText), ParentId: strPtr("c00"),
			Text: &larkdocx.Text{Elements: []*larkdocx.TextElement{
				{TextRun: &larkdocx.TextRun{
					Content:          strPtr("bold"),
					TextElementStyle: &larkdocx.TextElementStyle{Bold: boolPtr(true)},
				}},
			}},
		},
	}
	result := BlocksToMarkdown(blocks, "doc1", "Doc", nil, nil)
	if !strings.Contains(result, "**bold**") {
		t.Errorf("expected bold text in table cell, got:\n%s", result)
	}
}

func TestBlocksToMarkdown_EmptyBlocks(t *testing.T) {
	blocks := []*larkdocx.Block{makePageBlock("doc1")}
	result := BlocksToMarkdown(blocks, "doc1", "Empty", nil, nil)
	if result != "" {
		t.Errorf("expected empty output for page-only doc, got:\n%s", result)
	}
}

func TestBlocksToMarkdown_ConsecutiveEmptyText(t *testing.T) {
	blocks := []*larkdocx.Block{
		makePageBlock("doc1"),
		makeTextBlock("b1", "above"),
		{
			BlockId: strPtr("e1"), BlockType: intPtr(BlockTypeText), ParentId: strPtr("doc1"),
			Text: &larkdocx.Text{Elements: []*larkdocx.TextElement{
				{TextRun: &larkdocx.TextRun{Content: strPtr("")}},
			}},
		},
		{
			BlockId: strPtr("e2"), BlockType: intPtr(BlockTypeText), ParentId: strPtr("doc1"),
			Text: &larkdocx.Text{Elements: []*larkdocx.TextElement{
				{TextRun: &larkdocx.TextRun{Content: strPtr("")}},
			}},
		},
		makeTextBlock("b2", "below"),
	}
	result := BlocksToMarkdown(blocks, "doc1", "Doc", nil, nil)
	expected := "above\n\nbelow\n\n"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBlocksToMarkdown_UnknownType(t *testing.T) {
	blocks := []*larkdocx.Block{
		makePageBlock("doc1"),
		{BlockId: strPtr("u1"), BlockType: intPtr(99), ParentId: strPtr("doc1")},
	}
	result := BlocksToMarkdown(blocks, "doc1", "Doc", nil, nil)
	if !strings.Contains(result, "feishu:id=u1,type=unknown_99") {
		t.Errorf("expected unknown type placeholder, got:\n%s", result)
	}
}

func TestDisplayWidth(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"abc", 3},
		{"中文", 4},
		{"a中b", 4},
		{"", 0},
	}
	for _, tt := range tests {
		if got := displayWidth(tt.input); got != tt.expected {
			t.Errorf("displayWidth(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestPadRight(t *testing.T) {
	if got := padRight("abc", 6); got != "abc   " {
		t.Errorf("padRight('abc', 6) = %q", got)
	}
	if got := padRight("abc", 2); got != "abc" {
		t.Errorf("padRight('abc', 2) = %q", got)
	}
}

func TestBlocksToMarkdown_Mermaid(t *testing.T) {
	record := `{"data":"graph TD\n    A-->B","theme":"default","view":"chart"}`
	blocks := []*larkdocx.Block{
		makePageBlock("doc1"),
		{
			BlockId: strPtr("m1"), BlockType: intPtr(BlockTypeAddOns), ParentId: strPtr("doc1"),
			AddOns: &larkdocx.AddOns{
				ComponentTypeId: strPtr("blk_631fefbbae02400430b8f9f4"),
				Record:          &record,
			},
		},
	}
	result := BlocksToMarkdown(blocks, "doc1", "Doc", nil, nil)
	if !strings.Contains(result, "```mermaid\n") {
		t.Errorf("expected mermaid code fence, got:\n%s", result)
	}
	if !strings.Contains(result, "graph TD\n    A-->B") {
		t.Errorf("expected mermaid content, got:\n%s", result)
	}
}

func TestRenderInline_Equation(t *testing.T) {
	formula := "x^2 + y^2 = r^2"
	t1 := &larkdocx.Text{
		Elements: []*larkdocx.TextElement{
			{TextRun: &larkdocx.TextRun{Content: strPtr("The formula ")}},
			{Equation: &larkdocx.Equation{Content: &formula}},
			{TextRun: &larkdocx.TextRun{Content: strPtr(" is a circle.")}},
		},
	}
	ec := &exportCtx{}
	result := ec.renderInline(t1)
	if !strings.Contains(result, "$x^2 + y^2 = r^2$") {
		t.Errorf("expected inline equation, got: %s", result)
	}
}

func TestRenderInline_MentionUser(t *testing.T) {
	t1 := &larkdocx.Text{
		Elements: []*larkdocx.TextElement{
			{MentionUser: &larkdocx.MentionUser{UserId: strPtr("ou_abc123")}},
		},
	}
	ec := &exportCtx{}
	result := ec.renderInline(t1)
	if !strings.Contains(result, "@ou_abc123") {
		t.Errorf("expected mention user, got: %s", result)
	}
}

func TestRenderInline_MentionDoc(t *testing.T) {
	t1 := &larkdocx.Text{
		Elements: []*larkdocx.TextElement{
			{MentionDoc: &larkdocx.MentionDoc{
				Title: strPtr("设计文档"),
				Url:   strPtr("https://example.feishu.cn/docx/xxx"),
			}},
		},
	}
	ec := &exportCtx{}
	result := ec.renderInline(t1)
	if !strings.Contains(result, "[设计文档](https://example.feishu.cn/docx/xxx)") {
		t.Errorf("expected mention doc link, got: %s", result)
	}
}

func TestRenderInline_EscapeSpecialChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"asterisk", "2 * 3 = 6", `2 \* 3 = 6`},
		{"hash at start", "#标题", `\#标题`},
		{"hash not at start", "C#语言", `C#语言`},
		{"brackets", "[link]", `\[link\]`},
		{"pipe", "a|b", `a\|b`},
		{"backtick", "use `code`", "use \\`code\\`"},
		{"gt at start", "> quoted", `\> quoted`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeMd(tt.input)
			if result != tt.expected {
				t.Errorf("escapeMd(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRenderInline_NoEscapeInCode(t *testing.T) {
	t1 := &larkdocx.Text{
		Elements: []*larkdocx.TextElement{
			{TextRun: &larkdocx.TextRun{
				Content:          strPtr("a * b"),
				TextElementStyle: &larkdocx.TextElementStyle{InlineCode: boolPtr(true)},
			}},
		},
	}
	ec := &exportCtx{}
	result := ec.renderInline(t1)
	if result != "`a * b`" {
		t.Errorf("expected no escape in inline code, got: %s", result)
	}
}

func TestReverseLangMap(t *testing.T) {
	if got := reverseLangMap(22); got != "go" {
		t.Errorf("expected 'go', got %q", got)
	}
	if got := reverseLangMap(49); got != "python" {
		t.Errorf("expected 'python', got %q", got)
	}
	if got := reverseLangMap(9999); got != "" {
		t.Errorf("expected empty for unknown, got %q", got)
	}
}

func TestDeref(t *testing.T) {
	s := "hello"
	if got := Deref(&s); got != "hello" {
		t.Errorf("Deref(&'hello') = %q", got)
	}
	if got := Deref(nil); got != "" {
		t.Errorf("Deref(nil) = %q", got)
	}
}

func TestDerefInt(t *testing.T) {
	n := 42
	if got := DerefInt(&n); got != 42 {
		t.Errorf("DerefInt(&42) = %d", got)
	}
	if got := DerefInt(nil); got != 0 {
		t.Errorf("DerefInt(nil) = %d", got)
	}
}

func TestDerefBool(t *testing.T) {
	b := true
	if got := DerefBool(&b); got != true {
		t.Error("DerefBool(&true) should be true")
	}
	if got := DerefBool(nil); got != false {
		t.Error("DerefBool(nil) should be false")
	}
}

func TestEscapeMd_HashMiddle(t *testing.T) {
	result := escapeMd("C# is great")
	if result != `C# is great` {
		t.Errorf("expected C# is great (no escape mid-line), got: %s", result)
	}
}

func TestRenderPlainText_OnlyLastNewline(t *testing.T) {
	ec := &exportCtx{}
	text := &larkdocx.Text{
		Elements: []*larkdocx.TextElement{
			{TextRun: &larkdocx.TextRun{Content: strPtr("line1\n\nline3\n")}},
		},
	}
	result := ec.renderPlainText(text)
	if result != "line1\n\nline3" {
		t.Errorf("expected only last newline trimmed, got: %q", result)
	}
}
