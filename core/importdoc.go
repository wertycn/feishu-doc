package core

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/wertycn/feishu-doc/core/markdown"

	larkdocx "github.com/larksuite/oapi-sdk-go/v3/service/docx/v1"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
)

const (
	rateLimitCode   = 99991400
	maxRetries      = 3
	cellUploadDelay = 200 * time.Millisecond
	downloadTimeout = 30 * time.Second
	maxImageSize    = 50 * 1024 * 1024
	maxRedirects    = 5
)

var imageHTTPClient = &http.Client{
	Timeout: downloadTimeout,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxRedirects {
			return fmt.Errorf("过多重定向 (超过 %d 次)", maxRedirects)
		}
		return nil
	},
	Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ip := net.ParseIP(host)
			if ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()) {
				return nil, fmt.Errorf("禁止访问内网地址: %s", host)
			}
			return (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, network, addr)
		},
	},
}

func (c *Client) ImportMarkdown(ctx context.Context, docID string, source []byte, mdFilePath string) error {
	blocks := markdown.Convert(source)
	if len(blocks) == 0 {
		return nil
	}
	return c.UploadBlocks(ctx, docID, blocks, mdFilePath)
}

// R1: --overwrite 全量覆盖模式
func (c *Client) OverwriteMarkdown(ctx context.Context, docID string, source []byte, mdFilePath string) error {
	blocks := markdown.Convert(source)

	existingBlocks, err := c.FetchAllBlocks(ctx, docID)
	if err != nil {
		return fmt.Errorf("获取现有块失败: %w", err)
	}

	var topLevelCount int
	for _, b := range existingBlocks {
		if Deref(b.ParentId) == docID && DerefInt(b.BlockType) != BlockTypePage {
			topLevelCount++
		}
	}

	if topLevelCount > 0 {
		fmt.Fprintf(c.Logger, "  删除 %d 个现有块...\n", topLevelCount)
		if err := c.DeleteBlockChildren(ctx, docID, 0, topLevelCount); err != nil {
			return fmt.Errorf("清空文档失败: %w", err)
		}
	}

	if len(blocks) == 0 {
		fmt.Fprintln(c.Logger, "  Markdown 内容为空，文档已清空")
		return nil
	}

	fmt.Fprintf(c.Logger, "  创建 %d 个新块...\n", len(blocks))
	return c.UploadBlocks(ctx, docID, blocks, mdFilePath)
}

func (c *Client) UploadBlocks(ctx context.Context, docID string, blocks []markdown.Block, mdFile string) error {
	total := len(blocks)
	parentAt := map[int]string{0: docID}

	for i := 0; i < total; {
		block := blocks[i]

		if block.ImageSrc != "" {
			err := c.UploadImageBlock(ctx, docID, block, mdFile)
			if err != nil {
				fmt.Fprintf(c.Logger, "  [warn] 图片失败: %v，插入占位文本\n", err)
				alt := block.ImageAlt
				if alt == "" {
					alt = block.ImageSrc
				}
				el := larkdocx.NewTextElementBuilder().
					TextRun(larkdocx.NewTextRunBuilder().Content("[图片: "+alt+"]").Build()).Build()
				fallback := larkdocx.NewBlockBuilder().BlockType(BlockTypeText).
					Text(larkdocx.NewTextBuilder().Elements([]*larkdocx.TextElement{el}).Build()).Build()
				req := larkdocx.NewCreateDocumentBlockChildrenReqBuilder().
					DocumentId(docID).BlockId(docID).
					Body(larkdocx.NewCreateDocumentBlockChildrenReqBodyBuilder().
						Children([]*larkdocx.Block{fallback}).Index(-1).Build()).
					Build()
				c.larkClient.Docx.DocumentBlockChildren.Create(ctx, req, c.RequestOpts()...)
			}
			fmt.Fprintf(c.Logger, "  已上传 %d/%d 块 (图片)\n", i+1, total)
			i++
			continue
		}

		if block.TableData != nil {
			if err := c.UploadTable(ctx, docID, block); err != nil {
				return fmt.Errorf("上传表格失败 (块 %d): %w", i+1, err)
			}
			fmt.Fprintf(c.Logger, "  已上传 %d/%d 块 (表格 %dx%d)\n", i+1, total, block.TableData.Rows, block.TableData.Cols)
			i++
			continue
		}

		depth := block.IndentLevel
		parentID := docID
		if depth > 0 {
			for d := depth - 1; d >= 0; d-- {
				if pid, ok := parentAt[d]; ok {
					parentID = pid
					break
				}
			}
		}

		j := i + 1
		if depth == 0 {
			for j < total && blocks[j].IndentLevel == 0 && blocks[j].TableData == nil && blocks[j].ImageSrc == "" {
				j++
			}
		}
		batch := blocks[i:j]

		var rawBlocks []*larkdocx.Block
		for _, b := range batch {
			rawBlocks = append(rawBlocks, b.Raw)
		}

		resp, err := c.createBlockWithRetry(ctx, docID, parentID, rawBlocks, -1)
		if err != nil {
			return fmt.Errorf("上传失败 (块 %d-%d): %w", i+1, j, err)
		}
		if !resp.Success() {
			return fmt.Errorf("上传失败 (块 %d-%d): code=%d, msg=%s", i+1, j, resp.Code, resp.Msg)
		}

		for k, child := range resp.Data.Children {
			idx := i + k
			if child.BlockId != nil && idx < len(blocks) {
				bd := blocks[idx].IndentLevel
				parentAt[bd] = *child.BlockId
			}
		}

		fmt.Fprintf(c.Logger, "  已上传 %d/%d 块\n", j, total)
		i = j
	}
	return nil
}

func (c *Client) UploadImageBlock(ctx context.Context, docID string, block markdown.Block, mdFile string) error {
	src := block.ImageSrc

	var imgPath string
	var fileName string
	var cleanup func()

	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		tmpFile, name, err := downloadToTemp(src)
		if err != nil {
			return fmt.Errorf("下载远程图片失败: %w", err)
		}
		imgPath = tmpFile
		fileName = name
		cleanup = func() { os.Remove(tmpFile) }
	} else {
		imgPath = src
		if !filepath.IsAbs(imgPath) {
			imgPath = filepath.Join(filepath.Dir(mdFile), imgPath)
		}
		fileName = filepath.Base(imgPath)
		cleanup = func() {}
	}
	defer cleanup()

	emptyImg := larkdocx.NewBlockBuilder().BlockType(BlockTypeImage).
		Image(larkdocx.NewImageBuilder().Build()).Build()

	createReq := larkdocx.NewCreateDocumentBlockChildrenReqBuilder().
		DocumentId(docID).
		BlockId(docID).
		Body(larkdocx.NewCreateDocumentBlockChildrenReqBodyBuilder().
			Children([]*larkdocx.Block{emptyImg}).
			Index(-1).
			Build()).
		Build()

	createResp, err := c.larkClient.Docx.DocumentBlockChildren.Create(ctx, createReq, c.RequestOpts()...)
	if err != nil {
		return fmt.Errorf("创建图片块失败: %w", err)
	}
	if !createResp.Success() {
		return fmt.Errorf("创建图片块失败: code=%d, msg=%s", createResp.Code, createResp.Msg)
	}
	blockID := *createResp.Data.Children[0].BlockId

	f, err := os.Open(imgPath)
	if err != nil {
		return fmt.Errorf("打开图片失败: %w", err)
	}
	defer f.Close()

	info, _ := f.Stat()

	uploadReq := larkdrive.NewUploadAllMediaReqBuilder().
		Body(larkdrive.NewUploadAllMediaReqBodyBuilder().
			FileName(fileName).
			ParentType(larkdrive.ParentTypeUploadAllMediaDocxImage).
			ParentNode(blockID).
			Size(int(info.Size())).
			File(f).
			Build()).
		Build()

	uploadResp, err := c.larkClient.Drive.Media.UploadAll(ctx, uploadReq, c.RequestOpts()...)
	if err != nil {
		return fmt.Errorf("上传素材失败: %w", err)
	}
	if !uploadResp.Success() {
		return fmt.Errorf("上传素材失败: code=%d, msg=%s", uploadResp.Code, uploadResp.Msg)
	}
	fileToken := *uploadResp.Data.FileToken

	patchReq := larkdocx.NewPatchDocumentBlockReqBuilder().
		DocumentId(docID).
		BlockId(blockID).
		DocumentRevisionId(-1).
		UpdateBlockRequest(larkdocx.NewUpdateBlockRequestBuilder().
			ReplaceImage(larkdocx.NewReplaceImageRequestBuilder().
				Token(fileToken).
				Build()).
			Build()).
		Build()

	patchResp, err := c.larkClient.Docx.DocumentBlock.Patch(ctx, patchReq, c.RequestOpts()...)
	if err != nil {
		return fmt.Errorf("关联图片失败: %w", err)
	}
	if !patchResp.Success() {
		return fmt.Errorf("关联图片失败: code=%d, msg=%s", patchResp.Code, patchResp.Msg)
	}

	fmt.Fprintf(c.Logger, "  图片已上传: %s → %s\n", fileName, fileToken)
	return nil
}

func (c *Client) createBlockWithRetry(ctx context.Context, docID, parentID string, children []*larkdocx.Block, index int) (*larkdocx.CreateDocumentBlockChildrenResp, error) {
	req := larkdocx.NewCreateDocumentBlockChildrenReqBuilder().
		DocumentId(docID).
		BlockId(parentID).
		Body(larkdocx.NewCreateDocumentBlockChildrenReqBodyBuilder().
			Children(children).
			Index(index).
			Build()).
		Build()

	var resp *larkdocx.CreateDocumentBlockChildrenResp
	var lastErr error
	for retry := 0; retry < maxRetries; retry++ {
		if retry > 0 {
			backoff := time.Duration(1<<uint(retry)) * 500 * time.Millisecond
			fmt.Fprintf(c.Logger, "  [retry] 限流，等待 %v 后重试...\n", backoff)
			time.Sleep(backoff)
		}
		resp, lastErr = c.larkClient.Docx.DocumentBlockChildren.Create(ctx, req, c.RequestOpts()...)
		if lastErr != nil {
			if strings.Contains(lastErr.Error(), "429") || strings.Contains(lastErr.Error(), "content-type not json") {
				continue
			}
			return nil, lastErr
		}
		if resp.Code == rateLimitCode {
			lastErr = fmt.Errorf("rate limited")
			continue
		}
		return resp, nil
	}
	return nil, lastErr
}

func (c *Client) UploadTable(ctx context.Context, docID string, block markdown.Block) error {
	td := block.TableData

	resp, err := c.createBlockWithRetry(ctx, docID, docID, []*larkdocx.Block{block.Raw}, -1)
	if err != nil {
		return fmt.Errorf("创建表格块失败: %w", err)
	}
	if !resp.Success() {
		return fmt.Errorf("创建表格块失败: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	if len(resp.Data.Children) == 0 || resp.Data.Children[0].Table == nil {
		return fmt.Errorf("表格创建响应中无 cell 数据")
	}
	cellIDs := resp.Data.Children[0].Table.Cells

	for idx, cellID := range cellIDs {
		if idx >= len(td.CellBlocks) {
			break
		}
		cellResp, cellErr := c.createBlockWithRetry(ctx, docID, cellID, []*larkdocx.Block{td.CellBlocks[idx]}, 0)
		if cellErr != nil {
			return fmt.Errorf("填充单元格失败 [%d]: %w", idx, cellErr)
		}
		if !cellResp.Success() {
			return fmt.Errorf("填充单元格失败 [%d]: code=%d, msg=%s", idx, cellResp.Code, cellResp.Msg)
		}
		time.Sleep(cellUploadDelay)
	}
	return nil
}

func downloadToTemp(rawURL string) (tmpPath, fileName string, err error) {
	resp, err := imageHTTPClient.Get(rawURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	fileName = path.Base(resp.Request.URL.Path)
	if fileName == "" || fileName == "/" || fileName == "." {
		fileName = "image.png"
	}

	tmpFile, err := os.CreateTemp("", "feishu-img-*-"+fileName)
	if err != nil {
		return "", "", err
	}
	defer tmpFile.Close()

	limitedBody := io.LimitReader(resp.Body, maxImageSize)
	if _, err := io.Copy(tmpFile, limitedBody); err != nil {
		os.Remove(tmpFile.Name())
		return "", "", err
	}

	return tmpFile.Name(), fileName, nil
}
