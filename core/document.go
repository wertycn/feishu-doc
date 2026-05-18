package core

import (
	"context"
	"fmt"

	larkcontact "github.com/larksuite/oapi-sdk-go/v3/service/contact/v3"
	larkdocx "github.com/larksuite/oapi-sdk-go/v3/service/docx/v1"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	larkwiki "github.com/larksuite/oapi-sdk-go/v3/service/wiki/v2"
)

type DocumentInfo struct {
	DocumentID string
	Title      string
	RevisionID int
}

func (c *Client) CreateDocument(ctx context.Context, title, folderToken string) (*DocumentInfo, error) {
	bodyBuilder := larkdocx.NewCreateDocumentReqBodyBuilder().Title(title)
	if folderToken != "" {
		bodyBuilder = bodyBuilder.FolderToken(folderToken)
	}

	req := larkdocx.NewCreateDocumentReqBuilder().
		Body(bodyBuilder.Build()).
		Build()

	resp, err := c.larkClient.Docx.Document.Create(ctx, req, c.RequestOpts()...)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	if !resp.Success() {
		return nil, fmt.Errorf("创建失败: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	doc := resp.Data.Document
	return &DocumentInfo{
		DocumentID: Deref(doc.DocumentId),
		Title:      Deref(doc.Title),
		RevisionID: DerefInt(doc.RevisionId),
	}, nil
}

func (c *Client) GetDocument(ctx context.Context, docID string) (*DocumentInfo, error) {
	req := larkdocx.NewGetDocumentReqBuilder().DocumentId(docID).Build()

	resp, err := c.larkClient.Docx.Document.Get(ctx, req, c.RequestOpts()...)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	if !resp.Success() {
		return nil, fmt.Errorf("获取失败: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	doc := resp.Data.Document
	return &DocumentInfo{
		DocumentID: Deref(doc.DocumentId),
		Title:      Deref(doc.Title),
		RevisionID: DerefInt(doc.RevisionId),
	}, nil
}

func (c *Client) GetRawContent(ctx context.Context, docID string) (string, error) {
	req := larkdocx.NewRawContentDocumentReqBuilder().
		DocumentId(docID).
		Lang(0).
		Build()

	resp, err := c.larkClient.Docx.Document.RawContent(ctx, req, c.RequestOpts()...)
	if err != nil {
		return "", fmt.Errorf("获取内容失败: %w", err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("获取内容失败: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	return Deref(resp.Data.Content), nil
}

func (c *Client) FetchAllBlocks(ctx context.Context, docID string) ([]*larkdocx.Block, error) {
	var all []*larkdocx.Block
	pageToken := ""

	for {
		builder := larkdocx.NewListDocumentBlockReqBuilder().
			DocumentId(docID).
			PageSize(500)
		if pageToken != "" {
			builder = builder.PageToken(pageToken)
		}
		resp, err := c.larkClient.Docx.DocumentBlock.List(ctx, builder.Build(), c.RequestOpts()...)
		if err != nil {
			return nil, fmt.Errorf("获取块失败: %w", err)
		}
		if !resp.Success() {
			return nil, fmt.Errorf("获取块失败: code=%d, msg=%s", resp.Code, resp.Msg)
		}
		all = append(all, resp.Data.Items...)
		if !DerefBool(resp.Data.HasMore) {
			break
		}
		pageToken = Deref(resp.Data.PageToken)
	}
	return all, nil
}

func (c *Client) UpdateBlockTextElements(ctx context.Context, docID, blockID string, elements []*larkdocx.TextElement) error {
	req := larkdocx.NewPatchDocumentBlockReqBuilder().
		DocumentId(docID).
		BlockId(blockID).
		DocumentRevisionId(-1).
		UpdateBlockRequest(larkdocx.NewUpdateBlockRequestBuilder().
			UpdateTextElements(larkdocx.NewUpdateTextElementsRequestBuilder().
				Elements(elements).
				Build()).
			Build()).
		Build()

	resp, err := c.larkClient.Docx.DocumentBlock.Patch(ctx, req, c.RequestOpts()...)
	if err != nil {
		return fmt.Errorf("更新块失败: %w", err)
	}
	if !resp.Success() {
		return fmt.Errorf("更新块失败: code=%d, msg=%s", resp.Code, resp.Msg)
	}
	return nil
}

func (c *Client) DeleteBlockChildren(ctx context.Context, docID string, startIdx, endIdx int) error {
	req := larkdocx.NewBatchDeleteDocumentBlockChildrenReqBuilder().
		DocumentId(docID).
		BlockId(docID).
		Body(larkdocx.NewBatchDeleteDocumentBlockChildrenReqBodyBuilder().
			StartIndex(startIdx).
			EndIndex(endIdx).
			Build()).
		Build()

	resp, err := c.larkClient.Docx.DocumentBlockChildren.BatchDelete(ctx, req, c.RequestOpts()...)
	if err != nil {
		return fmt.Errorf("删除块失败: %w", err)
	}
	if !resp.Success() {
		return fmt.Errorf("删除块失败: code=%d, msg=%s", resp.Code, resp.Msg)
	}
	return nil
}

func (c *Client) GetUserName(ctx context.Context, userID string) string {
	req := larkcontact.NewGetUserReqBuilder().
		UserId(userID).
		UserIdType("open_id").
		Build()

	resp, err := c.larkClient.Contact.User.Get(ctx, req, c.RequestOpts()...)
	if err != nil || !resp.Success() || resp.Data == nil || resp.Data.User == nil {
		return ""
	}
	return Deref(resp.Data.User.Name)
}

func (c *Client) BatchGetUserNames(ctx context.Context, userIDs []string) map[string]string {
	result := make(map[string]string)
	for _, uid := range userIDs {
		if name := c.GetUserName(ctx, uid); name != "" {
			result[uid] = name
		}
	}
	return result
}

type CopyResult struct {
	Token   string
	Name    string
	Type    string
	URL     string
	OwnerID string
}

func (c *Client) CopyFile(ctx context.Context, fileToken, name, fileType, folderToken string) (*CopyResult, error) {
	bodyBuilder := larkdrive.NewCopyFileReqBodyBuilder().
		Name(name).
		Type(fileType)
	if folderToken != "" {
		bodyBuilder = bodyBuilder.FolderToken(folderToken)
	}

	req := larkdrive.NewCopyFileReqBuilder().
		FileToken(fileToken).
		Body(bodyBuilder.Build()).
		Build()

	resp, err := c.larkClient.Drive.File.Copy(ctx, req, c.RequestOpts()...)
	if err != nil {
		return nil, fmt.Errorf("复制文件失败: %w", err)
	}
	if !resp.Success() {
		return nil, fmt.Errorf("复制文件失败: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	f := resp.Data.File
	return &CopyResult{
		Token:   Deref(f.Token),
		Name:    Deref(f.Name),
		Type:    Deref(f.Type),
		URL:     Deref(f.Url),
		OwnerID: Deref(f.OwnerId),
	}, nil
}

func (c *Client) CopyWikiNode(ctx context.Context, spaceID, nodeToken, targetParentToken, targetSpaceID, title string) (*larkwiki.Node, error) {
	bodyBuilder := larkwiki.NewCopySpaceNodeReqBodyBuilder().
		TargetParentToken(targetParentToken)
	if targetSpaceID != "" {
		bodyBuilder = bodyBuilder.TargetSpaceId(targetSpaceID)
	} else {
		bodyBuilder = bodyBuilder.TargetSpaceId(spaceID)
	}
	if title != "" {
		bodyBuilder = bodyBuilder.Title(title)
	}

	req := larkwiki.NewCopySpaceNodeReqBuilder().
		SpaceId(spaceID).
		NodeToken(nodeToken).
		Body(bodyBuilder.Build()).
		Build()

	resp, err := c.larkClient.Wiki.SpaceNode.Copy(ctx, req, c.RequestOpts()...)
	if err != nil {
		return nil, fmt.Errorf("复制知识库节点失败: %w", err)
	}
	if !resp.Success() {
		return nil, fmt.Errorf("复制知识库节点失败: code=%d, msg=%s", resp.Code, resp.Msg)
	}
	return resp.Data.Node, nil
}

func (c *Client) AppendText(ctx context.Context, docID, content string) ([]string, error) {
	textRun := larkdocx.NewTextRunBuilder().Content(content).Build()
	textElement := larkdocx.NewTextElementBuilder().TextRun(textRun).Build()
	text := larkdocx.NewTextBuilder().Elements([]*larkdocx.TextElement{textElement}).Build()

	block := larkdocx.NewBlockBuilder().
		BlockType(2).
		Text(text).
		Build()

	req := larkdocx.NewCreateDocumentBlockChildrenReqBuilder().
		DocumentId(docID).
		BlockId(docID).
		Body(larkdocx.NewCreateDocumentBlockChildrenReqBodyBuilder().
			Children([]*larkdocx.Block{block}).
			Index(-1).
			Build()).
		Build()

	resp, err := c.larkClient.Docx.DocumentBlockChildren.Create(ctx, req, c.RequestOpts()...)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	if !resp.Success() {
		return nil, fmt.Errorf("追加失败: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	var ids []string
	for _, b := range resp.Data.Children {
		ids = append(ids, *b.BlockId)
	}
	return ids, nil
}
