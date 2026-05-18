package core

import (
	"context"
	"fmt"

	larkwiki "github.com/larksuite/oapi-sdk-go/v3/service/wiki/v2"
)

func (c *Client) ListSpaces(ctx context.Context) ([]*larkwiki.Space, error) {
	var allSpaces []*larkwiki.Space
	pageToken := ""

	for {
		builder := larkwiki.NewListSpaceReqBuilder().PageSize(50)
		if pageToken != "" {
			builder = builder.PageToken(pageToken)
		}
		resp, err := c.larkClient.Wiki.Space.List(ctx, builder.Build(), c.RequestOpts()...)
		if err != nil {
			return nil, fmt.Errorf("请求失败: %w", err)
		}
		if !resp.Success() {
			return nil, fmt.Errorf("获取失败: code=%d, msg=%s", resp.Code, resp.Msg)
		}
		allSpaces = append(allSpaces, resp.Data.Items...)
		if !DerefBool(resp.Data.HasMore) {
			break
		}
		pageToken = Deref(resp.Data.PageToken)
	}
	return allSpaces, nil
}

func (c *Client) GetNodeInfo(ctx context.Context, token string) (*larkwiki.Node, error) {
	req := larkwiki.NewGetNodeSpaceReqBuilder().Token(token).Build()

	resp, err := c.larkClient.Wiki.Space.GetNode(ctx, req, c.RequestOpts()...)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	if !resp.Success() {
		return nil, fmt.Errorf("获取失败: code=%d, msg=%s", resp.Code, resp.Msg)
	}
	return resp.Data.Node, nil
}

func (c *Client) CreateWikiChild(ctx context.Context, spaceID, parentToken, title string) (*larkwiki.Node, error) {
	req := larkwiki.NewCreateSpaceNodeReqBuilder().
		SpaceId(spaceID).
		Node(larkwiki.NewNodeBuilder().
			ObjType("docx").
			ParentNodeToken(parentToken).
			NodeType("origin").
			Title(title).
			Build()).
		Build()

	resp, err := c.larkClient.Wiki.SpaceNode.Create(ctx, req, c.RequestOpts()...)
	if err != nil {
		return nil, fmt.Errorf("创建子文档失败: %w", err)
	}
	if !resp.Success() {
		return nil, fmt.Errorf("创建子文档失败: code=%d, msg=%s", resp.Code, resp.Msg)
	}
	return resp.Data.Node, nil
}

func (c *Client) ListChildren(ctx context.Context, spaceID, parentToken string) ([]*larkwiki.Node, error) {
	var all []*larkwiki.Node
	pageToken := ""

	for {
		builder := larkwiki.NewListSpaceNodeReqBuilder().
			SpaceId(spaceID).
			PageSize(50)
		if parentToken != "" {
			builder = builder.ParentNodeToken(parentToken)
		}
		if pageToken != "" {
			builder = builder.PageToken(pageToken)
		}

		resp, err := c.larkClient.Wiki.SpaceNode.List(ctx, builder.Build(), c.RequestOpts()...)
		if err != nil {
			return nil, fmt.Errorf("请求失败: %w", err)
		}
		if !resp.Success() {
			return nil, fmt.Errorf("获取节点失败: code=%d, msg=%s", resp.Code, resp.Msg)
		}
		all = append(all, resp.Data.Items...)
		if !DerefBool(resp.Data.HasMore) {
			break
		}
		pageToken = Deref(resp.Data.PageToken)
	}
	return all, nil
}
