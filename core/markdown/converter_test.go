package markdown

import (
	"testing"

	larkdocx "github.com/larksuite/oapi-sdk-go/v3/service/docx/v1"
)

func blockType(b *larkdocx.Block) int {
	if b.BlockType == nil {
		return 0
	}
	return *b.BlockType
}

func textContent(t *larkdocx.Text) string {
	if t == nil {
		return ""
	}
	var s string
	for _, el := range t.Elements {
		if el.TextRun != nil && el.TextRun.Content != nil {
			s += *el.TextRun.Content
		}
	}
	return s
}

func TestConvertParagraph(t *testing.T) {
	blocks := Convert([]byte("Hello world"))
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	b := blocks[0]
	if blockType(b.Raw) != 2 {
		t.Errorf("expected BlockType 2 (Text), got %d", blockType(b.Raw))
	}
	if got := textContent(b.Raw.Text); got != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", got)
	}
}

func TestConvertHeadings(t *testing.T) {
	md := "# H1\n\n## H2\n\n### H3\n\n#### H4\n\n##### H5\n\n###### H6\n"
	blocks := Convert([]byte(md))
	if len(blocks) != 6 {
		t.Fatalf("expected 6 blocks, got %d", len(blocks))
	}

	expectedTypes := []int{3, 4, 5, 6, 7, 8}
	expectedTexts := []string{"H1", "H2", "H3", "H4", "H5", "H6"}

	for i, b := range blocks {
		if blockType(b.Raw) != expectedTypes[i] {
			t.Errorf("block %d: expected type %d, got %d", i, expectedTypes[i], blockType(b.Raw))
		}
		var txt *larkdocx.Text
		switch expectedTypes[i] {
		case 3:
			txt = b.Raw.Heading1
		case 4:
			txt = b.Raw.Heading2
		case 5:
			txt = b.Raw.Heading3
		case 6:
			txt = b.Raw.Heading4
		case 7:
			txt = b.Raw.Heading5
		case 8:
			txt = b.Raw.Heading6
		}
		if got := textContent(txt); got != expectedTexts[i] {
			t.Errorf("block %d: expected text %q, got %q", i, expectedTexts[i], got)
		}
	}
}

func TestConvertBoldItalicCode(t *testing.T) {
	md := "**bold** *italic* `code`"
	blocks := Convert([]byte(md))
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	b := blocks[0]
	if blockType(b.Raw) != 2 {
		t.Fatalf("expected BlockType 2, got %d", blockType(b.Raw))
	}
	els := b.Raw.Text.Elements

	findStyled := func(name string, check func(*larkdocx.TextElementStyle) bool) {
		for _, el := range els {
			if el.TextRun == nil || el.TextRun.TextElementStyle == nil {
				continue
			}
			if check(el.TextRun.TextElementStyle) {
				return
			}
		}
		t.Errorf("expected an element with %s style", name)
	}

	findStyled("bold", func(s *larkdocx.TextElementStyle) bool {
		return s.Bold != nil && *s.Bold
	})
	findStyled("italic", func(s *larkdocx.TextElementStyle) bool {
		return s.Italic != nil && *s.Italic
	})
	findStyled("inline code", func(s *larkdocx.TextElementStyle) bool {
		return s.InlineCode != nil && *s.InlineCode
	})
}

func TestConvertLink(t *testing.T) {
	md := "[click](https://example.com)"
	blocks := Convert([]byte(md))
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	el := blocks[0].Raw.Text.Elements[0]
	if el.TextRun.TextElementStyle == nil || el.TextRun.TextElementStyle.Link == nil {
		t.Fatal("expected link style")
	}
	if got := *el.TextRun.TextElementStyle.Link.Url; got != "https://example.com" {
		t.Errorf("expected URL 'https://example.com', got %q", got)
	}
	if got := *el.TextRun.Content; got != "click" {
		t.Errorf("expected text 'click', got %q", got)
	}
}

func TestConvertBulletList(t *testing.T) {
	md := "- item1\n- item2\n- item3\n"
	blocks := Convert([]byte(md))
	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(blocks))
	}
	for i, b := range blocks {
		if blockType(b.Raw) != 12 {
			t.Errorf("block %d: expected type 12 (Bullet), got %d", i, blockType(b.Raw))
		}
		if b.IndentLevel != 0 {
			t.Errorf("block %d: expected indent 0, got %d", i, b.IndentLevel)
		}
	}
}

func TestConvertOrderedList(t *testing.T) {
	md := "1. first\n2. second\n"
	blocks := Convert([]byte(md))
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	for i, b := range blocks {
		if blockType(b.Raw) != 13 {
			t.Errorf("block %d: expected type 13 (Ordered), got %d", i, blockType(b.Raw))
		}
	}
}

func TestConvertNestedList(t *testing.T) {
	md := "- parent\n  - child\n    - grandchild\n"
	blocks := Convert([]byte(md))
	if len(blocks) < 3 {
		t.Fatalf("expected at least 3 blocks, got %d", len(blocks))
	}
	if blocks[0].IndentLevel != 0 {
		t.Errorf("parent indent: expected 0, got %d", blocks[0].IndentLevel)
	}
	if blocks[1].IndentLevel != 1 {
		t.Errorf("child indent: expected 1, got %d", blocks[1].IndentLevel)
	}
	if blocks[2].IndentLevel != 2 {
		t.Errorf("grandchild indent: expected 2, got %d", blocks[2].IndentLevel)
	}
}

func TestConvertNestedOrderedList(t *testing.T) {
	md := "1. parent\n   1. child\n      1. grandchild\n"
	blocks := Convert([]byte(md))
	if len(blocks) < 3 {
		t.Fatalf("expected at least 3 blocks, got %d", len(blocks))
	}
	if blocks[0].IndentLevel != 0 {
		t.Errorf("parent indent: expected 0, got %d", blocks[0].IndentLevel)
	}
	if blocks[1].IndentLevel != 1 {
		t.Errorf("child indent: expected 1, got %d", blocks[1].IndentLevel)
	}
	if blocks[2].IndentLevel != 2 {
		t.Errorf("grandchild indent: expected 2, got %d", blocks[2].IndentLevel)
	}
}

func TestConvertCodeBlock(t *testing.T) {
	md := "```go\nfmt.Println(\"hello\")\n```\n"
	blocks := Convert([]byte(md))
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	b := blocks[0]
	if blockType(b.Raw) != 14 {
		t.Errorf("expected type 14 (Code), got %d", blockType(b.Raw))
	}
	if got := textContent(b.Raw.Code); got != "fmt.Println(\"hello\")" {
		t.Errorf("expected code content, got %q", got)
	}
	if b.Raw.Code.Style == nil || b.Raw.Code.Style.Language == nil {
		t.Fatal("expected language style")
	}
	if *b.Raw.Code.Style.Language != 22 {
		t.Errorf("expected Go language code 22, got %d", *b.Raw.Code.Style.Language)
	}
}

func TestConvertCodeBlock_PreservesInternalNewlines(t *testing.T) {
	md := "```\nline1\n\nline3\n```\n"
	blocks := Convert([]byte(md))
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	got := textContent(blocks[0].Raw.Code)
	if got != "line1\n\nline3" {
		t.Errorf("expected preserved empty line, got %q", got)
	}
}

func TestConvertMermaidBlock(t *testing.T) {
	md := "```mermaid\ngraph TD\n  A-->B\n```\n"
	blocks := Convert([]byte(md))
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	b := blocks[0]
	if blockType(b.Raw) != 40 {
		t.Errorf("expected type 40 (AddOns), got %d", blockType(b.Raw))
	}
	if b.Raw.AddOns == nil {
		t.Fatal("expected AddOns to be set")
	}
	if b.Raw.AddOns.ComponentTypeId == nil || *b.Raw.AddOns.ComponentTypeId != mermaidComponentTypeID {
		t.Error("expected mermaid component type ID")
	}
}

func TestConvertBlockquote(t *testing.T) {
	md := "> quoted text\n"
	blocks := Convert([]byte(md))
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	b := blocks[0]
	if blockType(b.Raw) != 15 {
		t.Errorf("expected type 15 (Quote), got %d", blockType(b.Raw))
	}
	if got := textContent(b.Raw.Quote); got != "quoted text" {
		t.Errorf("expected 'quoted text', got %q", got)
	}
}

func TestConvertMultilineBlockquote(t *testing.T) {
	md := "> line1\n> line2\n"
	blocks := Convert([]byte(md))
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block for continuous blockquote, got %d", len(blocks))
	}
}

func TestConvertThematicBreak(t *testing.T) {
	md := "---\n"
	blocks := Convert([]byte(md))
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blockType(blocks[0].Raw) != 22 {
		t.Errorf("expected type 22 (Divider), got %d", blockType(blocks[0].Raw))
	}
}

func TestConvertTodoList(t *testing.T) {
	md := "- [x] done\n- [ ] todo\n"
	blocks := Convert([]byte(md))
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	for _, b := range blocks {
		if blockType(b.Raw) != 17 {
			t.Errorf("expected type 17 (Todo), got %d", blockType(b.Raw))
		}
	}
	if blocks[0].Raw.Todo.Style == nil || blocks[0].Raw.Todo.Style.Done == nil || !*blocks[0].Raw.Todo.Style.Done {
		t.Error("expected first todo to be done")
	}
	if blocks[1].Raw.Todo.Style != nil && blocks[1].Raw.Todo.Style.Done != nil && *blocks[1].Raw.Todo.Style.Done {
		t.Error("expected second todo to not be done")
	}
}

func TestConvertImage(t *testing.T) {
	md := "![alt text](images/photo.png)\n"
	blocks := Convert([]byte(md))
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	b := blocks[0]
	if b.ImageSrc != "images/photo.png" {
		t.Errorf("expected ImageSrc 'images/photo.png', got %q", b.ImageSrc)
	}
	if b.ImageAlt != "alt text" {
		t.Errorf("expected ImageAlt 'alt text', got %q", b.ImageAlt)
	}
}

func TestConvertTable(t *testing.T) {
	md := "| A | B |\n| --- | --- |\n| 1 | 2 |\n| 3 | 4 |\n"
	blocks := Convert([]byte(md))
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	b := blocks[0]
	if blockType(b.Raw) != 31 {
		t.Errorf("expected type 31 (Table), got %d", blockType(b.Raw))
	}
	if b.TableData == nil {
		t.Fatal("expected TableData to be set")
	}
	if b.TableData.Rows != 3 {
		t.Errorf("expected 3 rows, got %d", b.TableData.Rows)
	}
	if b.TableData.Cols != 2 {
		t.Errorf("expected 2 cols, got %d", b.TableData.Cols)
	}
	if len(b.TableData.CellBlocks) != 6 {
		t.Errorf("expected 6 cell blocks, got %d", len(b.TableData.CellBlocks))
	}
}

func TestConvertStrikethrough(t *testing.T) {
	md := "~~deleted~~"
	blocks := Convert([]byte(md))
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	el := blocks[0].Raw.Text.Elements[0]
	if el.TextRun.TextElementStyle == nil || el.TextRun.TextElementStyle.Strikethrough == nil || !*el.TextRun.TextElementStyle.Strikethrough {
		t.Error("expected strikethrough style")
	}
}

func TestConvertEmptyInput(t *testing.T) {
	blocks := Convert([]byte(""))
	if len(blocks) != 0 {
		t.Errorf("expected 0 blocks for empty input, got %d", len(blocks))
	}
}

func TestConvertMixedContent(t *testing.T) {
	md := `# Title

Some text.

- bullet1
- bullet2

` + "```python" + `
print("hello")
` + "```" + `

---
`
	blocks := Convert([]byte(md))
	if len(blocks) < 6 {
		t.Fatalf("expected at least 6 blocks, got %d", len(blocks))
	}

	expected := []int{3, 2, 12, 12, 14, 22}
	for i, bt := range expected {
		if i >= len(blocks) {
			break
		}
		if blockType(blocks[i].Raw) != bt {
			t.Errorf("block %d: expected type %d, got %d", i, bt, blockType(blocks[i].Raw))
		}
	}
}

func TestConvertBoldItalicCombined(t *testing.T) {
	md := "***bold and italic***"
	blocks := Convert([]byte(md))
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	els := blocks[0].Raw.Text.Elements
	found := false
	for _, el := range els {
		if el.TextRun == nil || el.TextRun.TextElementStyle == nil {
			continue
		}
		s := el.TextRun.TextElementStyle
		if s.Bold != nil && *s.Bold && s.Italic != nil && *s.Italic {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected an element with both bold and italic styles")
	}
}

func TestConvertTableWithStyles(t *testing.T) {
	md := "| **bold** | *italic* |\n| --- | --- |\n| normal | `code` |\n"
	blocks := Convert([]byte(md))
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	td := blocks[0].TableData
	if td == nil {
		t.Fatal("expected TableData")
	}
	boldCell := td.CellBlocks[0]
	found := false
	for _, el := range boldCell.Text.Elements {
		if el.TextRun != nil && el.TextRun.TextElementStyle != nil &&
			el.TextRun.TextElementStyle.Bold != nil && *el.TextRun.TextElementStyle.Bold {
			found = true
		}
	}
	if !found {
		t.Error("expected bold style in table cell")
	}
}
