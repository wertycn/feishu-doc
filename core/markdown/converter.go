package markdown

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"

	larkdocx "github.com/larksuite/oapi-sdk-go/v3/service/docx/v1"
)

var (
	uidCommentRe = regexp.MustCompile(`^<!-- uid:(\S+) -->$`)
	inlineMathRe = regexp.MustCompile(`\$([^\$]+)\$`)
)

type ConvertResult struct {
	Blocks      []Block
	FrontMatter *FrontMatter
}

type inlineStyle struct {
	bold, italic, strikethrough, inlineCode bool
	linkURL                                 string
}

type Block struct {
	Raw         *larkdocx.Block
	IndentLevel int
	TableData   *TableData
	ImageSrc    string
	ImageAlt    string
}

type TableData struct {
	Rows       int
	Cols       int
	CellBlocks []*larkdocx.Block
}

func Convert(source []byte) []Block {
	r := ConvertWithMeta(source)
	return r.Blocks
}

func ConvertWithMeta(source []byte) ConvertResult {
	_, body := ParseFrontMatter(source)

	md := goldmark.New(goldmark.WithExtensions(extension.GFM, extension.TaskList))
	doc := md.Parser().Parse(text.NewReader(body))

	var out []Block
	for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
		nodes := convertNode(c, body, 0)
		out = append(out, nodes...)
	}
	return ConvertResult{Blocks: out}
}

func wrap(b *larkdocx.Block, indent int) []Block {
	return []Block{{Raw: b, IndentLevel: indent}}
}

func convertNode(node ast.Node, source []byte, depth int) []Block {
	switch n := node.(type) {
	case *ast.Heading:
		return wrap(convertHeading(n, source), 0)
	case *ast.Paragraph:
		if img := detectSoloImage(n); img != nil {
			src := string(img.Destination)
			alt := string(img.Text(source))
			return []Block{{
				Raw:      larkdocx.NewBlockBuilder().BlockType(27).Build(),
				ImageSrc: src,
				ImageAlt: alt,
			}}
		}
		return wrap(convertParagraph(n, source), 0)
	case *ast.TextBlock:
		return wrap(convertParagraph(n, source), 0)
	case *ast.FencedCodeBlock:
		return wrap(convertFencedCode(n, source), 0)
	case *ast.CodeBlock:
		return wrap(convertIndentedCode(n, source), 0)
	case *ast.List:
		return convertList(n, source, depth)
	case *ast.Blockquote:
		return convertBlockquote(n, source)
	case *ast.ThematicBreak:
		return wrap(makeDivider(), 0)
	case *east.Table:
		return convertTable(n, source)
	case *ast.HTMLBlock:
		return nil
	default:
		return nil
	}
}

func convertHeading(n *ast.Heading, source []byte) *larkdocx.Block {
	els := extractInline(n, source, inlineStyle{})
	txt := larkdocx.NewTextBuilder().Elements(els).Build()
	b := larkdocx.NewBlockBuilder()
	switch n.Level {
	case 1:
		b.BlockType(3).Heading1(txt)
	case 2:
		b.BlockType(4).Heading2(txt)
	case 3:
		b.BlockType(5).Heading3(txt)
	case 4:
		b.BlockType(6).Heading4(txt)
	case 5:
		b.BlockType(7).Heading5(txt)
	default:
		b.BlockType(8).Heading6(txt)
	}
	return b.Build()
}

func convertParagraph(n ast.Node, source []byte) *larkdocx.Block {
	els := extractInline(n, source, inlineStyle{})
	if len(els) == 0 {
		els = []*larkdocx.TextElement{makeTextElement("", inlineStyle{})}
	}
	return larkdocx.NewBlockBuilder().BlockType(2).
		Text(larkdocx.NewTextBuilder().Elements(els).Build()).Build()
}

func detectSoloImage(n ast.Node) *ast.Image {
	count := 0
	var img *ast.Image
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if i, ok := c.(*ast.Image); ok {
			img = i
		}
		count++
	}
	if count == 1 && img != nil {
		return img
	}
	return nil
}

const mermaidComponentTypeID = "blk_631fefbbae02400430b8f9f4"

func convertFencedCode(n *ast.FencedCodeBlock, source []byte) *larkdocx.Block {
	var buf bytes.Buffer
	for i := 0; i < n.Lines().Len(); i++ {
		line := n.Lines().At(i)
		buf.Write(line.Value(source))
	}
	// R7: 只去除最后一个换行
	content := buf.String()
	if strings.HasSuffix(content, "\n") {
		content = content[:len(content)-1]
	}

	lang := ""
	if n.Language(source) != nil {
		lang = string(n.Language(source))
	}

	if strings.ToLower(lang) == "mermaid" {
		return makeMermaidBlock(content)
	}
	return makeCodeBlock(content, lang)
}

func makeMermaidBlock(content string) *larkdocx.Block {
	record, _ := json.Marshal(map[string]string{
		"data":  content,
		"theme": "default",
		"view":  "chart",
	})
	return larkdocx.NewBlockBuilder().
		BlockType(40).
		AddOns(larkdocx.NewAddOnsBuilder().
			ComponentTypeId(mermaidComponentTypeID).
			Record(string(record)).
			Build()).
		Build()
}

func convertIndentedCode(n *ast.CodeBlock, source []byte) *larkdocx.Block {
	var buf bytes.Buffer
	for i := 0; i < n.Lines().Len(); i++ {
		line := n.Lines().At(i)
		buf.Write(line.Value(source))
	}
	content := buf.String()
	if strings.HasSuffix(content, "\n") {
		content = content[:len(content)-1]
	}
	return makeCodeBlock(content, "")
}

func makeCodeBlock(content, lang string) *larkdocx.Block {
	style := larkdocx.NewTextStyleBuilder().Language(mapLanguage(lang)).Build()
	el := makeTextElement(content, inlineStyle{})
	txt := larkdocx.NewTextBuilder().Style(style).Elements([]*larkdocx.TextElement{el}).Build()
	return larkdocx.NewBlockBuilder().BlockType(14).Code(txt).Build()
}

func convertList(n *ast.List, source []byte, depth int) []Block {
	isOrdered := n.Marker == '.' || n.Marker == ')'
	var out []Block

	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		item, ok := child.(*ast.ListItem)
		if !ok {
			continue
		}
		for gc := item.FirstChild(); gc != nil; gc = gc.NextSibling() {
			switch gcn := gc.(type) {
			case *ast.Paragraph, *ast.TextBlock:
				isTodo, done := detectTaskCheckBox(gcn)
				els := extractInline(gcn, source, inlineStyle{})
				if len(els) == 0 {
					continue
				}

				var b *larkdocx.Block
				if isTodo {
					style := larkdocx.NewTextStyleBuilder().Done(done).Build()
					txt := larkdocx.NewTextBuilder().Style(style).Elements(els).Build()
					b = larkdocx.NewBlockBuilder().BlockType(17).Todo(txt).Build()
				} else if isOrdered {
					txt := larkdocx.NewTextBuilder().Elements(els).Build()
					b = larkdocx.NewBlockBuilder().BlockType(13).Ordered(txt).Build()
				} else {
					txt := larkdocx.NewTextBuilder().Elements(els).Build()
					b = larkdocx.NewBlockBuilder().BlockType(12).Bullet(txt).Build()
				}
				out = append(out, Block{Raw: b, IndentLevel: depth})
			case *ast.List:
				out = append(out, convertList(gcn, source, depth+1)...)
			default:
				out = append(out, convertNode(gcn, source, depth)...)
			}
		}
	}
	return out
}

func detectTaskCheckBox(node ast.Node) (isTodo bool, done bool) {
	first := node.FirstChild()
	if first == nil {
		return false, false
	}
	if cb, ok := first.(*east.TaskCheckBox); ok {
		return true, cb.IsChecked
	}
	return false, false
}

func convertBlockquote(n *ast.Blockquote, source []byte) []Block {
	var out []Block
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		switch cn := child.(type) {
		case *ast.Paragraph, *ast.TextBlock:
			els := extractInline(cn, source, inlineStyle{})
			if len(els) == 0 {
				els = []*larkdocx.TextElement{makeTextElement("", inlineStyle{})}
			}
			txt := larkdocx.NewTextBuilder().Elements(els).Build()
			out = append(out, Block{Raw: larkdocx.NewBlockBuilder().BlockType(15).Quote(txt).Build()})
		case *ast.Blockquote:
			out = append(out, convertBlockquote(cn, source)...)
		default:
			out = append(out, convertNode(cn, source, 0)...)
		}
	}
	return out
}

func makeDivider() *larkdocx.Block {
	return larkdocx.NewBlockBuilder().BlockType(22).Divider(&larkdocx.Divider{}).Build()
}

func extractInline(node ast.Node, source []byte, style inlineStyle) []*larkdocx.TextElement {
	var els []*larkdocx.TextElement
	for c := node.FirstChild(); c != nil; c = c.NextSibling() {
		els = append(els, extractInlineNode(c, source, style)...)
	}
	els = mergeAndSplitMath(els)
	els = cleanupMentionPrefix(els)
	return els
}

func mergeAndSplitMath(els []*larkdocx.TextElement) []*larkdocx.TextElement {
	var merged bytes.Buffer
	hasPlain := false
	for _, el := range els {
		if el.TextRun != nil && el.TextRun.Content != nil && el.TextRun.TextElementStyle == nil {
			merged.WriteString(*el.TextRun.Content)
			hasPlain = true
		}
	}
	if !hasPlain || !strings.Contains(merged.String(), "$") {
		return els
	}
	full := merged.String()
	if !inlineMathRe.MatchString(full) {
		return els
	}
	var out []*larkdocx.TextElement
	plainBuf := &bytes.Buffer{}
	for _, el := range els {
		if el.TextRun != nil && el.TextRun.Content != nil && el.TextRun.TextElementStyle == nil {
			plainBuf.WriteString(*el.TextRun.Content)
		} else {
			if plainBuf.Len() > 0 {
				out = append(out, splitInlineMath(plainBuf.String(), inlineStyle{})...)
				plainBuf.Reset()
			}
			out = append(out, el)
		}
	}
	if plainBuf.Len() > 0 {
		out = append(out, splitInlineMath(plainBuf.String(), inlineStyle{})...)
	}
	return out
}

func cleanupMentionPrefix(els []*larkdocx.TextElement) []*larkdocx.TextElement {
	for i := 1; i < len(els); i++ {
		if els[i].MentionUser == nil {
			continue
		}
		prev := els[i-1]
		if prev.TextRun == nil || prev.TextRun.Content == nil {
			continue
		}
		content := *prev.TextRun.Content
		if idx := strings.LastIndex(content, "@"); idx >= 0 {
			trimmed := content[:idx]
			if trimmed == "" {
				els = append(els[:i-1], els[i:]...)
				i--
			} else {
				prev.TextRun.Content = &trimmed
			}
		}
	}
	return els
}

func unescapeMd(s string) string {
	var buf strings.Builder
	buf.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			next := s[i+1]
			if (next >= '!' && next <= '/') || (next >= ':' && next <= '@') ||
				(next >= '[' && next <= '`') || (next >= '{' && next <= '~') {
				buf.WriteByte(next)
				i++
				continue
			}
		}
		buf.WriteByte(s[i])
	}
	return buf.String()
}

func extractInlineNode(node ast.Node, source []byte, s inlineStyle) []*larkdocx.TextElement {
	switch n := node.(type) {
	case *ast.Text:
		content := unescapeMd(string(n.Segment.Value(source)))
		if n.SoftLineBreak() || n.HardLineBreak() {
			content += "\n"
		}
		return []*larkdocx.TextElement{makeTextElement(content, s)}
	case *ast.String:
		return []*larkdocx.TextElement{makeTextElement(string(n.Value), s)}
	case *ast.Emphasis:
		ns := s
		if n.Level == 1 {
			ns.italic = true
		} else {
			ns.bold = true
		}
		return extractInline(n, source, ns)
	case *east.Strikethrough:
		ns := s
		ns.strikethrough = true
		return extractInline(n, source, ns)
	case *ast.CodeSpan:
		ns := s
		ns.inlineCode = true
		var buf bytes.Buffer
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*ast.Text); ok {
				buf.Write(t.Segment.Value(source))
			}
		}
		return []*larkdocx.TextElement{makeTextElement(buf.String(), ns)}
	case *ast.Link:
		ns := s
		ns.linkURL = string(n.Destination)
		return extractInline(n, source, ns)
	case *ast.AutoLink:
		ns := s
		ns.linkURL = string(n.URL(source))
		return []*larkdocx.TextElement{makeTextElement(string(n.Label(source)), ns)}
	case *east.TaskCheckBox:
		return nil
	case *ast.Image:
		alt := string(n.Text(source))
		if alt == "" {
			alt = "图片"
		}
		return []*larkdocx.TextElement{makeTextElement("[图片: "+alt+"]", s)}
	case *ast.RawHTML:
		var buf bytes.Buffer
		for i := 0; i < n.Segments.Len(); i++ {
			seg := n.Segments.At(i)
			buf.Write(seg.Value(source))
		}
		raw := strings.TrimSpace(buf.String())
		if m := uidCommentRe.FindStringSubmatch(raw); m != nil {
			return []*larkdocx.TextElement{
				larkdocx.NewTextElementBuilder().
					MentionUser(larkdocx.NewMentionUserBuilder().UserId(m[1]).Build()).
					Build(),
			}
		}
		return []*larkdocx.TextElement{makeTextElement(buf.String(), s)}
	default:
		return extractInline(node, source, s)
	}
}

func splitInlineMath(content string, s inlineStyle) []*larkdocx.TextElement {
	matches := inlineMathRe.FindAllStringIndex(content, -1)
	if len(matches) == 0 {
		return []*larkdocx.TextElement{makeTextElement(content, s)}
	}
	var els []*larkdocx.TextElement
	pos := 0
	for _, loc := range matches {
		if loc[0] > pos {
			els = append(els, makeTextElement(content[pos:loc[0]], s))
		}
		formula := content[loc[0]+1 : loc[1]-1]
		els = append(els, larkdocx.NewTextElementBuilder().
			Equation(larkdocx.NewEquationBuilder().Content(formula).Build()).
			Build())
		pos = loc[1]
	}
	if pos < len(content) {
		els = append(els, makeTextElement(content[pos:], s))
	}
	return els
}

func makeTextElement(content string, s inlineStyle) *larkdocx.TextElement {
	rb := larkdocx.NewTextRunBuilder().Content(content)
	if s.bold || s.italic || s.strikethrough || s.inlineCode || s.linkURL != "" {
		sb := larkdocx.NewTextElementStyleBuilder()
		if s.bold {
			sb.Bold(true)
		}
		if s.italic {
			sb.Italic(true)
		}
		if s.strikethrough {
			sb.Strikethrough(true)
		}
		if s.inlineCode {
			sb.InlineCode(true)
		}
		if s.linkURL != "" {
			sb.Link(larkdocx.NewLinkBuilder().Url(s.linkURL).Build())
		}
		rb.TextElementStyle(sb.Build())
	}
	return larkdocx.NewTextElementBuilder().TextRun(rb.Build()).Build()
}

var langMap = map[string]int{
	"plaintext": 1, "bash": 7, "sh": 60, "shell": 60, "zsh": 60,
	"c": 10, "cpp": 9, "c++": 9, "csharp": 8, "c#": 8, "cs": 8,
	"css": 12, "dart": 15, "dockerfile": 18, "docker": 18,
	"go": 22, "golang": 22, "groovy": 23, "html": 24, "http": 26,
	"haskell": 27, "json": 28, "java": 29,
	"javascript": 30, "js": 30, "julia": 31,
	"kotlin": 32, "kt": 32, "latex": 33, "tex": 33,
	"lua": 36, "makefile": 38, "make": 38, "markdown": 39, "md": 39,
	"nginx": 40, "objective-c": 41, "objc": 41, "php": 43, "perl": 44,
	"powershell": 46, "ps1": 46, "protobuf": 48, "proto": 48,
	"python": 49, "py": 49, "r": 50, "ruby": 52, "rb": 52,
	"rust": 53, "rs": 53, "scss": 55, "sql": 56, "scala": 57,
	"swift": 61, "typescript": 63, "ts": 63, "tsx": 63,
	"xml": 66, "yaml": 67, "yml": 67, "toml": 75,
	"cmake": 68, "diff": 69, "graphql": 71, "gql": 71,
	"properties": 73, "ini": 73, "solidity": 74, "sol": 74,
}

func convertTable(n *east.Table, source []byte) []Block {
	var rows [][]ast.Node

	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		var row []ast.Node
		for cell := child.FirstChild(); cell != nil; cell = cell.NextSibling() {
			row = append(row, cell)
		}
		if len(row) > 0 {
			rows = append(rows, row)
		}
	}

	if len(rows) == 0 {
		return nil
	}

	numRows := len(rows)
	numCols := len(rows[0])

	var cellBlocks []*larkdocx.Block
	for _, row := range rows {
		for c := 0; c < numCols; c++ {
			var els []*larkdocx.TextElement
			if c < len(row) {
				els = extractInline(row[c], source, inlineStyle{})
			}
			if len(els) == 0 {
				els = []*larkdocx.TextElement{makeTextElement("", inlineStyle{})}
			}
			txt := larkdocx.NewTextBuilder().Elements(els).Build()
			cellBlocks = append(cellBlocks, larkdocx.NewBlockBuilder().BlockType(2).Text(txt).Build())
		}
	}

	tableBlock := larkdocx.NewBlockBuilder().
		BlockType(31).
		Table(larkdocx.NewTableBuilder().
			Property(larkdocx.NewTablePropertyBuilder().
				RowSize(numRows).
				ColumnSize(numCols).
				HeaderRow(true).
				Build()).
			Build()).
		Build()

	return []Block{{
		Raw:       tableBlock,
		TableData: &TableData{Rows: numRows, Cols: numCols, CellBlocks: cellBlocks},
	}}
}

func mapLanguage(lang string) int {
	if code, ok := langMap[strings.ToLower(strings.TrimSpace(lang))]; ok {
		return code
	}
	return 1
}
