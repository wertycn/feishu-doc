package markdown

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type FrontMatter struct {
	DocID      string
	Revision   int
	Title      string
	SpaceID    string
	NodeToken  string
	ExportedAt time.Time
	Mode       string
}

func (fm *FrontMatter) Render() string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("feishu:\n")
	b.WriteString(fmt.Sprintf("  doc_id: %q\n", fm.DocID))
	if fm.Revision > 0 {
		b.WriteString(fmt.Sprintf("  revision: %d\n", fm.Revision))
	}
	b.WriteString(fmt.Sprintf("  title: %q\n", fm.Title))
	if fm.SpaceID != "" || fm.NodeToken != "" {
		b.WriteString("  wiki:\n")
		if fm.SpaceID != "" {
			b.WriteString(fmt.Sprintf("    space_id: %q\n", fm.SpaceID))
		}
		if fm.NodeToken != "" {
			b.WriteString(fmt.Sprintf("    node_token: %q\n", fm.NodeToken))
		}
	}
	b.WriteString(fmt.Sprintf("  exported_at: %q\n", fm.ExportedAt.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("  mode: %q\n", fm.Mode))
	b.WriteString("---\n\n")
	return b.String()
}

var (
	fmFieldRe = regexp.MustCompile(`^\s*(\w+):\s*"?([^"]*)"?\s*$`)
	fmIntRe   = regexp.MustCompile(`^\s*(\w+):\s*(\d+)\s*$`)
)

func ParseFrontMatter(source []byte) (*FrontMatter, []byte) {
	text := string(source)
	if !strings.HasPrefix(text, "---\n") {
		return nil, source
	}
	end := strings.Index(text[4:], "\n---")
	if end < 0 {
		return nil, source
	}
	end += 4
	fmContent := text[4:end]
	rest := text[end+4:]
	rest = strings.TrimLeft(rest, "\n")

	fm := &FrontMatter{}
	for _, line := range strings.Split(fmContent, "\n") {
		if m := fmIntRe.FindStringSubmatch(line); m != nil {
			switch m[1] {
			case "revision":
				fmt.Sscanf(m[2], "%d", &fm.Revision)
			}
			continue
		}
		if m := fmFieldRe.FindStringSubmatch(line); m != nil {
			switch m[1] {
			case "doc_id":
				fm.DocID = m[2]
			case "title":
				fm.Title = m[2]
			case "space_id":
				fm.SpaceID = m[2]
			case "node_token":
				fm.NodeToken = m[2]
			case "mode":
				fm.Mode = m[2]
			case "exported_at":
				if t, err := time.Parse(time.RFC3339, m[2]); err == nil {
					fm.ExportedAt = t
				}
			}
		}
	}

	if fm.DocID == "" {
		return nil, source
	}
	return fm, []byte(rest)
}
