package core

import (
	"context"
	"fmt"

	larkauthen "github.com/larksuite/oapi-sdk-go/v3/service/authen/v1"
)

type TokenResult struct {
	AccessToken      string
	RefreshToken     string
	ExpiresIn        int
	RefreshExpiresIn int
}

func (c *Client) ExchangeToken(ctx context.Context, code string) (*TokenResult, error) {
	req := larkauthen.NewCreateOidcAccessTokenReqBuilder().
		Body(larkauthen.NewCreateOidcAccessTokenReqBodyBuilder().
			GrantType("authorization_code").
			Code(code).
			Build()).
		Build()

	resp, err := c.larkClient.Authen.OidcAccessToken.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("token 交换失败: %w", err)
	}
	if !resp.Success() {
		return nil, fmt.Errorf("token 交换失败: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	return &TokenResult{
		AccessToken:      Deref(resp.Data.AccessToken),
		RefreshToken:     Deref(resp.Data.RefreshToken),
		ExpiresIn:        DerefInt(resp.Data.ExpiresIn),
		RefreshExpiresIn: DerefInt(resp.Data.RefreshExpiresIn),
	}, nil
}

func (c *Client) RefreshUserToken(ctx context.Context, refreshToken string) (*TokenResult, error) {
	req := larkauthen.NewCreateOidcRefreshAccessTokenReqBuilder().
		Body(larkauthen.NewCreateOidcRefreshAccessTokenReqBodyBuilder().
			GrantType("refresh_token").
			RefreshToken(refreshToken).
			Build()).
		Build()

	resp, err := c.larkClient.Authen.OidcRefreshAccessToken.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("刷新失败: %w", err)
	}
	if !resp.Success() {
		return nil, fmt.Errorf("刷新失败: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	return &TokenResult{
		AccessToken:      Deref(resp.Data.AccessToken),
		RefreshToken:     Deref(resp.Data.RefreshToken),
		ExpiresIn:        DerefInt(resp.Data.ExpiresIn),
		RefreshExpiresIn: DerefInt(resp.Data.RefreshExpiresIn),
	}, nil
}
